package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const anthropicVersion = "2023-06-01"

// anthropicProvider Anthropic Provider 实现
type anthropicProvider struct{}

// newAnthropicProvider 创建 Anthropic Provider
func newAnthropicProvider() *anthropicProvider {
	return &anthropicProvider{}
}

func (p *anthropicProvider) Chat(ctx context.Context, cfg *config, messages []Message, tools []ToolDefinition) (*ChatResult, error) {
	endpoint := anthropicMessagesEndpoint(cfg.baseURL)

	anthropicMessages, systemText := toAnthropicMessages(messages)

	reqBody := struct {
		System      string          `json:"system,omitempty"`
		Model       string          `json:"model"`
		Messages    []anthropicMsg  `json:"messages"`
		Tools       []anthropicTool `json:"tools,omitempty"`
		MaxTokens   int             `json:"max_tokens"`
		Temperature *float64        `json:"temperature,omitempty"`
	}{
		Model:       cfg.model,
		Messages:    anthropicMessages,
		System:      systemText,
		MaxTokens:   cfg.maxTokens,
		Temperature: float64Ptr(cfg.temperature),
	}
	if len(tools) > 0 {
		converted, err := toAnthropicTools(tools)
		if err != nil {
			return nil, err
		}
		reqBody.Tools = converted
	}
	logger().Infow("chat start",
		"model", cfg.model,
		"msgs_count", len(messages),
		"tools_count", len(tools),
		"tools", ToolLogs(tools),
		"has_tools", len(tools) > 0,
		"endpoint", endpoint,
	)

	b, err := json.Marshal(reqBody)
	if err != nil {
		logger().Infow("marshal chat request failed", "err", err)
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		logger().Infow("create chat request failed", "err", err, "endpoint", endpoint)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if cfg.apiKey != "" {
		req.Header.Set("x-api-key", cfg.apiKey)
	}
	req.Header.Set("anthropic-version", anthropicVersion)
	for k, v := range cfg.headers {
		req.Header.Set(k, v)
	}

	hc := cfg.httpClient
	if hc == nil {
		hc = &http.Client{Timeout: cfg.timeout}
	}

	resp, err := hc.Do(req)
	if err != nil {
		logger().Warnw("anthropic request failed", "err", err, "endpoint", endpoint)
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		logger().Warnw("anthropic response error", "status", resp.StatusCode, "body", string(body))
		return nil, errMsg
	}

	return parseAnthropicResponse(body)
}

func (p *anthropicProvider) StreamChat(ctx context.Context, cfg *config, messages []Message, tools []ToolDefinition) (<-chan StreamResult, error) {
	ch := make(chan StreamResult, 100)

	go func() {
		defer close(ch)

		// 构建请求
		endpoint := anthropicMessagesEndpoint(cfg.baseURL)
		anthropicMessages, systemText := toAnthropicMessages(messages)

		reqBody := struct {
			Model       string          `json:"model"`
			Messages    []anthropicMsg  `json:"messages"`
			System      string          `json:"system,omitempty"`
			Tools       []anthropicTool `json:"tools,omitempty"`
			MaxTokens   int             `json:"max_tokens"`
			Temperature *float64        `json:"temperature,omitempty"`
			Stream      bool            `json:"stream"`
		}{
			Model:       cfg.model,
			Messages:    anthropicMessages,
			System:      systemText,
			MaxTokens:   cfg.maxTokens,
			Temperature: float64Ptr(cfg.temperature),
			Stream:      true,
		}
		if len(tools) > 0 {
			converted, err := toAnthropicTools(tools)
			if err != nil {
				ch <- StreamResult{Error: err}
				return
			}
			reqBody.Tools = converted
		}

		logger().Infow("stream start",
			"model", cfg.model,
			"msgs_count", len(messages),
			"tools_count", len(tools),
			"tools", ToolLogs(tools),
			"has_tools", len(tools) > 0,
			"endpoint", endpoint,
			"messages", MessagesLogged(messages),
		)

		// 序列化请求体，保存用于错误时打印
		reqBodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			logger().Warnw("marshal stream request failed", "err", err)
			ch <- StreamResult{Error: err}
			return
		}

		// 构建请求
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBodyBytes))
		if err != nil {
			logger().Warnw("create stream request failed", "err", err, "reqBody", string(reqBodyBytes))
			ch <- StreamResult{Error: err}
			return
		}

		req.Header.Set("Content-Type", "application/json")
		if cfg.apiKey != "" {
			req.Header.Set("x-api-key", cfg.apiKey)
		}
		req.Header.Set("anthropic-version", anthropicVersion)
		for k, v := range cfg.headers {
			req.Header.Set(k, v)
		}

		// 发送请求
		hc := cfg.httpClient
		if hc == nil {
			hc = &http.Client{Timeout: 0}
		}

		resp, err := hc.Do(req)
		if err != nil {
			logger().Warnw("stream request failed", "err", err, "reqBody", string(reqBodyBytes))
			ch <- StreamResult{Error: err}
			return
		}

		// 检查响应状态码
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			fmt.Fprintf(os.Stderr, "\n%s\n", string(reqBodyBytes))
			respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
			resp.Body.Close()
			errMsg := fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
			logger().Warnw("stream response error",
				"status", resp.StatusCode,
				// "reqBody", string(reqBodyBytes),
				"respBody", string(respBody))
			ch <- StreamResult{Error: errMsg}
			return
		}
		defer resp.Body.Close()

		// 解析流响应
		if err := p.parseStreamResponse(resp.Body, ch, cfg.debug); err != nil {
			ch <- StreamResult{Error: err}
		}
	}()

	return ch, nil
}

