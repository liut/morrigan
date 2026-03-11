package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jpillora/eventsource"
	"github.com/marcsv/go-binder/binder"

	"github.com/liut/morign/pkg/models/aigc"
	"github.com/liut/morign/pkg/models/corpus"
	"github.com/liut/morign/pkg/models/mcps"
	"github.com/liut/morign/pkg/services/llm"
	"github.com/liut/morign/pkg/services/stores"
	toolsvc "github.com/liut/morign/pkg/services/tools"
	"github.com/liut/morign/pkg/settings"
)

func init() {
	regHI(true, "POST", "/chat", "", func(a *api) http.HandlerFunc {
		return a.postChat
	})
	regHI(true, "POST", "/chat-{suffix}", "", func(a *api) http.HandlerFunc {
		return a.postChat
	})
	regHI(true, "GET", "/welcome", "", func(a *api) http.HandlerFunc {
		return a.getWelcome
	})
	regHI(true, "GET", "/history/{cid}", "", func(a *api) http.HandlerFunc {
		return a.getHistory
	})
	regHI(true, "GET", "/tools", "", func(a *api) http.HandlerFunc {
		return a.getTools
	})
}

// chatRequest 内部聊天请求结构
type chatRequest struct {
	messages  []llm.Message
	tools     []llm.ToolDefinition
	isSSE     bool
	cs        stores.Conversation
	hi        *aigc.HistoryItem
	chunkIdx  int  // 全局 chunk 计数器，用于 SSE 事件序号
}

// convertMCPToolsToLLMTools 将 MCP 工具描述转换为 LLM 工具定义
func convertMCPToolsToLLMTools(tools []mcps.ToolDescriptor) []llm.ToolDefinition {
	result := make([]llm.ToolDefinition, 0, len(tools))
	for _, td := range tools {
		result = append(result, llm.ToolDefinition{
			Type: "function",
			Function: llm.FunctionDefinition{
				Name:        td.Name,
				Description: td.Description,
				Parameters:  td.InputSchema,
			},
		})
	}
	return result
}

func (a *api) prepareChatRequest(ctx context.Context, param *ChatRequest) *chatRequest {
	cs := stores.NewConversation(ctx, param.GetConversionID())
	var messages []llm.Message

	systemPrompt := dftSystemMsg
	if len(a.preset.SystemPrompt) > 0 {
		systemPrompt = a.preset.SystemPrompt
	}
	if settings.Current.DateInContext {
		systemPrompt = systemPrompt + "\n" + thisMoment()
	}
	messages = append(messages, llm.Message{
		Role:    llm.RoleSystem,
		Content: systemPrompt,
	})

	var tools []llm.ToolDefinition
	if len(a.toolreg.ToolsFor(ctx)) > 0 {
		// 转换 MCP 工具为 LLM 工具定义
		tools = convertMCPToolsToLLMTools(a.toolreg.ToolsFor(ctx))
		toolsPrompt := dftToolsMsg
		if len(a.preset.ToolsPrompt) > 0 {
			toolsPrompt = a.preset.ToolsPrompt
		}
		messages = append(messages, llm.Message{
			Role:    llm.RoleSystem,
			Content: toolsPrompt,
		})
	} else { // 没有工具，使用问答
		docs, err := a.sto.Cob().MatchDocments(ctx, stores.MatchSpec{
			Question: param.Prompt,
			Limit:    5,
		})
		if err == nil {
			logger().Infow("matches", "docs", len(docs), "prompt", param.Prompt)
			content := docs.MarkdownText()
			if len(docs) == 0 {
				// 知识库未命中，添加明确提示
				content += "\nPlease honestly state that you don't know rather than making up an answer."
			}
			messages = append(messages, llm.Message{
				Role:    llm.RoleSystem,
				Content: content,
			})
		} else {
			logger().Infow("match fail", "err", err)
			// TODO: err ?
		}
	}

	data, err := cs.ListHistory(ctx)
	if err == nil && len(data) > 0 {
		logger().Infow("found history", "size", len(data), "hist", aigc.HiLogged(data))
		data = data.RecentlyWithTokens(historyLimitToken)

		for i, hi := range data {
			if hi.ChatItem != nil {
				isLast := i == len(data)-1
				isRetry := hi.ChatItem.User == param.Prompt

				// 最后一条特殊处理：如果是重试或 Regenerate，完全跳过这一条历史
				if isLast && (isRetry || param.Regenerate) {
					logger().Debugw("skip last history", "retry", isRetry, "regenerate", param.Regenerate)
					break
				}

				// 添加 User
				if len(hi.ChatItem.User) > 0 {
					messages = append(messages, llm.Message{
						Role: llm.RoleUser, Content: hi.ChatItem.User})
				}

				// 添加 Assistant
				if len(hi.ChatItem.Assistant) > 0 {
					messages = append(messages, llm.Message{
						Role: llm.RoleAssistant, Content: hi.ChatItem.Assistant})
				}
			}
		}
	}

	messages = append(messages, llm.Message{
		Role:    llm.RoleUser,
		Content: param.Prompt,
	})

	return &chatRequest{
		messages: messages,
		tools:    tools,
		cs:       cs,
		hi: &aigc.HistoryItem{
			Time: time.Now().Unix(),
			ChatItem: &aigc.HistoryChatItem{
				User: param.Prompt,
			},
		},
	}
}

