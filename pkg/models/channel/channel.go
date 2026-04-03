package channel

import (
	"context"
	"errors"
)

// ErrNotSupported indicates a channel doesn't support a particular operation.
var ErrNotSupported = errors.New("operation not supported by this channel")

// Channel abstracts a messaging channel (WeCom, Feishu, DingTalk, etc.).
type Channel interface {
	Name() string
	Start(handler MessageHandler) error
	Reply(ctx context.Context, replyCtx any, content string) error
	Send(ctx context.Context, replyCtx any, content string) error
	Stop() error
}

// ReplyContextReconstructor is an optional interface for channels that can
// recreate a reply context from a session key. This is needed for cron jobs
// to send messages to users without an incoming message.
type ReplyContextReconstructor interface {
	ReconstructReplyCtx(sessionKey string) (any, error)
}

// TypingIndicator is an optional interface for channels that can show a
// "processing" indicator (typing bubble, emoji reaction, etc.) while the
// agent is working.
type TypingIndicator interface {
	StartTyping(ctx context.Context, replyCtx any) (stop func())
}

// ImageSender is an optional interface for channels that support sending images.
type ImageSender interface {
	SendImage(ctx context.Context, replyCtx any, img ImageAttachment) error
}

// FileSender is an optional interface for channels that support sending files.
type FileSender interface {
	SendFile(ctx context.Context, replyCtx any, file FileAttachment) error
}

// MessageUpdater is an optional interface for channels that support updating messages.
type MessageUpdater interface {
	UpdateMessage(ctx context.Context, replyCtx any, content string) error
}

// StreamReplier is an optional interface for channels that support streaming message updates.
// The channel accumulates content locally and sends full accumulated content on each update.
type StreamReplier interface {
	StartStream(ctx context.Context, replyCtx any, content string) (streamID string, err error)
	AppendStream(ctx context.Context, replyCtx any, streamID string, content string) error
	FinishStream(ctx context.Context, replyCtx any, streamID string, finalContent string) error
}

// MessageHandler is called by channels when a new message arrives.
type MessageHandler func(p Channel, msg *Message)
