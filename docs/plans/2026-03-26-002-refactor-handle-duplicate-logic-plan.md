---
title: "refactor: 抽取 handle_convo 与 handle_platform 重复逻辑"
type: refactor
status: completed
date: 2026-03-26
---

# refactor: 抽取 handle_convo 与 handle_platform 重复逻辑

## Overview

`handle_convo.go` 和 `handle_platform.go` 中存在重复的工具调用循环逻辑，需要抽取为共享代码，消除代码冗余并统一日志规范。

## Problem Statement

### 重复的 `executeToolCallLoop`

两个文件都有独立的 `executeToolCallLoop` 实现：

| 文件 | 行号 | 日志方式 | 日志详细程度 |
|------|------|----------|-------------|
| `handle_platform.go` | 183-232 | `slog.Warn` | 最小化 |
| `handle_convo.go` | 751-805 | `logger().Infow` | 详细 |

**`handle_platform.go` 版本特点**：
- 使用 `slog.Warn` 记录错误
- 仅在解析参数失败和调用工具失败时记录日志
- 无成功调用的日志

**`handle_convo.go` 版本特点**：
- 使用 `logger().Infow`（项目规范的日志方式）
- 详细记录 `toolCallID`、`toolCallType`、`toolCallName`
- 成功调用时记录 `invokeTool ok` 及返回内容

### 其他共享元素（无需修改）

| 元素 | 位置 | 说明 |
|------|------|------|
| `formatToolResult` | handle_convo.go:685-718 | 同一包内，可直接调用 |
| `chatExecutor` 类型 | handle_convo.go:743-744 | 已共享 |
| `convertToolCallsForJSON` | handle_convo.go:720-741 | 仅 convo 使用 |

## Proposed Solution

### 方案：提取共享的 `ToolExecutor`

创建 `pkg/web/api/tool_executor.go`，包含：

1. **`ToolExecutor` 结构体**：封装通用的工具调用循环逻辑
2. **`ExecuteToolCallLoop` 方法**：处理工具调用循环直到无 tool calls
3. **`chatExecutor` 类型**：作为函数参数传入，支持流式/非流式执行器

### 架构设计

```
pkg/web/api/
├── handle_convo.go          # api.executeToolCallLoop → 使用 ToolExecutor
├── handle_platform.go       # channelHandler.executeToolCallLoop → 使用 ToolExecutor
├── tool_executor.go         # NEW: 共享的 ToolExecutor
```

### 核心实现

```go
// tool_executor.go

// ToolExecutor 封装工具调用循环逻辑
type ToolExecutor struct {
    toolreg *tools.Registry
}

// NewToolExecutor 创建 ToolExecutor
func NewToolExecutor(toolreg *tools.Registry) *ToolExecutor {
    return &ToolExecutor{toolreg: toolreg}
}

// ExecuteToolCallLoop 执行工具调用循环，直到无 tool calls
func (e *ToolExecutor) ExecuteToolCallLoop(
    ctx context.Context,
    messages []llm.Message,
    tools []llm.ToolDefinition,
    exec chatExecutor,
) (string, []llm.ToolCall, *llm.Usage, error) {
    for {
        answer, toolCalls, usage, err := exec(ctx, messages, tools)
        if err != nil {
            return "", nil, nil, err
        }

        if len(toolCalls) == 0 {
            return answer, nil, usage, nil
        }

        // 添加 assistant 消息（带 tool calls）
        messages = append(messages, llm.Message{
            Role:      llm.RoleAssistant,
            ToolCalls: toolCalls,
        })

        // 执行工具调用
        for _, tc := range toolCalls {
            if tc.Type != "function" {
                continue
            }

            var parameters map[string]any
            args := string(tc.Function.Arguments)
            if args != "" && args != "{}" {
                if err := json.Unmarshal([]byte(args), &parameters); err != nil {
                    logger().Infow("chat", "toolCallID", tc.ID, "args", args, "err", err)
                    continue
                }
            }
            if parameters == nil {
                parameters = make(map[string]any)
            }

            content, err := e.toolreg.Invoke(ctx, tc.Function.Name, parameters)
            if err != nil {
                logger().Infow("invokeTool fail", "toolCallName", tc.Function.Name, "err", err)
                continue
            }

            logger().Infow("invokeTool ok", "toolCallName", tc.Function.Name,
                "content", toolsvc.ResultLogs(content))
            messages = append(messages, llm.Message{
                Role:       llm.RoleTool,
                Content:    formatToolResult(content),
                ToolCallID: tc.ID,
            })
        }
    }
}
```

