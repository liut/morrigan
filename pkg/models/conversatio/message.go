package conversatio

type Message struct {
	Role    string `json:"role,omitempty" yaml:"role,omitempty"`
	Content string `json:"content" yaml:"content"`
	ID      string `json:"id,omitempty" yaml:"id,omitempty"`
}

type Messages []Message

type Preset struct {
	Welcome     *Message `json:"welcome,omitempty" yaml:"welcome,omitempty"`
	Messages    Messages `json:"messages,omitempty" yaml:"messages,omitempty"`
	Model       string   `json:"model,omitempty" yaml:"model,omitempty"`
	MaxTokens   int      `json:"maxTokens,omitempty" yaml:"maxTokens,omitempty"`
	Temperature float32  `json:"temperature,omitempty" yaml:"temperature,omitempty"`
	Stop        []string `json:"stop,omitempty,omitempty" yaml:"stop,omitempty"`
}
