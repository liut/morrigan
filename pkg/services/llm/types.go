package llm

import (
	"encoding/json"
)

// Message 表示聊天消息
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"` // 不可省略
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"` // only for role=tool
	Name       string     `json:"name,omitempty"`
}

// ToolCall 表示工具调用请求
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolCallFunc `json:"function"`
}

// ToolCallFunc 工具调用函数
type ToolCallFunc struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
	Results   any             `json:"results,omitempty"`
}

// MarshalJSON 自定义序列化，将 Arguments 转为字符串（OpenAI API 要求）
func (f ToolCallFunc) MarshalJSON() ([]byte, error) {
	type Alias ToolCallFunc
	aux := &struct {
		Arguments any `json:"arguments"`
		*Alias
	}{
		Alias: (*Alias)(&f),
	}
	// 将 json.RawMessage 转为 string
	if len(f.Arguments) > 0 {
		aux.Arguments = string(f.Arguments)
	} else {
		aux.Arguments = "{}"
	}
	return json.Marshal(aux)
}

// ToolDefinition 工具定义
type ToolDefinition struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition 函数定义
type FunctionDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

// ChatResult 聊天结果
type ChatResult struct {
	Content   string
	ToolCalls []ToolCall
	Usage     Usage
}

// HasToolCalls 判断是否有工具调用
func (r *ChatResult) HasToolCalls() bool {
	return len(r.ToolCalls) > 0
}

// StreamResult 流式响应结果
type StreamResult struct {
	Delta        string
	ToolCalls    []ToolCall
	Done         bool `json:",omitempty"`
	FinishReason string
	Error        error `json:",omitempty"`
}

// Usage token 使用统计
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Response 完整响应
type Response struct {
	Content          string
	ToolCalls        []ToolCall
	Usage            Usage
	CompletionTokens int
}

// MessageRole 消息角色常量
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)