// parseStreamResponse 解析流式响应
func (p *anthropicProvider) parseStreamResponse(body io.Reader, ch chan<- StreamResult, debug bool) error {
	var currentToolCalls []ToolCall
	var currentText strings.Builder

	bufReader := bufio.NewReaderSize(body, 1024)

	for {
		line, err := bufReader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				ch <- StreamResult{Done: true}
			} else {
				logger().Infow("read stream response failed", "err", err)
				return fmt.Errorf("read: %w", err)
			}
			return nil
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// 跳过非 data: 开头的行
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}

		if debug {
			fmt.Fprintln(os.Stderr, string(line))
		}

		data := bytes.TrimSpace(line[5:])
		if string(data) == "[DONE]" {
			ch <- StreamResult{Done: true}
			return nil
		}

		event, err := p.parseStreamEvent(data)
		if err != nil {
			continue
		}
		// logger().Debugw("stream event parsed", "type", event.Type, "index", event.Index,
		// 	"delta", &event.Delta)

		done, toolCalls := p.handleStreamEvent(event, &currentText, currentToolCalls, ch)
		currentToolCalls = toolCalls

		if done {
			logger().Infow("stream done", "event_type", event.Type, "tool_calls_count", len(currentToolCalls))
			return nil
		}
	}
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func (au *anthropicUsage) toUsage() *Usage {
	if au == nil {
		return nil
	}
	return &Usage{
		InputTokens:  au.InputTokens,
		OutputTokens: au.OutputTokens,
		TotalTokens:  au.InputTokens + au.OutputTokens,
	}
}

// streamEvent Anthropic 流事件
type streamEvent struct {
	Type         string `json:"type"`
	Index        int    `json:"index"`
	ContentBlock *struct {
		Type  string          `json:"type"`
		Text  string          `json:"text,omitempty"`
		ID    string          `json:"id,omitempty"`
		Name  string          `json:"name,omitempty"`
		Input json.RawMessage `json:"input,omitempty"`
	} `json:"content_block,omitempty"`
	Delta struct {
		Text        string `json:"text,omitempty"`
		Thinking    string `json:"thinking,omitempty"`
		Type        string `json:"type,omitempty"`
		ID          string `json:"id,omitempty"`
		Name        string `json:"name,omitempty"`
		Input       any    `json:"input,omitempty"`
		PartialJSON string `json:"partial_json,omitempty"`
		StopReason  string `json:"stop_reason,omitempty"`
	} `json:"delta,omitempty"`
	Message *struct {
		ID    string `json:"id,omitempty"`
		Model string `json:"model,omitempty"`
	} `json:"message,omitempty"`
	Usage *anthropicUsage `json:"usage,omitempty"`
}

// parseStreamEvent 解析单个流事件
func (p *anthropicProvider) parseStreamEvent(data []byte) (streamEvent, error) {
	var event streamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		logger().Infow("parse anthropic stream event fail", "err", err, "data", string(data))
		return event, err
	}
	return event, nil
}

