---
title: feat: 将 StartStream 提前到 StreamChat 调用前
type: feat
status: completed
date: 2026-04-03
---

# 将 StartStream 提前到 StreamChat 调用前

## Overview

在 `doChannelStream` 中，将 `StartStream` 调用从 LLM 首次返回内容时提前到 `StreamChat` 调用之前，让平台用户更早看到"正在回复"的提示，改善用户体验。

## Problem Statement

当前 `doChannelStream` 的执行顺序：

1. 调用 `chh.llm.StreamChat()` 建立流
2. 遍历 stream，直到收到非空 `Delta` 时才调用 `StartStream`
3. 用户在 LLM 开始响应前看不到任何反馈

问题：
- LLM 响应可能有延迟（网络、模型加载等）
- 用户不知道请求是否被接受
- 若 `StreamChat` 失败，只能通过 `channelReplyError` 回退，但此时 stream 未启动

## Proposed Solution

### 核心改动

```go
// doChannelStream 修改后的伪代码
func (chh *channelHandler) doChannelStream(...) (string, []llm.ToolCall, string, error) {
    // 1. 立即 StartStream，通知平台开始处理
    streamID, err := sr.StartStream(ctx, msg.ReplyCtx, "正在思考...")
    if err != nil {
        slog.Error("channel: start stream failed", "err", err)
        return "", nil, "", err
    }

    // 2. 调用 StreamChat
    stream, err := chh.llm.StreamChat(ctx, messages, tools)
    if err != nil {
        // 3a. StreamChat 失败：FinishStream 并传入错误提示
        slog.Error("channel: stream chat failed", "error", err)
        finishErr := sr.FinishStream(ctx, msg.ReplyCtx, streamID, translateError(err))
        if finishErr != nil {
            slog.Warn("channel: finish stream after error failed", "err", finishErr)
        }
        return "", nil, streamID, err
    }

    // 4. 正常流程：遍历 stream，AppendStream
    // ...
}
```

### 错误消息转换

`StreamChat` 错误需要转换为对用户友好的自然语言消息：

| 原始错误 | 用户看到的消息 |
|---------|--------------|
| `context deadline exceeded` | "请求超时，请稍后重试" |
| `rate limit exceeded` | "请求过于频繁，请稍后重试" |
| `model not found` | "AI 服务暂时不可用" |
| 其他错误 | "抱歉，发生了错误，请稍后重试" |

## Technical Considerations

### 1. StreamReplier 接口无需修改

现有接口已支持此模式：
- `StartStream(ctx, replyCtx, content)` - content 可传入初始提示如"正在思考..."
- `FinishStream(ctx, replyCtx, streamID, finalContent)` - finalContent 传入错误消息

### 2. 与 AbortStream 的交互

参见 `todos/002-pending-p1-orphaned-stream-error.md`，当前 `StreamReplier` 缺少 `AbortStream`。若后续添加该方法：

- `StartStream` 成功后的 `StreamChat` 失败：应调用 `AbortStream` 而非 `FinishStream`（或 `FinishStream` 传空 content）
- 当前实现先用 `FinishStream` 传递错误消息是合理的 workaround

### 3. 并发安全

`StartStream` 在 stream 遍历循环外部调用，唯一的竞争是 `streamID` 的赋值，无并发问题。

### 4. 平台兼容性

部分平台可能不支持流式消息的中间状态更新（只有最终内容）。`StartStream` 时传入的"正在思考..."可能不会被显示，或被后续 `AppendStream` 覆盖。这是可接受的行为退化。

## System-Wide Impact

### Interaction Graph

```
User Message → MessageHandler
  → handleStreamingReply (若支持 StreamReplier)
    → StartStream ("正在思考...") ← 在循环外，仅首次调用
    → doChannelStream (StreamChat + AppendStream)
      → 若 StreamChat 失败: FinishStream + channelReplyError 备用
    → [如有 tool calls，循环执行 tools 后再次 doChannelStream]
    → FinishStream
  → 保存历史
```

### Error Propagation

| 错误发生点 | 行为 |
|----------|------|
| StartStream 失败 | 直接返回 error，调用 channelReplyError |
| StreamChat 失败 | FinishStream 传错误消息；若 FinishStream 也失败则 channelReplyError 备用 |
| AppendStream 失败 | log warning，继续处理 |
| FinishStream 失败 | log warning |

## Acceptance Criteria

- [ ] `StartStream` 在 `StreamChat` 之前调用
- [ ] `StreamChat` 失败时，通过 `FinishStream` 传回自然语言错误消息
- [ ] 正常流程不受影响（仍正常追加和结束流）
- [ ] 错误消息使用友好的用户语言

## File Changes

| File | Change |
|------|--------|
| `pkg/web/api/handle_platform.go` | `doChannelStream` 重构，将 `StartStream` 提前 |

## Implementation Notes

### 实际架构

Stream 生命周期由 `handleStreamingReply` 管理，`doChannelStream` 只负责单次 StreamChat 调用。

**handleStreamingReply 中的改动：**
```go
for {
    // StartStream 只在首次迭代时调用
    if streamID == "" {
        streamID, err = sr.StartStream(ctx, msg.ReplyCtx, "正在思考...")
        if err != nil {
            slog.Error("channel: start stream failed", "err", err)
            channelReplyError(p, msg, "AI processing failed")
            return
        }
    }

    // doChannelStream 只做 StreamChat + AppendStream
    answer, toolCalls, err := chh.doChannelStream(ctx, p, msg, sr, streamID, messages, tools)
    if err != nil {
        slog.Error("channel: stream round failed", "iter", iter, "err", err)
        finishErr := sr.FinishStream(ctx, msg.ReplyCtx, streamID, translateLLMErrorToUser(err))
        if finishErr != nil {
            slog.Warn("channel: finish stream after error failed, falling back to Reply", "err", finishErr)
            channelReplyError(p, msg, "AI processing failed")
        }
        return
    }
    // ... tool call loop handling
}
```

**doChannelStream 改动：**
- 签名变化：增加 `streamID string` 参数
- 返回值变化：移除 `streamID`（由调用方管理）
- 不再调用 StartStream，只调用 AppendStream

添加辅助函数：

```go
// translateLLMErrorToUser 将 LLM 错误转换为用户友好的消息
func translateLLMErrorToUser(err error) string {
    if err == nil {
        return ""
    }
    errStr := err.Error()
    switch {
    case strings.Contains(errStr, "context deadline exceeded"):
        return "请求超时，请稍后重试"
    case strings.Contains(errStr, "rate limit"):
        return "请求过于频繁，请稍后重试"
    case strings.Contains(errStr, "model not found"):
        return "AI 服务暂时不可用"
    default:
        return "抱歉，发生了错误，请稍后重试"
    }
}
```