// @Tags 聊天
// @Summary 发送聊天消息
// @Accept json
// @Produce json
// @Param token header string false "登录票据凭证"
// @Param chatRequest body ChatRequest true "聊天请求"
// @Success 200 {object} Done{result=ChatMessage}
// @Success 200 {object} Done{result=ConversationResponse}
// @Failure 400 {object} Failure "请求或参数错误"
// @Failure 500 {object} Failure "服务端错误"
// @Router /api/chat [post]
func (a *api) postChat(w http.ResponseWriter, r *http.Request) {
	var param ChatRequest
	if err := binder.BindBody(r, &param); err != nil {
		apiFail(w, r, 400, err)
		return
	}
	isProcess := strings.HasSuffix(r.URL.Path, "-process")
	isSSE := param.Stream || strings.HasSuffix(r.URL.Path, "-sse")
	isStream := param.Stream || isSSE || isProcess
	ccr := a.prepareChatRequest(r.Context(), &param)

	ccr.isSSE = isSSE

	logger().Infow("chat", "csid", param.GetConversionID(), "msgs", len(ccr.messages), "prompt", param.Prompt, "ip", r.RemoteAddr)

	if isStream {
		res := a.chatStreamResponseLoop(ccr, w, r)
		logger().Infow("stream response", "answer_len", len(res.answer), "toolCalls_len", len(res.toolCalls))
		if len(res.answer) > 0 {
			ccr.hi.ChatItem.Assistant = res.answer
			_ = ccr.cs.AddHistory(r.Context(), ccr.hi)

			if settings.Current.QAChatLog {
				in := corpus.ChatLogBasic{
					ChatID:   ccr.cs.GetOID(),
					Question: param.Prompt,
					Answer:   res.answer,
				}
				ip, _, _ := strings.Cut(r.RemoteAddr, ":")
				in.MetaAddKVs("ip", ip)
				if res.usage != nil {
					in.MetaAddKVs("usage", res.usage)
				}
				_, err := stores.Sgt().Cob().CreateChatLog(r.Context(), in)
				if err != nil {
					logger().Infow("save chat log fail", "err", err)
				}
			}
		}

		return
	}

	// 非流式场景：使用循环执行工具调用
	exec := func(ctx context.Context, messages []llm.Message, tools []llm.ToolDefinition) (string, []llm.ToolCall, *llm.Usage, error) {
		result, err := a.llm.Chat(ctx, messages, tools)
		if err != nil {
			return "", nil, nil, err
		}
		return result.Content, result.ToolCalls, &result.Usage, nil
	}
	answer, _, _, err := a.executeToolCallLoop(r.Context(), ccr.messages, ccr.tools, exec)
	if err != nil {
		apiFail(w, r, 500, err)
		return
	}
	logger().Infow("chat", "answer", answer)

	var cm ChatMessage
	cm.ID = ccr.cs.GetID() // TODO: deprecated by new message id
	cm.Text = answer
	render.JSON(w, r, &cm)
}