### 修改点

#### 1. 创建 `tool_executor.go`

新文件，包含：
- `ToolExecutor` 结构体
- `NewToolExecutor` 构造函数
- `ExecuteToolCallLoop` 方法
- 从 `handle_convo.go` 移动 `chatExecutor` 类型定义

#### 2. 修改 `handle_convo.go`

```go
// 添加字段
type api struct {
    // ... existing fields ...
    toolExec *ToolExecutor
}

// 在 newapi() 中初始化
a.toolExec = NewToolExecutor(toolreg)

// 修改 executeToolCallLoop 为委托调用
func (a *api) executeToolCallLoop(ctx context.Context, messages []llm.Message, tools []llm.ToolDefinition, exec chatExecutor) (string, []llm.ToolCall, *llm.Usage, error) {
    return a.toolExec.ExecuteToolCallLoop(ctx, messages, tools, exec)
}
```

#### 3. 修改 `handle_platform.go`

```go
// channelHandler 添加 toolExec 字段
type channelHandler struct {
    toolExec *ToolExecutor
    // ... existing fields ...
}

// 修改 InitChannels 接收 toolExec
func InitChannels(r chi.Router, preset *aigc.Preset, sto stores.Storage, llmClient llm.Client, toolreg *tools.Registry) error {
    phandler = &channelHandler{
        toolExec: NewToolExecutor(toolreg),
        // ... existing fields ...
    }
    // ...
}

// 修改 executeToolCallLoop 为委托调用
func (phandler *channelHandler) executeToolCallLoop(ctx context.Context, messages []llm.Message, tools []llm.ToolDefinition, exec chatExecutor) (string, []llm.ToolCall, *llm.Usage, error) {
    return phandler.toolExec.ExecuteToolCallLoop(ctx, messages, tools, exec)
}
```

## Technical Considerations

### 日志标准化

统一使用 `logger().Infow` 而非 `slog.Warn`，符合 AGENTS.md 中的日志规范：
```go
// ✅ 规范写法
logger().Infow("invokeTool fail", "toolCallName", tc.Function.Name, "err", err)

// ❌ 避免
slog.Warn("platform: invoke tool failed", "tool", tc.Function.Name, "err", err)
```

### 错误处理策略

两个原实现都使用 `continue` 处理单个工具调用失败，不阻塞循环。保持此行为。

### 依赖注入

`ToolExecutor` 通过 `*tools.Registry` 初始化，遵循现有依赖注入模式。

## Files to Change

| 文件 | 变更 |
|------|------|
| `pkg/web/api/tool_executor.go` | 新增 - 共享的 ToolExecutor |
| `pkg/web/api/handle_convo.go` | 使用 ToolExecutor，移除重复的 executeToolCallLoop |
| `pkg/web/api/handle_platform.go` | 使用 ToolExecutor，移除重复的 executeToolCallLoop |
| `pkg/web/api/api.go` | 初始化 api.toolExec 字段 |

## Acceptance Criteria

- [ ] `executeToolCallLoop` 仅在 `tool_executor.go` 中有一个实现
- [ ] `handle_convo.go` 和 `handle_platform.go` 都使用共享的 `ToolExecutor`
- [ ] 日志统一使用 `logger().Infow`（符合项目规范）
- [ ] 工具调用行为保持一致（continue on single failure）
- [ ] `make vet lint` 通过
- [ ] 单元测试（如有）仍然通过

## Verification

```bash
# 验证 vet 和 lint
make vet lint

# 验证构建
go build ./...

# 手动测试：发送聊天消息，验证工具调用正常
```
