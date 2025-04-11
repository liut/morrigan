package mcputils

import (
	"github.com/cupogo/andvari/utils/zlog"
)

// nolint
func logger() zlog.Logger {
	return zlog.Get()
}
