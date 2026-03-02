package resp

import "time"

func getTime() int64 {
	return time.Now().UnixMilli()
}
