package web

import (
	"github.com/cupogo/andvari/utils/zlog"
)

func logger() zlog.Logger {
	return zlog.Get()
}