// handleStreamEvent 处理流事件，返回是否结束及更新后的 toolCalls
func (p *anthropicProvider) handleStreamEvent(event streamEvent, currentText *strings.Builder, currentToolCalls []ToolCall, ch chan<- StreamResult) (bool, []ToolCall) {
	switch event.Type {
	case "content_block_start":
		// 开始新的内容块，检查是否是 tool_use 类型
		if event.ContentBlock != nil && event.ContentBlock.Type == "tool_use" {
			toolID := event.ContentBlock.ID
			if toolID == "" {
				toolID = fmt.Sprintf("toolu_%d", event.Index)
			}
			currentToolCalls = append(currentToolCalls, ToolCall{
				ID:   toolID,
				Type: "function",
				Function: ToolCallFunc{
					Name: event.ContentBlock.Name,
				},
			})
			logger().Debugw("tool_use started", "id", toolID, "name", event.ContentBlock.Name)
		}
	case "content_block_delta":
		if event.Delta.Type == "text_delta" {
			currentText.WriteString(event.Delta.Text)
			ch <- StreamResult{
				Delta:     event.Delta.Text,
				ToolCalls: currentToolCalls,
			}
		} else if event.Delta.Type == "thinking_delta" {
			ch <- StreamResult{
				Think:     event.Delta.Thinking,
				ToolCalls: currentToolCalls,
			}
		} else if event.Delta.Type == "input_json_delta" {
			// 处理 tool_use 的参数，直接取最后一个 tool_call
			if len(currentToolCalls) > 0 && event.Delta.PartialJSON != "" {
				lastIdx := len(currentToolCalls) - 1
				currentToolCalls[lastIdx].Function.Arguments = append(
					currentToolCalls[lastIdx].Function.Arguments,
					event.Delta.PartialJSON...,
				)
			}
			logger().Debugw("input json", "delta", event.Delta, "toolCalls", currentToolCalls)
		}
	case "content_block_stop":
		// 内容块结束
	case "message_delta":
		stopReason := FinishReason(event.Delta.StopReason)
		if stopReason == "end_turn" {
			stopReason = "stop"
		} else if len(currentToolCalls) > 0 { // 检查是否有 tool_calls
			stopReason = "tool_calls" // 为了兼容 OpenAI
		}
		// 发送完成信号
		ch <- StreamResult{
			Done:         true,
			ToolCalls:    currentToolCalls,
			FinishReason: stopReason,
			Usage:        event.Usage.toUsage(),
		}
		return true, currentToolCalls
	case "message_stop":
		ch <- StreamResult{
			Done:      true,
			ToolCalls: currentToolCalls,
		}
		return true, currentToolCalls
	case "message_start":
		if event.Message != nil {
			ch <- StreamResult{
				Model:      event.Message.Model,
				ResponseID: event.Message.ID,
			}
		}
		// 忽略
	case "ping":
		// 忽略
	default:
		logger().Infow("unknown anthropic event type", "type", event.Type)
	}
	return false, currentToolCalls
}

func (p *anthropicProvider) Generate(ctx context.Context, cfg *config, prompt string) (string, *Usage, error) {
	// Anthropic 不支持 Completion API，使用 Chat 代替
	messages := []Message{{Role: RoleUser, Content: prompt}}
	result, err := p.Chat(ctx, cfg, messages, nil)
	if err != nil {
		return "", nil, err
	}
	return result.Content, result.Usage, nil
}

func (p *anthropicProvider) Embedding(ctx context.Context, cfg *config, texts []string) ([]float64, error) {
	// Anthropic 不支持 Embedding API，返回错误
	return nil, fmt.Errorf("embedding not supported for anthropic provider")
}

// anthropicMsg Anthropic 消息格式
type anthropicMsg struct {
	Role    string                 `json:"role"`
	Content []anthropicContentPart `json:"content"`
}

// anthropicContentPart Anthropic 内容块
type anthropicContentPart struct {
	Type      string           `json:"type"`
	Text      string           `json:"text,omitempty"`
	Source    *anthropicSource `json:"source,omitempty"`
	ID        string           `json:"id,omitempty"`
	Name      string           `json:"name,omitempty"`
	Input     json.RawMessage  `json:"input,omitempty"`
	ToolUseID string           `json:"tool_use_id,omitempty"`
	Content   string           `json:"content,omitempty"`
}

// anthropicSource Anthropic 图片源
type anthropicSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type,omitempty"`
	Data      string `json:"data,omitempty"`
}

// anthropicTool Anthropic 工具定义
type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema,omitempty"`
}

// toAnthropicTools 将 ToolDefinition 转换为 Anthropic 工具格式
func toAnthropicTools(tools []ToolDefinition) ([]anthropicTool, error) {
	out := make([]anthropicTool, 0, len(tools))
	for _, t := range tools {
		params, err := schemaToRawJSON(t.Function.Parameters)
		if err != nil {
			logger().Infow("convert tool to anthropic format failed", "err", err,
				"tool", t.Function.Name)
			return nil, fmt.Errorf("anthropic tool schema %s: %w", t.Function.Name, err)
		}
		out = append(out, anthropicTool{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			InputSchema: params,
		})
	}
	return out, nil
}