// writeEvent write and auto flush
func writeEvent(w io.Writer, id string, m any) bool {
	var b []byte
	var err error
	if s, ok := m.(string); ok {
		b = []byte(s)
	} else {
		b, err = json.Marshal(m)
		if err != nil {
			logger().Infow("json marshal fail", "m", m, "err", err)
			return false
		}
	}

	if err = eventsource.WriteEvent(w, eventsource.Event{
		ID:   id,
		Data: b,
	}); err != nil {
		logger().Infow("eventsource write fail", "err", err)
		return false
	}

	return true
}

// chatStreamResponseLoop 循环处理流式响应，支持工具调用循环
func (a *api) chatStreamResponseLoop(ccr *chatRequest, w http.ResponseWriter, r *http.Request) (res ChatResponse) {
	// 预先设置 HTTP 头信息（只设置一次）
	if _, ok := w.(http.Flusher); !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return ChatResponse{}
	}

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	if ccr.isSSE {
		w.Header().Set("Content-Type", "text/event-stream")
	} else {
		w.Header().Add("Content-type", "application/octet-stream")
	}
	w.Header().Add("Conversation-ID", ccr.cs.GetID())

	for {
		// 调用流式响应处理
		streamRes := a.doChatStream(ccr, w, r)
		logger().Infow("stream round done", "answer_len", len(streamRes.answer), "toolCalls_len", len(streamRes.toolCalls))

		// 累积答案
		res.answer += streamRes.answer
		if streamRes.usage != nil {
			res.usage = streamRes.usage
		}

		// 如果没有工具调用，返回结果
		if len(streamRes.toolCalls) == 0 {
			return
		}
		logger().Infow("before execute tool calls", "tools", len(streamRes.toolCalls), "msgs", len(ccr.messages))

		var hasToolCall bool
		// 执行工具调用
		ccr.messages, hasToolCall = a.doExecuteToolCalls(r.Context(), streamRes.toolCalls, ccr.messages)
		logger().Infow("executed tool calls", "hasToolCall", hasToolCall, "msgs", len(ccr.messages))
		if !hasToolCall {
			// 没有成功执行任何工具，返回当前结果
			return
		}

		// 清除工具定义，避免死循环
		ccr.tools = nil
		// 循环继续，会再次调用 LLM
	}
}

// doChatStream 执行一次流式调用，返回累积的 answer 和 toolCalls
func (a *api) doChatStream(ccr *chatRequest, w http.ResponseWriter, r *http.Request) ChatResponse {
	stream, err := a.llm.StreamChat(r.Context(), ccr.messages, ccr.tools)
	if err != nil {
		logger().Infow("call chat stream fail", "err", err)
		apiFail(w, r, 500, err)
		return ChatResponse{}
	}

	var cm ChatMessage
	if !ccr.isSSE {
		cm.ConversationID = ccr.cs.GetID()
	}

	var res ChatResponse
	var lastWriteEmpty bool // 标记上一次是否写入了空消息

	for result := range stream {
		if result.Error != nil {
			logger().Infow("stream error", "err", result.Error)
			break
		}

		var wrote bool

		cm.Delta = result.Delta
		res.answer += result.Delta
		if len(result.ToolCalls) > 0 && result.FinishReason == llm.FinishReasonToolCalls {
			cm.ToolCalls = convertToolCallsForJSON(result.ToolCalls)
		}

		if ccr.isSSE {
			if result.Done {
				ccr.chunkIdx++
				cm.ConversationID = ccr.cs.GetID()
				cm.FinishReason = string(result.FinishReason)
				_ = writeEvent(w, strconv.Itoa(ccr.chunkIdx), &cm)
			} else {
				// 判断当前是否为空消息
				isEmpty := result.Delta == "" && len(cm.ToolCalls) == 0
				if !isEmpty || !lastWriteEmpty {
					// 有内容，或者上一次不是空的，则输出
					ccr.chunkIdx++
					if wrote = writeEvent(w, strconv.Itoa(ccr.chunkIdx), &cm); !wrote {
						break
					}
					lastWriteEmpty = isEmpty
				}
				// 如果当前是空的且上一次也是空的，跳过（连续空消息只保留第一个）
			}
		} else {
			ccr.chunkIdx++
			cm.Text += result.Delta
			if err = json.NewEncoder(w).Encode(&cm); err != nil {
				logger().Infow("json encode fail", "err", err)
				break
			}
		}
		w.(http.Flusher).Flush()

		if result.Done {
			res.toolCalls = result.ToolCalls
			break
		}
	}
	logger().Infow("llm stream done", "answer", len(res.answer),
		"ahead", res.answer[0:min(32, len(res.answer))])
	return res
}

