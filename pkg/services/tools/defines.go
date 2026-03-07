package tools

import (
	"fmt"
	"strings"
)

const (
	ToolNameKBSearch = "kb_search"
	ToolNameKBCreate = "kb_create"
	ToolNameFetch    = "fetch"
)

// ResultLogs 是工具调用结果的日志类型别名
type ResultLogs map[string]any

// formatValue 格式化单个值
// string 取前 30 字符，slice 遍历处理，map 递归处理，其他返回 ""
func formatValue(v any) string {
	switch val := v.(type) {
	case string:
		maxLen := len(val)
		if maxLen > 30 {
			maxLen = 30
		}
		return val[:maxLen]
	case []any:
		if len(val) == 0 {
			return ""
		}
		var sb strings.Builder
		sb.WriteString("[")
		for i, item := range val {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(formatValue(item))
		}
		sb.WriteString("]")
		return sb.String()
	case map[string]any:
		if len(val) == 0 {
			return ""
		}
		var sb strings.Builder
		sb.WriteString("{")
		first := true
		for k, v := range val {
			if !first {
				sb.WriteString(", ")
			}
			first = false
			sb.WriteString(fmt.Sprintf("%s: %q", k, formatValue(v)))
		}
		sb.WriteString("}")
		return sb.String()
	}
	return ""
}

// String 实现 fmt.Stringer 接口，返回格式化的结果日志
// 格式为 {key: val[0:min(30,len(val)}，如果值不是 string/slice/map，则为 key: ""
func (rl ResultLogs) String() string {
	if rl == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("{")
	first := true
	for k, v := range rl {
		if !first {
			sb.WriteString(", ")
		}
		first = false
		sb.WriteString(fmt.Sprintf("%s: %q", k, formatValue(v)))
	}
	sb.WriteString("}")
	return sb.String()
}
