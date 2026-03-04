package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jpillora/eventsource"
	"github.com/marcsv/go-binder/binder"
	"github.com/sashabaranov/go-openai"

	"github.com/liut/morrigan/pkg/models/aigc"
	"github.com/liut/morrigan/pkg/models/cob"
	"github.com/liut/morrigan/pkg/services/mcputils"
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
	regHI(true, "POST", "/completions", "", func(a *api) http.HandlerFunc {
		return a.postCompletions
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

func (a *api) prepareChatRequest(ctx context.Context, param *ChatRequest) *ChatCompletionRequest {
	cs := stores.NewConversation(ctx, param.GetConversionID())
	var messages []ChatCompletionMessage

	systemPrompt := dftSystemMsg
	if len(a.preset.SystemPrompt) > 0 {
		systemPrompt = a.preset.SystemPrompt
	}
	if settings.Current.DateInContext {
		systemPrompt = systemPrompt + "\n" + thisMoment()
	}
	messages = append(messages, ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemPrompt,
	})

	if len(a.toolreg.Tools()) == 0 { // 没有工具，使用问答
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
			messages = append(messages, ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
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
					messages = append(messages, ChatCompletionMessage{
						Role: openai.ChatMessageRoleUser, Content: hi.ChatItem.User})
				}
				if len(hi.ChatItem.Assistant) > 0 {
					messages = append(messages, ChatCompletionMessage{
						Role: openai.ChatMessageRoleAssistant, Content: hi.ChatItem.Assistant})
				}
				// skip the last one
				if i == len(data)-1 && param.Regenerate {
					break
				}
			}
		}
	}
	ccr := new(ChatCompletionRequest)
	ccr.Model = a.cmodel
	ccr.cs = cs

	if len(a.toolreg.Tools()) > 0 {
		// 为LLM转换工具结构
		if tools, err := mcputils.MCPToolsToOpenAITools(a.toolreg.ToolsFor(ctx)); err == nil {
			ccr.Tools = tools
			toolsPrompt := dftToolsMsg
			if len(a.preset.ToolsPrompt) > 0 {
				toolsPrompt = a.preset.ToolsPrompt
			}
			messages = append(messages, ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: toolsPrompt,
			})
		}
	}
	ccr.Messages = append(messages, ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: param.Prompt,
	})
	ccr.hi = &aigc.HistoryItem{
		Time: time.Now().Unix(),
		ChatItem: &aigc.HistoryChatItem{
			User: param.Prompt,
		},
	}

	return ccr
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

	ccr.Stream = isStream
	if settings.Current.AuthRequired {
		if user, ok := UserFromContext(r.Context()); ok {
			ccr.User = "mog-uid-" + user.UID
		}
	}

	ccr.isSSE = isSSE

	// logger().Infow("chat", "id", csid, "prompt", param.Prompt, "stream", isStream)
	logger().Infow("chat", "csid", param.GetConversionID(), "msgs", len(ccr.Messages), "prompt", param.Prompt, "ip", r.RemoteAddr)

	if ccr.Stream {
		res := a.chatStreamResponse(ccr, w, r)
		if len(res.toolCalls) > 0 {
			var hasToolCall bool
			ccr.Messages = append(ccr.Messages, openai.ChatCompletionMessage{
				Role:      openai.ChatMessageRoleAssistant,
				ToolCalls: res.toolCalls,
			})
			for _, tc := range res.toolCalls {
				logger().Infow("chat", "toolCall", tc)
				if tc.Type == "function" {
					logger().Infow("chat found function", "toolCall", tc)
					var parameters map[string]any
					if err := json.Unmarshal([]byte(tc.Function.Arguments), &parameters); err != nil {
						logger().Infow("chat", "toolCall", tc, "err", err)
						continue
					}
					content, err := a.toolreg.Invoke(r.Context(), tc.Function.Name, parameters)
					if err != nil {
						logger().Infow("invokeTool fail", "toolCall", tc, "err", err)
					} else {
						logger().Debugw("invokeTool ok", "toolCall", tc, "content", content)
						ccr.Messages = append(ccr.Messages, mcpContentToChatMessage(tc.ID, content))
						hasToolCall = true
					}
				}
			}
			if hasToolCall {
				// 继续调用
				ccr.Tools = nil // 清除工具，有时会导致死循环
				res = a.chatStreamResponse(ccr, w, r)
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

	res, err := a.oc.CreateChatCompletion(r.Context(), ccr.ChatCompletionRequest)
	if err != nil {
		apiFail(w, r, 500, err)
		return
	}
	logger().Infow("chat", "res", &res)

	if param.Full {
		var cr ConversationResponse
		cr.ConversationID = ccr.cs.GetID()
		cr.Detail.Created = res.Created
		cr.Detail.ID = res.ID
		cr.Detail.Model = res.Model
		cr.Detail.Object = res.Object
		if len(res.Choices) > 0 {
			cr.Detail.Choices = []ChatCompletionChoice{{
				FinishReason: string(res.Choices[0].FinishReason),
				Index:        res.Choices[0].Index,
				Text:         res.Choices[0].Message.Content,
			}}
		}

		cr.Detail.Usage.CompletionTokens = res.Usage.CompletionTokens
		cr.Detail.Usage.PromptTokens = res.Usage.PromptTokens
		cr.Detail.Usage.TotalTokens = res.Usage.TotalTokens
		render.JSON(w, r, &cr)
		return
	}

	var cm ChatMessage
	cm.ID = ccr.cs.GetID()
	if len(res.Choices) > 0 {
		cm.Text = res.Choices[0].Message.Content
		if res.Choices[0].FinishReason != "stop" {
			cm.FinishReason = string(res.Choices[0].FinishReason)
		}
	}
	render.JSON(w, r, &cm)
}

func (a *api) chatStreamResponse(ccr *ChatCompletionRequest, w http.ResponseWriter, r *http.Request) (res ChatResponse) {
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

	logger().Debugw("ccs recv start", "ccr", ccr)
	ccs, err := a.oc.CreateChatCompletionStream(r.Context(), ccr.ChatCompletionRequest)
	if err != nil {
		logger().Infow("call chat stream fail", "err", err)
		apiFail(w, r, 500, err)
		return
	}
	defer ccs.Close()

	var cm ChatMessage
	if !ccr.isSSE {
		// for github.com/Chanzhaoyu/chatgpt-web chat-process only
		cm.ConversationID = cm.ID
	}

	var finishReason string

	finalFinishFn := func(reason string) {
		if ccr.isSSE {
			_ = writeEvent(w, strconv.Itoa(ccr.chunkIdx), esDone)
			flusher.Flush()
		}
		logger().Debugw("ccs recv done", "reason", reason)
	}

	for {
		ccr.chunkIdx++
		var wrote bool
		ccsr, err := ccs.Recv()
		if errors.Is(err, io.EOF) { // choices is nil at the moment
			logger().Debugw("ccs recv eof", "reason", finishReason)
			finalFinishFn("EOF")
			break
		}
		if err != nil {
			logger().Infow("ccs recv fail", "err", err)
			break
		}
		if len(ccsr.Choices) > 0 {
			cm.ToolCalls = ccsr.Choices[0].Delta.ToolCalls
			finishReason = string(ccsr.Choices[0].FinishReason)
			cm.Delta = ccsr.Choices[0].Delta.Content
			res.answer += cm.Delta
			if len(cm.ToolCalls) > 0 {
				if res.toolCalls == nil {
					res.toolCalls = cm.ToolCalls
				} else {
					for i, tc := range cm.ToolCalls {
						if len(res.toolCalls) <= i {
							res.toolCalls = append(res.toolCalls, tc)
						} else {
							res.toolCalls[i].Function.Arguments += tc.Function.Arguments
						}
					}
				}
			}

			cm.FinishReason = finishReason
			if ccr.isSSE {
				if len(finishReason) > 0 && finishReason == "stop" {
					cm.ConversationID = ccr.cs.GetID()
				}
				if wrote = writeEvent(w, strconv.Itoa(ccr.chunkIdx), &cm); !wrote {
					break
				}
			} else {
				cm.Text += cm.Delta
				if err = json.NewEncoder(w).Encode(&cm); err != nil {
					logger().Infow("json encode fail", "err", err)
					break
				}
			}
			flusher.Flush()
			if len(finishReason) > 0 {
				logger().Infow("stream done", "reason", finishReason, "tc", res.toolCalls, "answer", res.answer)
				if len(res.toolCalls) == 0 {
					finalFinishFn(finishReason)
				}
				break
			}
		}
	}
	logger().Infow("ccs recv done", "answer", len(res.answer))
	// TODO: 处理工具调用
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
// @Summary 文本补全
// @Accept json
// @Produce json
// @Param token header string false "登录票据凭证"
// @Param completion body CompletionRequest true "补全请求"
// @Success 200 {object} Done{result=CompletionMessage}
// @Failure 400 {object} Failure "请求或参数错误"
// @Failure 503 {object} Failure "服务端错误"
// @Router /api/completions [post]
func (a *api) postCompletions(w http.ResponseWriter, r *http.Request) {
	var param CompletionRequest
	if err := binder.BindBody(r, &param); err != nil {
		apiFail(w, r, 400, err)
		return
	}

	param.cs = stores.NewConversation(r.Context(), param.ConversationID)

	header := "Answer the question as truthfully as possible using the provided context."

	if a.preset.Completion != nil {
		header = a.preset.Completion.Header
		if !settings.Current.QAEmbedding && len(a.preset.Completion.Model) > 0 {
			param.Model = a.preset.Completion.Model
		}
	}
	if len(a.preset.Stop) > 0 {
		param.Stop = a.preset.Stop
	}
	var prompt string
	if s, ok := param.Prompt.(string); ok {
		prompt = s
	} else {
		apiFail(w, r, 400, "invalid prompt")
		return
	}

	prompt, err := a.sto.Cob().ConstructPrompt(r.Context(), stores.MatchSpec{
		Question: prompt,
	})
	if err != nil {
		apiFail(w, r, 503, err)
		return
	}
	param.Prompt = header + "\n\nContext:\n" + prompt

	if len(param.Model) == 0 {
		param.Model = openai.GPT3Dot5TurboInstruct
	}

	param.MaxTokens = 1024
	logger().Infow("completion", "param", &param, "csid", param.cs.GetID())
	w.Header().Add("Conversation-ID", param.cs.GetID())

	var cm CompletionMessage
	cm.ID = param.cs.GetID()

	if param.Stream {
		if _, ok := w.(http.Flusher); !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}
		ccs, err := a.oc.CreateCompletionStream(r.Context(), param.CompletionRequest)
		if err != nil {
			apiFail(w, r, 400, err)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")

		var idx int
		var answer string
		var finishReason string
		finishFn := func(reason string) {
			logger().Infow("finish", "reason", reason, "answer", answer)
			cm.Text = answer // optional for chatgpt-web
			_ = writeEvent(w, strconv.Itoa(idx), esDone)
		}
		for {
			idx++
			var wrote bool
			ccsr, err := ccs.Recv()
			if errors.Is(err, io.EOF) {
				logger().Debugw("ccs recv eof", "reason", finishReason)
				finishFn("EOF")
				break
			}

			if err != nil {
				logger().Infow("ccs recv fail", "err", err)
				break
			}

			if len(ccsr.Choices) > 0 {
				// logger().Debugw("recv", "cohoices", ccsr.Choices)
				finishReason = ccsr.Choices[0].FinishReason
				cm.Delta = ccsr.Choices[0].Text
				answer += cm.Delta
				cm.FinishReason = finishReason
				if wrote = writeEvent(w, strconv.Itoa(idx), &cm); !wrote {
					break
				}
				if len(finishReason) > 0 {
					finishFn(finishReason)
					break
				}
			}
		}
		logger().Infow("ccs recv done", "answer", len(answer))
		return
	}

	res, err := a.oc.CreateCompletion(r.Context(), param.CompletionRequest)
	if err != nil {
		logger().Infow("completion fail", "err", err)
		apiFail(w, r, 400, err)
		return
	}
	logger().Infow("completion done", "res", &res)
	cm = CompletionMessage{Time: res.Created}
	if len(res.Choices) > 0 {
		logger().Infow("got choices", "finish_reason", res.Choices[0].FinishReason)
		cm.Text = strings.TrimSpace(res.Choices[0].Text)
	}
	apiOk(w, r, cm, 0)
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

func mcpContentToChatMessage(id string, result map[string]any) ChatCompletionMessage {
	// 将 map[string]any 转换为 JSON 字符串
	content := ""
	if result != nil {
		if b, err := json.Marshal(result); err == nil {
			content = string(b)
		}
	}

	return ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    content,
		ToolCallID: id,
	}
}
