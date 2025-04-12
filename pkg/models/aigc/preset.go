package aigc

type Completion struct {
	Header string `json:"header,omitempty" yaml:"header,omitempty"`
	Model  string `json:"model,omitempty" yaml:"model,omitempty"`
}

type Message struct {
	Role    string `json:"role,omitempty" yaml:"role,omitempty"`
	Content string `json:"content" yaml:"content"`
	ID      string `json:"id,omitempty" yaml:"id,omitempty"`
}

type Messages []Message

type Preset struct {
	Completion  *Completion `json:"completion,omitempty" yaml:"completion,omitempty"`
	Welcome     *Message    `json:"welcome,omitempty" yaml:"welcome,omitempty"`
	Messages    Messages    `json:"messages,omitempty" yaml:"messages,omitempty"`
	BeforeQA    Messages    `json:"beforeQA,omitempty" yaml:"beforeQA,omitempty"`
	AfterQA     Messages    `json:"afterQA,omitempty" yaml:"afterQA,omitempty"`
	Model       string      `json:"model,omitempty" yaml:"model,omitempty"`
	MaxTokens   int         `json:"maxTokens,omitempty" yaml:"maxTokens,omitempty"`
	Temperature float32     `json:"temperature,omitempty" yaml:"temperature,omitempty"`
	Stop        []string    `json:"stop,omitempty" yaml:"stop,omitempty"`
	MCPServers  []MCPServer `json:"mcps,omitempty" yaml:"mcps,omitempty"`
}

type MCPType string // 'sse' | 'stdio' | 'inmemory' | 'http'
const (
	MtSSE      MCPType = "sse"
	MtStdIO    MCPType = "stdio"
	MtInMemory MCPType = "inmemory"
	MtHTTP     MCPType = "http"
)

// MCPServer is a MCP server
type MCPServer struct {
	Code string  `json:"code" yaml:"code"`
	Name string  `json:"name" yaml:"name"`
	Type MCPType `json:"type" yaml:"type"`
	URL  string  `json:"url" yaml:"url"`
}