// doExecuteToolCalls 执行工具调用，返回更新后的 messages 和是否有成功执行的工具
func (a *api) doExecuteToolCalls(ctx context.Context, toolCalls []llm.ToolCall, messages []llm.Message) ([]llm.Message, bool) {
	if len(toolCalls) == 0 {
		return messages, false
	}

	messages = append(messages, llm.Message{
		Role:      llm.RoleAssistant,
		ToolCalls: toolCalls,
	})

	var hasToolCall bool
	for _, tc := range toolCalls {
		logger().Infow("chat", "toolCallID", tc.ID, "toolCallType", tc.Type, "toolCallName", tc.Function.Name)

		if tc.Type != "function" {
			continue
		}

		var parameters map[string]any
		args := string(tc.Function.Arguments)
		if args != "" && args != "{}" {
			if err := json.Unmarshal(tc.Function.Arguments, &parameters); err != nil {
				logger().Infow("chat", "toolCallID", tc.ID, "args", args, "err", err)
				continue
			}
		}
		// 空参数时使用空 map
		if parameters == nil {
			parameters = make(map[string]any)
		}

		content, err := a.toolreg.Invoke(ctx, tc.Function.Name, parameters)
		if err != nil {
			logger().Infow("invokeTool fail", "toolCallName", tc.Function.Name, "err", err)
			continue
		}

		logger().Infow("invokeTool ok", "toolCallName", tc.Function.Name,
			"content", toolsvc.ResultLogs(content))
		messages = append(messages, llm.Message{
			Role:       llm.RoleTool,
			Content:    formatToolResult(content),
			ToolCallID: tc.ID,
		})
		hasToolCall = true
	}

	return messages, hasToolCall
}

// @Tags 聊天
// @Summary 获取欢迎信息
// @Accept json
// @Produce json
// @Success 200 {object} Done{result=aigc.Message}
// @Router /api/welcome [get]
func (a *api) getWelcome(w http.ResponseWriter, r *http.Request) {
	msg := new(aigc.Message)

	if a.preset.Welcome != nil {
		msg.Content = a.preset.Welcome.Content
	} else {
		msg.Content = welcomeText
	}

	cs := stores.NewConversation(r.Context(), "")
	msg.ID = cs.GetID()
	apiOk(w, r, msg)
}

// @Tags 聊天
// @Summary 获取会话历史
// @Accept json
// @Produce json
// @Param token header string false "登录票据凭证"
// @Param cid path string true "会话ID"
// @Success 200 {object} Done{result=aigc.History}
// @Failure 500 {object} Failure "服务端错误"
// @Router /api/history/{cid} [get]
func (a *api) getHistory(w http.ResponseWriter, r *http.Request) {
	cid := chi.URLParam(r, "cid")
	cs := stores.NewConversation(r.Context(), cid)
	data, err := cs.ListHistory(r.Context())
	if err != nil {
		apiFail(w, r, 500, err)
		return
	}
	apiOk(w, r, data, 0)
}

// @Tags 聊天
// @Summary 获取可用工具列表
// @Accept json
// @Produce json
// @Param token header string false "登录票据凭证"
// @Success 200 {object} Done{result=[]Tool}
// @Router /api/tools [get]
func (a *api) getTools(w http.ResponseWriter, r *http.Request) {
	apiOk(w, r, a.toolreg.ToolsFor(r.Context()), 0)
}

