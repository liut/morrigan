package tools

import (
	"fmt"
	"strings"

	"github.com/liut/morign/pkg/models/mcps"
)

const (
	ToolNameKBSearch = "kb_search"
	ToolNameKBCreate = "kb_create"
	ToolNameFetch    = "fetch"
)

// ToolDescriptor 变量定义
var (
	// kbSearchDescriptor 知识库搜索工具描述
	kbSearchDescriptor = mcps.ToolDescriptor{
		Name:        ToolNameKBSearch,
		Description: "Search documents in knowledge base with subject. \nWhen faced with unknown or uncertain issues, prioritize consulting the knowledge base.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"subject": map[string]any{
					"type":        "string",
					"description": "text of keywords or subject",
				},
			},
			"required": []string{"subject"},
		},
	}

	// kbCreateDescriptor 知识库创建工具描述（需要 keeper 角色）
	kbCreateDescriptor = mcps.ToolDescriptor{
		Name:        ToolNameKBCreate,
		Description: "Create new document of knowledge base, all parameters are required. Note: Unless the user explicitly requests supplementary content, do not invoke it. Before invoking, always perform a kb_search to confirm there is no corresponding subject or content. If similar content already exists, do not invoke even if requested by the user!",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]any{
					"type":        "string",
					"description": "title of document, like a main name or topic",
				},
				"heading": map[string]any{
					"type":        "string",
					"description": "heading of document, like a sub name or property",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "long text of content of document",
				},
			},
			"required": []string{"title", "heading", "content"},
		},
	}

	// fetchDescriptor 网页抓取工具描述
	fetchDescriptor = mcps.ToolDescriptor{
		Name:        ToolNameFetch,
		Description: "Fetches a URL from the internet and optionally extracts its contents as markdown",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "URL to fetch",
				},
				"max_length": map[string]any{
					"type":        "number",
					"description": "Maximum number of characters to return, default 5000",
					"default":     5000,
					"minimum":     0,
					"maximum":     1000000,
				},
				"start_index": map[string]any{
					"type":        "number",
					"description": "On return output starting at this character index, default 0",
					"default":     0,
					"minimum":     0,
				},
				"raw": map[string]any{
					"type":        "boolean",
					"description": "Get the actual HTML content without simplification, default false",
					"default":     false,
				},
			},
			"required": []string{"url"},
		},
	}
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
