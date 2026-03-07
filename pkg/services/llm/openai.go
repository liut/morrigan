package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// headerDataRE 匹配 SSE data: 前缀（与 go-openai 保持一致）
var headerDataRE = regexp.MustCompile(`^data:\s*`)

// chatRequestBody OpenAI Chat Completion 请求体
type chatRequestBody struct {
	Model       string           `json:"model"`
	Messages    []Message        `json:"messages"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
	Stream      bool             `json:"stream"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	ToolChoice  string           `json:"tool_choice,omitempty"`
}

// openAIProvider OpenAI Provider 实现
type openAIProvider struct{}

// newOpenAIProvider 创建 OpenAI Provider
func newOpenAIProvider() *openAIProvider {
	return &openAIProvider{}
}

func (p *openAIProvider) Chat(ctx context.Context, cfg *config, messages []Message, tools []ToolDefinition) (*ChatResult, error) {
	body, err := p.doChatRequest(ctx, cfg, messages, tools, false)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string          `json:"name"`
						Arguments json.RawMessage `json:"arguments"`
						Results   json.RawMessage `json:"results"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	result := &ChatResult{
		Content: resp.Choices[0].Message.Content,
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	for _, tc := range resp.Choices[0].Message.ToolCalls {
		args := tc.Function.Arguments
		// 处理 arguments 可能被 JSON 字符串包裹的情况
		if len(args) > 0 && args[0] == '"' {
			var s string
			if err := json.Unmarshal(args, &s); err == nil {
				args = []byte(s)
			}
		}
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: ToolCallFunc{
				Name:      tc.Function.Name,
				Arguments: args,
				Results:   tc.Function.Results,
			},
		})
	}

	return result, nil
}

func (p *openAIProvider) StreamChat(ctx context.Context, cfg *config, messages []Message, tools []ToolDefinition) (<-chan StreamResult, error) {
	ch := make(chan StreamResult, 100)

	// 启动流式读取 goroutine
	go func() {
		defer close(ch)

		endpoint := buildEndpoint(cfg.baseURL, "/chat/completions")

		var toolsOpt []ToolDefinition
		if len(tools) > 0 {
			toolsOpt = tools
		}

		reqBody := chatRequestBody{
			Model:       cfg.model,
			Messages:    toOpenAIMessages(messages),
			MaxTokens:   cfg.maxTokens,
			Temperature: cfg.temperature,
			Stream:      true,
			Tools:       toolsOpt,
		}
		if len(toolsOpt) > 0 {
			reqBody.ToolChoice = "auto"
		}

		logger().Infow("stream start",
			"model", cfg.model,
			"msgs_count", len(messages),
			"tools_count", len(tools),
			"tools", tools,
			"has_tools", len(tools) > 0,
			"endpoint", endpoint,
			"messages", MessagesLogged(messages),
		)
		// logger().Debugw("stream start", "msgs", messages)

		// 发送流式请求
		body, err := p.doStreamRequest(ctx, cfg, endpoint, reqBody)
		if err != nil {
			logger().Infow("request fail", "err", err, "reqBody", &reqBody)
			ch <- StreamResult{Error: err}
			return
		}
		defer body.Close()

		// 使用 bufio.Reader 逐行读取 SSE 响应（与 go-openai 保持一致）
		bufReader := bufio.NewReaderSize(body, 1024)

		var currentToolCalls []ToolCall
		var finishReason string
		var lines int

		for {
			lines++
			rawLine, err := bufReader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					ch <- StreamResult{Done: true}
				} else {
					ch <- StreamResult{Error: fmt.Errorf("read: %w", err)}
				}
				return
			}

			noSpaceLine := bytes.TrimSpace(rawLine)
			// 跳过非 data: 开头的行
			if !headerDataRE.Match(noSpaceLine) {
				// logger().Debugw("noSpaceLine", "rawLine", rawLine)
				continue
			}

			// 去除 data: 前缀
			noPrefixLine := headerDataRE.ReplaceAll(noSpaceLine, nil)
			if string(noPrefixLine) == "[DONE]" {
				logger().Infow("stream DONE", "lines", lines)
				// 流结束
				ch <- StreamResult{Done: true}
				return
			}

			var chunk struct {
				Choices []struct {
					Delta struct {
						Content   string `json:"content"`
						ToolCalls []struct {
							ID       string `json:"id"`
							Type     string `json:"type"`
							Index    int    `json:"index"`
							Function struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
								Results   any    `json:"results"`
							} `json:"function"`
						} `json:"tool_calls"`
					} `json:"delta"`
					FinishReason string `json:"finish_reason"`
				} `json:"choices"`
			}

			if err := json.Unmarshal(noPrefixLine, &chunk); err != nil {
				logger().Infow("parse chunk fail", "err", err, "rawLine", rawLine)
				ch <- StreamResult{Error: fmt.Errorf("parse chunk: %w", err)}
				return
			}

			if len(chunk.Choices) == 0 {
				logger().Debugw("choices is empty", "rawLine", rawLine)
				continue
			}

			delta := chunk.Choices[0].Delta
			finishReason = chunk.Choices[0].FinishReason

			// 累积 tool calls
			for _, tc := range delta.ToolCalls {
				if tc.Index >= len(currentToolCalls) {
					currentToolCalls = append(currentToolCalls, ToolCall{
						ID:   tc.ID,
						Type: tc.Type,
					})
				}
				currentToolCalls[tc.Index].Function.Name += tc.Function.Name
				currentToolCalls[tc.Index].Function.Arguments = append(
					currentToolCalls[tc.Index].Function.Arguments,
					tc.Function.Arguments...,
				)
			}

			// 发送内容，每个 chunk 都带上累积的 tool_calls
			result := StreamResult{
				Delta:        delta.Content,
				ToolCalls:    currentToolCalls,
				FinishReason: finishReason,
			}

			// 检查是否需要结束流：
			// 1. finish_reason 不为空（标准行为）
			// 2. tool_calls_len 为 0 且之前累积了 tool_calls（DeepSeek 行为）
			shouldEndStream := finishReason != "" || (len(delta.ToolCalls) == 0 && len(currentToolCalls) > 0)

			if shouldEndStream {
				logger().Debugw("stream should done", "result", &result)
				result.Done = true
			}
			ch <- result

			// 结束流
			if shouldEndStream {
				logger().Infow("stream done", "finish_reason", finishReason,
					"tool_calls_count", len(currentToolCalls), "lines", lines)
				return
			}
		}
	}()

	return ch, nil
}

// doStreamRequest 发送流式请求，返回响应体供逐行读取
func (p *openAIProvider) doStreamRequest(ctx context.Context, cfg *config, endpoint string, body any) (io.ReadCloser, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if cfg.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.apiKey)
	}
	for k, v := range cfg.headers {
		req.Header.Set(k, v)
	}

	hc := cfg.httpClient
	if hc == nil {
		hc = &http.Client{Timeout: 0} // 流式请求不设置超时
	}

	resp, err := hc.Do(req)
	if err != nil {
		logger().Infow("request fail", "err", err)
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		resp.Body.Close()
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return resp.Body, nil
}

// doChatRequest 发送聊天请求的公共方法
func (p *openAIProvider) doChatRequest(ctx context.Context, cfg *config, messages []Message, tools []ToolDefinition, stream bool) ([]byte, error) {
	endpoint := buildEndpoint(cfg.baseURL, "/chat/completions")

	var toolsOpt []ToolDefinition
	if len(tools) > 0 {
		toolsOpt = tools
	}

	reqBody := chatRequestBody{
		Model:       cfg.model,
		Messages:    toOpenAIMessages(messages),
		MaxTokens:   cfg.maxTokens,
		Temperature: cfg.temperature,
		Stream:      stream,
		Tools:       toolsOpt,
		ToolChoice:  "auto",
	}

	return p.doRequest(ctx, cfg, endpoint, reqBody)
}

// buildEndpoint 构建 API 端点（OpenAI 兼容接口）
func buildEndpoint(baseURL, path string) string {
	endpoint := strings.TrimRight(baseURL, "/")
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}
	return endpoint + path
}

func (p *openAIProvider) doRequest(ctx context.Context, cfg *config, endpoint string, body any) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if cfg.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.apiKey)
	}
	for k, v := range cfg.headers {
		req.Header.Set(k, v)
	}

	hc := cfg.httpClient
	if hc == nil {
		hc = &http.Client{Timeout: cfg.timeout}
	}

	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return respBody, nil
}

func toOpenAIMessages(messages []Message) []Message {
	return messages
}

// Generate 简单文本生成（使用 Completion API）
func (p *openAIProvider) Generate(ctx context.Context, cfg *config, prompt string) (string, Usage, error) {
	endpoint := buildEndpoint(cfg.baseURL, "/completions")

	reqBody := map[string]any{
		"model":       cfg.model,
		"prompt":      prompt,
		"max_tokens":  cfg.maxTokens,
		"temperature": cfg.temperature,
	}

	body, err := p.doRequest(ctx, cfg, endpoint, reqBody)
	if err != nil {
		return "", Usage{}, err
	}

	var resp struct {
		Choices []struct {
			Text string `json:"text"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return "", Usage{}, fmt.Errorf("parse response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", Usage{}, fmt.Errorf("no choices in response")
	}

	usage := Usage{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}

	return resp.Choices[0].Text, usage, nil
}

// Embedding 向量化文本
func (p *openAIProvider) Embedding(ctx context.Context, cfg *config, texts []string) ([]float64, error) {
	endpoint := buildEndpoint(cfg.baseURL, "/embeddings")

	// 使用默认的 embedding model
	model := "text-embedding-3-small"
	if cfg.model != "" {
		model = cfg.model
	}

	reqBody := map[string]any{
		"input": texts,
		"model": model,
	}

	body, err := p.doRequest(ctx, cfg, endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data")
	}

	return resp.Data[0].Embedding, nil
}
