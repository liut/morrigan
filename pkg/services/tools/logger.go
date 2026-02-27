package tools

import (
	"github.com/cupogo/andvari/utils/zlog"
)

func logger() zlog.Logger {
	return zlog.Get()
}
