---
title: Anthropic/DeepSeek Thinking Mode reasoning_content 回传修复
type: fix
status: completed
date: 2026-04-25
---

# Anthropic/DeepSeek Thinking Mode reasoning_content 回传修复

## 问题分析

### DeepSeek Thinking Mode 规则

| 场景 | reasoning_content 处理 |
|------|----------------------|
| 两 user 之间，**无工具调用** | 无需回传 |
| 两 user 之间，**有工具调用** | **必须回传**，否则报 400 错误 |

### 问题链路

```
doChatStream() → result.Think 只发到 SSE，没有收集
                    ↓
chatResponse{answer, toolCalls, ...} ← 没有 think 字段
                    ↓
doExecuteToolCalls() → 下一轮消息没有 reasoning_content
                    ↓
DeepSeek API 报错: content[].thinking must be passed back
```

## 已完成的修复

### ✅ Step 1: `convo_basic.go` - chatResponse 添加 think 字段

```go
type chatResponse struct {
    answer    string
    toolCalls []llm.ToolCall
    usage     *llm.Usage
    finish    llm.FinishReason
    think     string // 新增：reasoning_content

    model    string
    llmResID string
}
```

### ✅ Step 2: `handle_convo.go` - doChatStream 收集 result.Think

在流式循环中累积 think：

```go
// doChatStream 中
for result := range stream {
    // ...
    res.answer += result.Delta
    res.think += result.Think  // 新增
    // ...
}
```

### ✅ Step 3: `handle_convo.go` - doExecuteToolCalls 接收并回传 thinking

```go
func (a *api) doExecuteToolCalls(ctx context.Context, toolCalls []llm.ToolCall, messages []llm.Message, think string) ([]llm.Message, bool) {
    // ...
    messages = append(messages, llm.Message{
        Role:      llm.RoleAssistant,
        Thinking:  think,  // 回传 reasoning_content
        ToolCalls: toolCalls,
    })
    // ...
}
```

### ✅ Step 4: `types.go` - Message 已包含 Thinking 字段

```go
type Message struct {
    Role       string
    Content    string
    Thinking   string     `json:"thinking,omitempty"` // 新增
    ToolCalls  []ToolCall
    ToolCallID string
}
```

### ✅ Step 5: `anthropic.go` - toAnthropicInputParts 处理 thinking

```go
func toAnthropicInputParts(m Message) []anthropicContentPart {
    if strings.TrimSpace(m.Content) == "" && strings.TrimSpace(m.Thinking) == "" {
        return nil
    }
    var parts []anthropicContentPart
    if strings.TrimSpace(m.Content) != "" {
        parts = append(parts, anthropicContentPart{Type: "text", Text: m.Content})
    }
    if strings.TrimSpace(m.Thinking) != "" {
        parts = append(parts, anthropicContentPart{Type: "thinking", Thinking: m.Thinking})
    }
    return parts
}
```

## 实际发现的问题（深入调试后）

### Bug 1: thinking_delta 伴随 input_json_delta

DeepSeek 在 thinking 模式时，`thinking_delta` 和 `input_json_delta` 会交替出现，但 `input_json_delta` 片段中可能包含 `"thinking` 前缀的 JSON，导致 tool_use 参数被污染。

**修复**：在 `input_json_delta` 处理时跳过以 `"thinking` 开头的片段：

```go
if !strings.HasPrefix(strings.TrimSpace(event.Delta.PartialJSON), "\"thinking") {
    currentToolCalls[lastIdx].Function.Arguments = append(...)
}
```

### Bug 2: toAnthropicMessages 的 pendingToolResults 延迟导致 tool_use_id 错位

**问题根因**：之前的实现使用 `pendingToolResults` 缓存 tool_result，并在 `flushToolResults` 时一次性追加到上一条 user 消息。但在多轮工具调用时，tool_use_id 被覆盖导致引用错位。

**错误现象**：
- 第一次工具调用：tool_use_id 匹配 ✅
- 第二次工具调用：tool_use_id 变成下一轮的 ID ❌

**修复**：重构 `toAnthropicMessages`：

1. **删除 `pendingToolResults` 和 `flushToolResults`** - 延迟 flush 导致 ID 引用错位
2. **Tool result 立即合并到前一个 user message** - 如果前一条是 user，就追加；否则创建新的 user message
3. **User 消息区分处理** - 有 `ToolCallID` 的 user 消息作为 tool_result 处理

```go
case "tool":
    toolResultBlock := anthropicContentPart{
        Type:      "tool_result",
        ToolUseID: m.ToolCallID,
        Content:   m.Content,
    }
    if len(out) > 0 {
        last := &out[len(out)-1]
        if last.Role == "user" {
            last.Content = append(last.Content, toolResultBlock)
            continue
        }
    }
    out = append(out, anthropicMsg{
        Role:    "user",
        Content: []anthropicContentPart{toolResultBlock},
    })
```

## 修改的文件

| 文件 | 修改内容 |
|------|---------|
| `pkg/web/api/convo_basic.go` | chatResponse 添加 think 字段 |
| `pkg/web/api/handle_convo.go` | doChatStream 收集 result.Think；doExecuteToolCalls 接收并回传 think |
| `pkg/services/llm/types.go` | Message 已包含 Thinking 字段 ✅ |
| `pkg/services/llm/anthropic.go` | toAnthropicInputParts 处理 thinking；input_json_delta 跳过 thinking 前缀；重构 toAnthropicMessages |

## 参考

- DeepSeek Thinking Mode: https://api-docs.deepseek.com/zh-cn/guides/thinking_mode
- 错误信息：`The content[].thinking must be passed back to the API.`
- 错误信息：`unexpected messages.N.content.0: tool_use_id found in tool_result blocks`
