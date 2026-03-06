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

	"github.com/liut/morrigan/pkg/models/aigc"
	"github.com/liut/morrigan/pkg/models/cob"
	"github.com/liut/morrigan/pkg/models/mcps"
	"github.com/liut/morrigan/pkg/services/llm"
	"github.com/liut/morrigan/pkg/services/stores"
	"github.com/liut/morrigan/pkg/settings"
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
	messages []llm.Message
	tools    []llm.ToolDefinition
	isSSE    bool
	cs       stores.Conversation
	hi       *aigc.HistoryItem
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

	if len(a.toolreg.ToolsFor(ctx)) == 0 { // 没有工具，使用问答
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
		logger().Infow("found history", "size", len(data))
		data = data.RecentlyWithTokens(historyLimitToken)
		for i, hi := range data {
			if hi.ChatItem != nil {
				if len(hi.ChatItem.User) > 0 {
					messages = append(messages, llm.Message{
						Role: llm.RoleUser, Content: hi.ChatItem.User})
				}
				if len(hi.ChatItem.Assistant) > 0 {
					messages = append(messages, llm.Message{
						Role: llm.RoleAssistant, Content: hi.ChatItem.Assistant})
				}
				// skip the last one
				if i == len(data)-1 && param.Regenerate {
					break
				}
			}
		}
	}

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
		res := a.chatStreamResponse(ccr, w, r)
		logger().Infow("stream response", "answer_len", len(res.answer), "toolCalls_len", len(res.toolCalls))
		if len(res.toolCalls) > 0 {
			var hasToolCall bool
			ccr.messages = append(ccr.messages, llm.Message{
				Role:      llm.RoleAssistant,
				ToolCalls: res.toolCalls,
			})
			for _, tc := range res.toolCalls {
				// 避免直接记录 tc（包含 json.RawMessage），只记录安全字段
				logger().Infow("chat", "toolCallID", tc.ID, "toolCallType", tc.Type, "toolCallName", tc.Function.Name)
				if tc.Type == "function" {
					// 检查 Arguments 是否为有效 JSON（流式响应中可能不完整）
					args := string(tc.Function.Arguments)
					if args == "" || args == "{}" {
						logger().Infow("chat", "toolCallID", tc.ID, "err", "empty arguments")
						continue
					}
					// 尝试解析，失败则跳过
					var parameters map[string]any
					if err := json.Unmarshal(tc.Function.Arguments, &parameters); err != nil {
						logger().Infow("chat", "toolCallID", tc.ID, "args", args, "err", err)
						continue
					}
					content, err := a.toolreg.Invoke(r.Context(), tc.Function.Name, parameters)
					if err != nil {
						logger().Infow("invokeTool fail", "toolCallName", tc.Function.Name, "err", err)
					} else {
						logger().Debugw("invokeTool ok", "toolCallName", tc.Function.Name)
						ccr.messages = append(ccr.messages, llm.Message{
							Role:       llm.RoleTool,
							Content:    formatToolResult(content),
							ToolCallID: tc.ID,
						})
						hasToolCall = true
					}
				}
			}
			if hasToolCall {
				// 继续调用，清除工具以避免死循环
				tools := ccr.tools
				ccr.tools = nil
				res = a.chatStreamResponse(ccr, w, r)
				ccr.tools = tools
			}
		}
		if len(res.answer) > 0 {
			ccr.hi.ChatItem.Assistant = res.answer
			_ = ccr.cs.AddHistory(r.Context(), ccr.hi)

			if settings.Current.QAChatLog {
				in := cob.ChatLogBasic{
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

	result, err := a.llm.Chat(r.Context(), ccr.messages, ccr.tools)
	if err != nil {
		apiFail(w, r, 500, err)
		return
	}
	logger().Infow("chat", "res", result)

	if param.Full {
		var cr ConversationResponse
		cr.ConversationID = ccr.cs.GetID()
		cr.Detail.Model = a.cmodel
		cr.Detail.Object = "chat.completion"
		if result.Content != "" {
			cr.Detail.Choices = []ChatCompletionChoice{{
				FinishReason: "stop",
				Index:        0,
				Text:         result.Content,
			}}
		}
		cr.Detail.Usage.CompletionTokens = result.Usage.CompletionTokens
		cr.Detail.Usage.PromptTokens = result.Usage.PromptTokens
		cr.Detail.Usage.TotalTokens = result.Usage.TotalTokens
		render.JSON(w, r, &cr)
		return
	}

	var cm ChatMessage
	cm.ID = ccr.cs.GetID()
	cm.Text = result.Content
	if result.HasToolCalls() {
		cm.FinishReason = "tool_calls"
		cm.ToolCalls = convertToolCallsForJSON(result.ToolCalls)
	}
	render.JSON(w, r, &cm)
}

func (a *api) chatStreamResponse(ccr *chatRequest, w http.ResponseWriter, r *http.Request) (res ChatResponse) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	if ccr.isSSE {
		w.Header().Set("Content-Type", "text/event-stream")
	} else {
		w.Header().Add("Content-type", "application/octet-stream")
	}
	w.Header().Add("Conversation-ID", ccr.cs.GetID())

	stream, err := a.llm.StreamChat(r.Context(), ccr.messages, ccr.tools)
	if err != nil {
		logger().Infow("call chat stream fail", "err", err)
		apiFail(w, r, 500, err)
		return
	}

	var cm ChatMessage
	if !ccr.isSSE {
		// for github.com/Chanzhaoyu/chatgpt-web chat-process only
		cm.ConversationID = cm.ID
	}

	var chunkIdx int

	for result := range stream {
		chunkIdx++
		if result.Error != nil {
			logger().Infow("stream error", "err", result.Error)
			break
		}

		var wrote bool

		// 处理内容
		cm.Delta = result.Delta
		res.answer += result.Delta
		// LLM 层已经返回累积的 ToolCalls，直接使用
		if len(result.ToolCalls) > 0 {
			cm.ToolCalls = convertToolCallsForJSON(result.ToolCalls)
		}

		if ccr.isSSE {
			if result.Done {
				cm.ConversationID = ccr.cs.GetID()
				cm.FinishReason = result.FinishReason
				_ = writeEvent(w, strconv.Itoa(chunkIdx), &cm)
			} else {
				if wrote = writeEvent(w, strconv.Itoa(chunkIdx), &cm); !wrote {
					break
				}
			}
		} else {
			cm.Text += result.Delta
			if err = json.NewEncoder(w).Encode(&cm); err != nil {
				logger().Infow("json encode fail", "err", err)
				break
			}
		}
		flusher.Flush()

		if result.Done {
			// 保存最终的 toolCalls 供后续处理
			res.toolCalls = result.ToolCalls
			break
		}
	}
	logger().Infow("llm stream done", "answer", len(res.answer))
	return
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

// formatToolResult 将工具结果转换为 JSON 字符串
func formatToolResult(result map[string]any) string {
	if result == nil {
		return ""
	}
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
