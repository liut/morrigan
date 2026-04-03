package llm

import (
	"net/http"
	"time"
)

// config LLM 客户端配置
type config struct {
	provider    string
	baseURL     string
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	timeout     time.Duration
	httpClient  HTTPDoer
	headers     map[string]string
	debug       bool
}

// HTTPDoer HTTP 请求接口
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Option 函数式选项
type Option func(*config)

// WithProvider 设置 provider
func WithProvider(provider string) Option {
	return func(c *config) {
		c.provider = provider
	}
}

// WithBaseURL 设置 base URL
func WithBaseURL(baseURL string) Option {
	return func(c *config) {
		c.baseURL = baseURL
	}
}

// WithAPIKey 设置 API Key
func WithAPIKey(apiKey string) Option {
	return func(c *config) {
		c.apiKey = apiKey
	}
}

// WithModel 设置模型
func WithModel(model string) Option {
	return func(c *config) {
		c.model = model
	}
}

// WithMaxTokens 设置最大 token 数
func WithMaxTokens(maxTokens int) Option {
	return func(c *config) {
		c.maxTokens = maxTokens
	}
}

// WithTemperature 设置温度
func WithTemperature(temperature float64) Option {
	return func(c *config) {
		c.temperature = temperature
	}
}

// WithTimeout 设置超时
func WithTimeout(timeout time.Duration) Option {
	return func(c *config) {
		c.timeout = timeout
	}
}

// WithHTTPClient 设置自定义 HTTP Client
func WithHTTPClient(client HTTPDoer) Option {
	return func(c *config) {
		c.httpClient = client
	}
}

// WithHeaders 设置自定义头
func WithHeaders(headers map[string]string) Option {
	return func(c *config) {
		c.headers = headers
	}
}

// WithMessages 设置消息列表（用于 Chat 方法）
func WithMessages(messages []Message) Option {
	return func(c *config) {
		// 消息通过 config.messages 传递，这里是个占位
		// 实际在方法参数中处理
		_ = messages
	}
}

// WithTools 设置工具定义
func WithTools(tools []ToolDefinition) Option {
	return func(c *config) {
		_ = tools
	}
}

// WithStream 设置是否流式
func WithStream(stream bool) Option {
	return func(c *config) {
		_ = stream
	}
}

// WithDebug 设置调试模式
func WithDebug(debug bool) Option {
	return func(c *config) {
		c.debug = debug
	}
}

// defaultConfig 返回默认配置
func defaultConfig() *config {
	return &config{
		provider:    "openai",
		model:       "gpt-3.5-turbo",
		maxTokens:   4096,
		temperature: 0.7,
		timeout:     90 * time.Second,
	}
}

// applyOptions 应用选项
func applyOptions(opts ...Option) *config {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
