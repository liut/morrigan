---
title: feat: WeCom WebSocket 流式消息支持
type: feat
status: superseded
date: 2026-04-02
superseded_by: docs/plans/2026-04-03-001-feat-stream-start-earlier-plan.md
---

# WeCom WebSocket 流式消息支持

## Overview

改造 WeCom WebSocket 的 `Reply` 方法，实现真正的流式消息：首次 `finish=false` 创建流，后续追加，最终 `finish=true` 结束。

## Problem Statement

当前 `WSChannel.Reply()` (`websocket.go:370`) 直接发送 `finish=true` 的流式消息，无法实现内容接续效果。

## Proposed Solution

### 1. WSChannel 添加三个方法

在 `pkg/services/channels/wecom/websocket.go` 中新增：

```go
// startStream 首次发送，finish=false
func (p *WSChannel) startStream(ctx context.Context, rctx wsReplyContext, content string) (string, error) {
    streamID := p.generateReqID("stream")
    frame := p.buildStreamFrame(rctx, streamID, content, false)
    if err := p.writeJSON(frame); err != nil {
        return "", err
    }
    return streamID, nil
}

// appendStream 后续发送，finish=false
func (p *WSChannel) appendStream(ctx context.Context, rctx wsReplyContext, streamID string, content string) error {
    frame := p.buildStreamFrame(rctx, streamID, content, false)
    return p.writeJSON(frame)
}

// finishStream 结束发送，finish=true
func (p *WSChannel) finishStream(ctx context.Context, rctx wsReplyContext, streamID string) error {
    frame := p.buildStreamFrame(rctx, streamID, "", true)
    return p.writeJSON(frame)
}

// wsStreamFrame 预定义流式消息帧结构
type wsStreamFrame struct {
    Cmd     string         `json:"cmd"`
    Headers wsFrameHeaders `json:"headers"`
    Body    wsStreamBody   `json:"body"`
}

type wsStreamBody struct {
    MsgType string        `json:"msgtype"`
    Stream  wsStreamContent `json:"stream"`
}

type wsStreamContent struct {
    ID      string `json:"id"`
    Finish  bool   `json:"finish"`
    Content string `json:"content"`
}

func (p *WSChannel) buildStreamFrame(rc wsReplyContext, streamID, content string, finish bool) wsStreamFrame {
    return wsStreamFrame{
        Cmd:     "aibot_respond_msg",
        Headers: wsFrameHeaders{ReqID: rc.reqID},
        Body: wsStreamBody{
            MsgType: "stream",
            Stream: wsStreamContent{
                ID:      streamID,
                Finish:  finish,
                Content: content,
            },
        },
    }
}
```

### 2. StreamReplier 接口（可选）

在 `pkg/models/channel/channel.go` 添加：

```go
// StreamReplier is an optional interface for channels that support streaming.
type StreamReplier interface {
    StartStream(ctx context.Context, replyCtx any, content string) (streamID string, err error)
    AppendStream(ctx context.Context, replyCtx any, streamID string, content string) error
    FinishStream(ctx context.Context, replyCtx any, streamID string) error
}
```

WSChannel 实现该接口。

### 3. MessageHandler 流式分支

在 `handle_platform.go` 中检测 `StreamReplier` 接口，走流式处理分支：

```go
// handleStreamingReply 处理 WeCom WebSocket 流式回复
func (chh *channelHandler) handleStreamingReply(p channel.Channel, msg *channel.Message) {
    ctx := context.Background()

    // 构建消息（复用现有逻辑）...
    messages, tools := chh.buildChatMessages(msg)

    // 流式调用 LLM
    stream, err := chh.llm.StreamChat(ctx, messages, tools)
    if err != nil {
        channelReplyError(p, msg, "AI processing failed")
        return
    }

    sr := p.(channel.StreamReplier)
    var streamID string
    var content strings.Builder // 本地累积完整内容

    for result := range stream {
        if result.Error != nil {
            slog.Warn("channel: stream error", "err", result.Error)
            break
        }

        content.WriteString(result.Delta) // 累积内容

        if streamID == "" {
            // 首次：startStream，发送累积内容
            sid, err := sr.StartStream(ctx, msg.ReplyCtx, content.String())
            if err != nil {
                slog.Error("channel: start stream failed", "err", err)
                channelReplyError(p, msg, "Failed to start streaming")
                return
            }
            streamID = sid
        } else {
            // 后续：appendStream，每次发送完整累积内容（覆盖更新）
            if err := sr.AppendStream(ctx, msg.ReplyCtx, streamID, content.String()); err != nil {
                slog.Warn("channel: append stream failed", "err", err)
            }
        }
    }

    // 结束流
    if streamID != "" {
        if err := sr.FinishStream(ctx, msg.ReplyCtx, streamID); err != nil {
            slog.Warn("channel: finish stream failed", "err", err)
        }
    }

    // 保存历史...
}
```

`MessageHandler` 中检测：

```go
func (chh *channelHandler) MessageHandler(p channel.Channel, msg *channel.Message) {
    // ... 命令检测等前置逻辑不变 ...

    if sr, ok := p.(channel.StreamReplier); ok {
        chh.handleStreamingReply(p, msg)
    } else {
        chh.handleRegularReply(p, msg)
    }
}
```

## File Changes

| File | Change |
|------|--------|
| `pkg/models/channel/channel.go` | 添加 `StreamReplier` 接口 |
| `pkg/services/channels/wecom/websocket.go` | 实现 `StreamReplier` 三个方法 |
| `pkg/web/api/handle_platform.go` | `MessageHandler` 增加流式分支检测 |

## Acceptance Criteria

- [ ] WeCom WebSocket 首次发送 `finish=false`
- [ ] 后续增量追加
- [ ] 最终发送 `finish=true`
- [ ] HTTP 模式和其他通道不受影响

## ⚠️ 待验证

- [x] ~~**WeCom 流式接续语义**~~ 官方文档明确：继续使用相同 `stream.id` 推送会**覆盖**消息内容，需本地累积完整内容后发送。设计已按此思路实现，验证不符时再修订。