// formatToolResult 将工具结果转换为文本字符串
// 优先提取 content 数组中的 text，否则使用 structuredContent
func formatToolResult(result map[string]any) string {
	if result == nil {
		return ""
	}
	logger().Debugw("formatToolResult", "result", result)
	// 优先提取 content 数组中的 text
	if content, ok := result["content"].([]any); ok {
		for _, c := range content {
			if cMap, ok := c.(map[string]any); ok {
				if text, ok := cMap["text"].(string); ok && text != "" {
					return text
				}
			}
		}
	}
	// 备选：使用 structuredContent
	if sc, ok := result["structuredContent"].(string); ok {
		return sc
	}
	if sc, ok := result["structuredContent"].(map[string]any); ok {
		for k, v := range sc {
			if s, ok := v.(string); ok && k == "text" {
				return s
			}
		}
	}
	// 最后：序列化为 JSON
	if b, err := json.Marshal(result); err == nil {
		return string(b)
	}
	return ""
}

// convertToolCallsForJSON 将 llm.ToolCall 转换为可序列化的 map 格式
func convertToolCallsForJSON(tcs []llm.ToolCall) []map[string]any {
	if len(tcs) == 0 {
		return nil
	}
	result := make([]map[string]any, len(tcs))
	for i, tc := range tcs {
		args := string(tc.Function.Arguments)
		if args == "" {
			args = "{}"
		}
		result[i] = map[string]any{
			"id":   tc.ID,
			"type": tc.Type,
			"function": map[string]any{
				"name":      tc.Function.Name,
				"arguments": args,
			},
		}
	}
	return result
}

// chatExecutor 定义聊天执行函数类型，支持流式/非流式
type chatExecutor func(ctx context.Context, messages []llm.Message, tools []llm.ToolDefinition) (string, []llm.ToolCall, *llm.Usage, error)

// executeToolCallLoop 执行工具调用循环，直到没有 tool calls
// - messages: 初始消息列表，会被修改
// - tools: 工具定义
// - exec: 执行聊天的函数（流式或非流式）
// 返回最终的 answer、最后的 toolCalls（如果有）、usage
func (a *api) executeToolCallLoop(ctx context.Context, messages []llm.Message, tools []llm.ToolDefinition, exec chatExecutor) (string, []llm.ToolCall, *llm.Usage, error) {
	for {
		answer, toolCalls, usage, err := exec(ctx, messages, tools)
		if err != nil {
			return "", nil, nil, err
		}

		if len(toolCalls) == 0 {
			return answer, nil, usage, nil
		}

		// 添加 assistant 消息（带 tool calls）
		messages = append(messages, llm.Message{
			Role:      llm.RoleAssistant,
			ToolCalls: toolCalls,
		})

		// 执行工具调用
		for _, tc := range toolCalls {
			logger().Infow("chat", "toolCallID", tc.ID, "toolCallType", tc.Type, "toolCallName", tc.Function.Name)

			if tc.Type != "function" {
				continue
			}

			var parameters map[string]any
			args := string(tc.Function.Arguments)
			if args != "" && args != "{}" {
				if err := json.Unmarshal(tc.Function.Arguments, &parameters); err != nil {
					logger().Infow("chat", "toolCallID", tc.ID, "args", args, "err", err)
					continue
				}
			}
			// 空参数时使用空 map
			if parameters == nil {
				parameters = make(map[string]any)
			}

			content, err := a.toolreg.Invoke(ctx, tc.Function.Name, parameters)
			if err != nil {
				logger().Infow("invokeTool fail", "toolCallName", tc.Function.Name, "err", err)
				continue
			}

			logger().Infow("invokeTool ok", "toolCallName", tc.Function.Name,
				"content", toolsvc.ResultLogs(content))
			messages = append(messages, llm.Message{
				Role:       llm.RoleTool,
				Content:    formatToolResult(content),
				ToolCallID: tc.ID,
			})
		}

		// 清除工具定义，避免死循环
		tools = nil
	}
}
