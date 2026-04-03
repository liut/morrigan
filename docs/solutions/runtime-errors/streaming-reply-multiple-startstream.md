---
title: "Streaming回复中StartStream被多次调用导致重复消息"
category: runtime-errors
date: 2026-04-03
tags: [streaming, wecom, channel, bug-fix]
issues:
  - "用户在聊天界面看到多个重复的回复区域"
  - "正在思考... 提示出现多次"
---

# Streaming回复中StartStream被多次调用导致重复消息

## Problem

在 `handleStreamingReply` 中处理流式回复时，用户在聊天界面看到：
1. "正在思考..." 提示出现多次
2. 同样的回复内容出现在多个区域，几乎完全相同的回复不断出现

## Root Cause

`handleStreamingReply` 有一个循环来处理 tool calls（最多 5 次迭代），每次迭代调用 `doChannelStream`。原来的 `doChannelStream` **内部管理 StartStream**：

```go
// 原来的 doChannelStream（在循环中被多次调用）
func doChannelStream(...) {
    stream, _ := chh.llm.StreamChat(...)
    for result := range stream {
        if currentStreamID == "" {
            sr.StartStream(...)  // 每次迭代都创建新 stream！
        }
    }
}
```

问题在于：
- 迭代 1：`doChannelStream` → `StartStream("正在思考...")` → 返回 toolCalls → **stream 未 finish**
- 迭代 2：`doChannelStream` → `StartStream("正在思考...")` → **又一个新 stream**
- ...
- 只有最后一次迭代结束后才调用 `FinishStream`

导致多个 orphan stream 和重复消息。

## Solution

**将 stream 生命周期管理移到 `handleStreamingReply` 层面**，而不是 `doChannelStream` 内部。

### 核心改动

**`handleStreamingReply`** 管理完整 stream 生命周期：
```go
func (chh *channelHandler) handleStreamingReply(...) {
    for {
        // StartStream 只在首次迭代时调用
        if streamID == "" {
            streamID, err = sr.StartStream(ctx, msg.ReplyCtx, "正在思考...")
            if err != nil {
                channelReplyError(p, msg, "AI processing failed")
                return
            }
        }

        // doChannelStream 只负责 StreamChat + AppendStream
        answer, toolCalls, err := chh.doChannelStream(ctx, p, msg, sr, streamID, messages, tools)
        if err != nil {
            // StreamChat 失败：通过 FinishStream 发送用户友好错误
            finishErr := sr.FinishStream(ctx, msg.ReplyCtx, streamID, translateLLMErrorToUser(err))
            if finishErr != nil {
                channelReplyError(p, msg, "AI processing failed")
            }
            return
        }
        // ... tool call loop
    }

    // 最后调用一次 FinishStream
    sr.FinishStream(ctx, msg.ReplyCtx, streamID, fullAnswer)
}
```

**`doChannelStream`** 不再管理 Start/Finish：
```go
func (chh *channelHandler) doChannelStream(ctx, streamID, ...) (string, []llm.ToolCall, error) {
    stream, _ := chh.llm.StreamChat(...)
    for result := range stream {
        contentBuilder.WriteString(result.Delta)
        if result.Delta != "" {
            sr.AppendStream(ctx, msg.ReplyCtx, streamID, content)  // 只追加
        }
    }
    return contentBuilder.String(), toolCalls, nil
}
```

### 错误处理改进

`StreamChat` 失败时，错误通过 `FinishStream` 传回给用户（友好的中文消息），只有 `FinishStream` 也失败时才 fallback 到 `channelReplyError`：

```go
finishErr := sr.FinishStream(ctx, msg.ReplyCtx, streamID, translateLLMErrorToUser(err))
if finishErr != nil {
    channelReplyError(p, msg, "AI processing failed")  // 备用
}
```

## Error Translation

`translateLLMErrorToUser` 将 LLM 错误转换为用户友好的中文消息：

| 原始错误 | 用户消息 |
|---------|---------|
| `context deadline exceeded` | "请求超时，请稍后重试" |
| `rate limit exceeded` | "请求过于频繁，请稍后重试" |
| `model not found` | "AI 服务暂时不可用" |
| 其他 | "抱歉，发生了错误，请稍后重试" |

## Files Changed

- `pkg/web/api/handle_platform.go`

## Prevention

### Code Review Checklist for Streaming Code

- [ ] `StartStream` 和 `FinishStream` 由**同一层**调用，不是分散在多个函数
- [ ] `StartStream` 在循环**外部**，只调用一次
- [ ] `streamID` 作为参数传递，不在循环内重新创建
- [ ] 错误路径确保用户收到反馈（不能 silent 失败）

### Anti-Patterns to Avoid

```go
// BAD: StartStream 在循环内
for {
    if streamID == "" {
        sr.StartStream(...)  // 每次迭代可能创建新 stream
    }
}

// BAD: doChannelStream 既 StartStream 又 FinishStream
func doChannelStream(...) {
    sr.StartStream(...)  // 谁是 owner？
    // ...
    sr.FinishStream(...)  // caller 可能也会调用
}
```

### Correct Pattern

```
handleStreamingReply  → StartStream (循环外) + FinishStream (循环后)
doChannelStream       → 只有 AppendStream
```

## Related

- `docs/plans/2026-04-03-001-feat-stream-start-earlier-plan.md` - 完整计划文档
- `todos/002-pending-p1-orphaned-stream-error.md` - 相关的孤儿 stream 问题
- `todos/008-pending-p3-reply-duplicates-streaming.md` - Reply 方法重复 streaming 逻辑
