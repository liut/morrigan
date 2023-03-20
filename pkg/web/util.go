package web

import (
	"fmt"
	"math"
	"reflect"
	"runtime"
)

func nameOfFunction(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

var byteUnits = []string{"B", "KB", "MB", "GB", "TB", "PB"}

// FormatRate formats a floating point b/s as string with closest unit
func FormatRate(rate float64) (str string) {
	return FormatBytes(rate, "/sec")
}

// FormatBytes formats a floating number as string with closest unit
func FormatBytes(num float64, suffix string) (str string) {
	if math.IsInf(num, 0) {
		str = "infinity"
		return
	}
	var idx int
	for num > 1024.0 {
		num /= 1024.0
		idx++
	}
	str = fmt.Sprintf("%.2f%s%s", num, byteUnits[idx], suffix)
	return
}
