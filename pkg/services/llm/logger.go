package llm

import (
	"github.com/cupogo/andvari/utils/zlog"
)

// logger returns the global logger instance
func logger() zlog.Logger {
	return zlog.Get()
}