// toAnthropicMessages 将 Message 列表转换为 Anthropic 消息格式
func toAnthropicMessages(messages []Message) ([]anthropicMsg, string) {
	out := make([]anthropicMsg, 0, len(messages))
	systemParts := make([]string, 0, 1)
	pendingToolResults := make([]anthropicContentPart, 0)

	flushToolResults := func() {
		if len(pendingToolResults) == 0 {
			return
		}
		out = append(out, anthropicMsg{
			Role:    "user",
			Content: pendingToolResults,
		})
		pendingToolResults = pendingToolResults[:0]
	}

	for _, m := range messages {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		switch role {
		case "system":
			if strings.TrimSpace(m.Content) != "" {
				systemParts = append(systemParts, m.Content)
			}
		case "tool":
			pendingToolResults = append(pendingToolResults, anthropicContentPart{
				Type:      "tool_result",
				ToolUseID: m.ToolCallID,
				Content:   m.Content,
			})
		case "user", "assistant":
			flushToolResults()

			parts := toAnthropicInputParts(m)
			if role == "assistant" {
				for i, tc := range m.ToolCalls {
					toolID := strings.TrimSpace(tc.ID)
					if toolID == "" {
						toolID = fmt.Sprintf("toolu_%d", i+1)
					}
					parts = append(parts, anthropicContentPart{
						Type:  "tool_use",
						ID:    toolID,
						Name:  tc.Function.Name,
						Input: parseArgsToRawJSON(string(tc.Function.Arguments)),
					})
				}
			}
			if len(parts) > 0 {
				out = append(out, anthropicMsg{
					Role:    role,
					Content: parts,
				})
			}
		}
	}
	flushToolResults()
	return out, strings.Join(systemParts, "\n\n")
}

// toAnthropicInputParts 将 Message 转换为 Anthropic 输入内容块
func toAnthropicInputParts(m Message) []anthropicContentPart {
	if strings.TrimSpace(m.Content) == "" {
		return nil
	}
	return []anthropicContentPart{{
		Type: "text",
		Text: m.Content,
	}}
}

// anthropicMessagesEndpoint 构建 Anthropic Messages API 端点
func anthropicMessagesEndpoint(baseURL string) string {
	base := strings.TrimRight(baseURL, "/")
	if base == "" {
		base = "https://api.anthropic.com"
	}
	if strings.HasSuffix(base, "/v1") {
		return base + "/messages"
	}
	return base + "/v1/messages"
}

// parseArgsToRawJSON 解析参数为 RawMessage
func parseArgsToRawJSON(s string) json.RawMessage {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return json.RawMessage(`{}`)
	}
	b := []byte(trimmed)
	if json.Valid(b) {
		return b
	}
	quoted, _ := json.Marshal(trimmed)
	return quoted
}

// parseAnthropicResponse 解析 Anthropic 响应
func parseAnthropicResponse(body []byte) (*ChatResult, error) {
	var parsed struct {
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text,omitempty"`
			ID    string          `json:"id,omitempty"`
			Name  string          `json:"name,omitempty"`
			Input json.RawMessage `json:"input,omitempty"`
		} `json:"content"`
		Usage *anthropicUsage `json:"usage,omitempty"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		logger().Warnw("parse anthropic response failed", "err", err, "body", string(body))
		return nil, fmt.Errorf("parse anthropic response: %w", err)
	}
	if len(parsed.Content) == 0 {
		logger().Warnw("anthropic response has empty content", "body", string(body))
		return nil, fmt.Errorf("anthropic response: empty content")
	}

	out := &ChatResult{
		Usage: parsed.Usage.toUsage(),
	}
	var textParts []string
	for i, part := range parsed.Content {
		switch part.Type {
		case "text":
			if strings.TrimSpace(part.Text) != "" {
				textParts = append(textParts, part.Text)
			}
		case "tool_use":
			toolID := strings.TrimSpace(part.ID)
			if toolID == "" {
				toolID = fmt.Sprintf("toolu_%d", i+1)
			}
			args := part.Input
			if len(args) == 0 {
				args = json.RawMessage(`{}`)
			}
			out.ToolCalls = append(out.ToolCalls, ToolCall{
				ID:   toolID,
				Type: "function",
				Function: ToolCallFunc{
					Name:      part.Name,
					Arguments: args,
				},
			})
		}
	}
	out.Content = strings.Join(textParts, "\n")
	return out, nil
}

// schemaToRawJSON 将参数 schema 转换为 RawMessage
func schemaToRawJSON(params any) (json.RawMessage, error) {
	if params == nil {
		return json.RawMessage(`{"type": "object", "properties": {}}`), nil
	}
	b, err := json.Marshal(params)
	if err != nil {
		logger().Infow("schema to raw JSON failed", "err", err, "params", params)
		return nil, err
	}
	return json.RawMessage(b), nil
}

// float64Ptr 返回 float64 指针
func float64Ptr(v float64) *float64 {
	return &v
}
