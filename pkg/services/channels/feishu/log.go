package feishu

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

type myLogger struct {
	log *slog.Logger
}

func (l *myLogger) logInternal(ctx context.Context, level slog.Level, args ...any) {
	pcs := make([]uintptr, 1)
	// skip=3: skip runtime.Callers → runtime.CallersFrames → logInternal → SDK's c.logger.Debug call site
	runtime.Callers(3, pcs)

	msg := ""
	extra := ""
	if len(args) > 0 {
		msg = args[0].(string)
		if len(args) > 1 {
			extra = " " + args[1].(string)
		}
	}

	r := &slog.Record{
		Time:    time.Now(),
		Level:   level,
		Message: msg + extra,
		PC:      pcs[0],
	}
	l.log.Handler().Handle(ctx, *r)
}

func (l *myLogger) Debug(ctx context.Context, args ...any) {
	l.logInternal(ctx, slog.LevelDebug, args...)
}
func (l *myLogger) Info(ctx context.Context, args ...any) {
	l.logInternal(ctx, slog.LevelInfo, args...)
}
func (l *myLogger) Warn(ctx context.Context, args ...any) {
	l.logInternal(ctx, slog.LevelWarn, args...)
}
func (l *myLogger) Error(ctx context.Context, args ...any) {
	l.logInternal(ctx, slog.LevelError, args...)
}

func newDefaultLog() *myLogger {
	return &myLogger{log: slog.Default()}
}
