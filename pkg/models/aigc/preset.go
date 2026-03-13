package aigc

type Preset struct {
	Welcome      string `json:"welcome,omitempty" yaml:"welcome,omitempty"`
	SystemPrompt string `json:"systemPrompt,omitempty" yaml:"systemPrompt,omitempty"`
	ToolsPrompt  string `json:"toolsPrompt,omitempty" yaml:"toolsPrompt,omitempty"`

	// toolName -> description
	Tools map[string]string `json:"tools,omitempty" yaml:"tools,omitempty"`
}
