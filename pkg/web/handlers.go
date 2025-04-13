package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jpillora/eventsource"
	"github.com/marcsv/go-binder/binder"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sashabaranov/go-openai"

	"github.com/liut/morrigan/pkg/models/aigc"
	"github.com/liut/morrigan/pkg/models/qas"
	"github.com/liut/morrigan/pkg/services/mcputils"
	"github.com/liut/morrigan/pkg/services/stores"
	"github.com/liut/morrigan/pkg/settings"
)

func (s *server) getModels(w http.ResponseWriter, r *http.Request) {
	res, err := s.oc.ListModels(r.Context())
	if err != nil {
		apiFail(w, r, 400, err)
		return
	}
	apiOk(w, r, res)
}

// func (s *server) getModel(w http.ResponseWriter, r *http.Request) {
// 	res, err := s.oc.RetrieveModel(r.Context(), chi.URLParam(r, "id"))
// 	if err != nil {
// 		apiFail(w, r, 400, err)
// 		return
// 	}
// 	apiOk(w, r, res, 0)
// }

func (s *server) prepareChatRequest(ctx context.Context, prompt, csid string) *ChatCompletionRequest {
	cs := stores.NewConversation(csid)
	var messages []ChatCompletionMessage

	systemPrompt := dftSystemMsg
	if len(s.preset.SystemPrompt) > 0 {
		systemPrompt = s.preset.SystemPrompt
	}
	if settings.Current.DateInContext {
		systemPrompt = systemPrompt + "\n" + thisMoment()
	}
	messages = append(messages, ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemPrompt,
	})

	var matched int
	if len(s.tools) == 0 { // 没有工具，使用问答
		docs, err := stores.Sgt().Qa().MatchDocments(ctx, stores.MatchSpec{
			Question: prompt,
			Limit:    5,
		})
		if err == nil {
			logger().Infow("matches", "docs", len(docs), "prompt", prompt)
			for _, doc := range docs {
				matched++
				logger().Infow("hit", "id", doc.ID, "head", doc.Heading)
				messages = append(messages,
					ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: doc.Heading},
					ChatCompletionMessage{Role: openai.ChatMessageRoleAssistant, Content: doc.Content},
				)
			}
		} else {
			logger().Infow("match fail", "err", err)
		}
	}

	data, err := cs.ListHistory(ctx)
	if err == nil && len(data) > 0 {
		logger().Infow("found history", "size", len(data))
		data = data.RecentlyWithTokens(historyLimitToken)
		for _, hi := range data {
			if hi.ChatItem != nil {
				if len(hi.ChatItem.User) > 0 {
					messages = append(messages, ChatCompletionMessage{
						Role: openai.ChatMessageRoleUser, Content: hi.ChatItem.User})
				}
				if len(hi.ChatItem.Assistant) > 0 {
					messages = append(messages, ChatCompletionMessage{
						Role: openai.ChatMessageRoleAssistant, Content: hi.ChatItem.Assistant})
				}
			}
		}
	}
	ccr := new(ChatCompletionRequest)
	ccr.Model = s.cmodel
	ccr.cs = cs

	if len(s.tools) > 0 {
		// 为LLM转换工具结构
		if tools, err := mcputils.MCPToolsToOpenAITools(s.tools); err == nil {
			ccr.Tools = tools
			toolsPrompt := dftToolsMsg
			if len(s.preset.ToolsPrompt) > 0 {
				toolsPrompt = s.preset.ToolsPrompt
			}
			messages = append(messages, ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: toolsPrompt,
			})
		}
	}
	ccr.Messages = append(messages, ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompt,
	})
	ccr.hi = &aigc.HistoryItem{
		Time: time.Now().Unix(),
		ChatItem: &aigc.HistoryChatItem{
			User: prompt,
		},
	}

	return ccr
}

func (s *server) postChat(w http.ResponseWriter, r *http.Request) {
	var param ChatRequest
	if err := binder.BindBody(r, &param); err != nil {
		apiFail(w, r, 400, err)
		return
	}
	isProcess := strings.HasSuffix(r.URL.Path, "-process")
	isSSE := param.Stream || strings.HasSuffix(r.URL.Path, "-sse")
	isStream := param.Stream || isSSE || isProcess
	var csid string
	if len(param.ConversationID) > 0 {
		csid = param.ConversationID
	} else {
		// for github.com/Chanzhaoyu/chatgpt-web only
		csid = param.Options.ConversationId
	}
	ccr := s.prepareChatRequest(r.Context(), param.Prompt, csid)

	ccr.Stream = isStream
	if settings.Current.AuthRequired {
		if user, ok := UserFromContext(r.Context()); ok {
			ccr.User = "mog-uid-" + user.UID
		}
	}

	ccr.isSSE = isSSE

	// logger().Infow("chat", "id", csid, "prompt", param.Prompt, "stream", isStream)
	logger().Infow("chat", "csid", csid, "msgs", len(ccr.Messages), "prompt", param.Prompt, "ip", r.RemoteAddr)

	if ccr.Stream {
		answer, toolCalls := s.chatStreamResponse(ccr, w, r)
		if len(toolCalls) > 0 {
			var hasToolCall bool
			ccr.Messages = append(ccr.Messages, openai.ChatCompletionMessage{
				Role:      openai.ChatMessageRoleAssistant,
				ToolCalls: toolCalls,
			})
			for _, tc := range toolCalls {
				logger().Infow("chat", "toolCall", tc)
				if tc.Type == "function" {
					logger().Infow("chat", "functionCall", tc)
					var parameters map[string]any
					if err := json.Unmarshal([]byte(tc.Function.Arguments), &parameters); err != nil {
						logger().Infow("chat", "functionCall", tc, "err", err)
						continue
					}
					switch tc.Function.Name {
					case ToolNameKBSearch:
						content, err := s.callKBSearch(r.Context(), parameters)
						if err != nil {
							logger().Infow("chat", "functionCall", tc, "err", err)
						} else {
							ccr.Messages = append(ccr.Messages, mcpContentToChatMessage(tc.ID, content))
							hasToolCall = true
						}
					case ToolNameKBCreate:
						content, err := s.callKBCreate(r.Context(), parameters)
						if err != nil {
							logger().Infow("chat", "functionCall", tc, "err", err)
						} else {
							ccr.Messages = append(ccr.Messages, mcpContentToChatMessage(tc.ID, content))
							hasToolCall = true
						}
					}
				}
			}
			if hasToolCall {
				// 继续调用
				answer, _ = s.chatStreamResponse(ccr, w, r)
			}
		}
		if len(answer) > 0 {
			ccr.hi.ChatItem.Assistant = answer
			_ = ccr.cs.AddHistory(r.Context(), ccr.hi)

			if settings.Current.QAChatLog {
				in := qas.ChatLogBasic{
					ChatID:   ccr.cs.GetOID(),
					Question: param.Prompt,
					Answer:   answer,
				}
				ip, _, _ := strings.Cut(r.RemoteAddr, ":")
				in.MetaAddKVs("ip", ip)
				_, err := stores.Sgt().Qa().CreateChatLog(r.Context(), in)
				if err != nil {
					logger().Infow("save chat log fail", "err", err)
				}
			}
		}

		return
	}

	res, err := s.oc.CreateChatCompletion(r.Context(), ccr.ChatCompletionRequest)
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

func (s *server) chatStreamResponse(ccr *ChatCompletionRequest, w http.ResponseWriter, r *http.Request) (answer string, toolCalls []ToolCall) {
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
	ccs, err := s.oc.CreateChatCompletionStream(r.Context(), ccr.ChatCompletionRequest)
	if err != nil {
		logger().Infow("call chat stream fail", "err", err)
		apiFail(w, r, 500, err)
		return
	}
	defer ccs.Close()

	var cm ChatMessage
	if !ccr.isSSE {
		// for github.com/Chanzhaoyu/chatgpt-web chat-process only
		cm.ConversationId = cm.ID
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
			answer += cm.Delta
			if len(cm.ToolCalls) > 0 {
				if toolCalls == nil {
					toolCalls = cm.ToolCalls
				} else {
					for i, tc := range cm.ToolCalls {
						if len(toolCalls) <= i {
							toolCalls = append(toolCalls, tc)
						} else {
							toolCalls[i].Function.Arguments += tc.Function.Arguments
						}
					}
				}
			}

			cm.FinishReason = finishReason
			if ccr.isSSE {
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
				logger().Infow("stream done", "reason", finishReason, "tc", toolCalls, "answer", answer)
				if len(toolCalls) == 0 {
					finalFinishFn(finishReason)
				}
				break
			}
		}
	}
	logger().Infow("ccs recv done", "answer", len(answer))
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

func (s *server) postCompletions(w http.ResponseWriter, r *http.Request) {
	var param CompletionRequest
	if err := binder.BindBody(r, &param); err != nil {
		apiFail(w, r, 400, err)
		return
	}

	param.cs = stores.NewConversation(param.ConversationID)

	header := "Answer the question as truthfully as possible using the provided context."

	if s.preset.Completion != nil {
		header = s.preset.Completion.Header
		if !settings.Current.QAEmbedding && len(s.preset.Completion.Model) > 0 {
			param.Model = s.preset.Completion.Model
		}
	}
	if len(s.preset.Stop) > 0 {
		param.Stop = s.preset.Stop
	}
	var prompt string
	if s, ok := param.Prompt.(string); ok {
		prompt = s
	} else {
		apiFail(w, r, 400, "invalid prompt")
		return
	}

	spec := stores.MatchSpec{}
	spec.Question = prompt
	prompt, err := stores.Sgt().Qa().ConstructPrompt(r.Context(), spec)
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
		ccs, err := s.oc.CreateCompletionStream(r.Context(), param.CompletionRequest)
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

	res, err := s.oc.CreateCompletion(r.Context(), param.CompletionRequest)
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

func (s *server) getWelcome(w http.ResponseWriter, r *http.Request) {
	msg := new(aigc.Message)

	if s.preset.Welcome != nil {
		msg.Content = s.preset.Welcome.Content
	} else {
		msg.Content = welcomeText
	}

	cs := stores.NewConversation("")
	msg.ID = cs.GetID()
	apiOk(w, r, msg)
}

func (s *server) getHistory(w http.ResponseWriter, r *http.Request) {
	cid := chi.URLParam(r, "cid")
	cs := stores.NewConversation(cid)
	data, err := cs.ListHistory(r.Context())
	if err != nil {
		apiFail(w, r, 500, err)
		return
	}
	apiOk(w, r, data, 0)
}

func (s *server) getTools(w http.ResponseWriter, r *http.Request) {
	apiOk(w, r, s.tools, 0)
}

func (s *server) callKBSearch(ctx context.Context, args map[string]any) (mcp.Content, error) {
	logger().Infow("mcp call qa search", "args", args)
	subjectArg, ok := args["subject"]
	if !ok {
		return nil, errors.New("missing required argument: subject")
	}
	subject, ok := subjectArg.(string)
	if !ok {
		return nil, errors.New("subject argument must be a string")
	}

	docs, err := s.sto.Qa().MatchDocments(ctx, stores.MatchSpec{
		Question:     subject,
		Limit:        5,
		SkipKeywords: true,
	})
	if err != nil {
		return nil, err
	}
	logger().Infow("matched", "docs", len(docs))
	if len(docs) == 0 {
		return mcp.NewTextContent("No relevant information found"), nil
	}

	return mcp.NewTextContent(docs.MarkdownText()), nil
}

func (s *server) callKBCreate(ctx context.Context, args map[string]any) (mcp.Content, error) {
	user, ok := stores.UserFromContext(ctx)
	if !ok {
		return nil, errors.New("only admin can create document")
	} else {
		logger().Infow("mcp call qa create", "args", args, "user", user)
	}
	titleArg, ok := args["title"]
	if !ok {
		return nil, errors.New("missing required argument: title")
	}
	headingArg, ok := args["heading"]
	if !ok {
		return nil, errors.New("missing required argument: heading")
	}
	contentArg, ok := args["content"]
	if !ok {
		return nil, errors.New("missing required argument: content")
	}
	docBasic := qas.DocumentBasic{
		Title:   titleArg.(string),
		Heading: headingArg.(string),
		Content: contentArg.(string),
	}
	docBasic.MetaAddKVs("creator", user.Name)
	obj, err := s.sto.Qa().CreateDocument(ctx, docBasic)
	if err != nil {
		logger().Infow("create document fail", "title", docBasic.Title, "heading", docBasic.Heading,
			"content", len(docBasic.Content), "err", err)
		return mcp.NewTextContent(fmt.Sprintf(
			"Create KB document with title %q and heading %q is failed, %s", docBasic.Title, docBasic.Heading, err)), nil
	}
	return mcp.NewTextContent(fmt.Sprintf("Created KB document with ID %s", obj.StringID())), nil
}

func mcpContentToChatMessage(id string, mc mcp.Content) ChatCompletionMessage {
	// 将 mcp.Content 转换为 JSON 字符串
	content := ""
	if mc != nil {
		if b, err := json.Marshal(mc); err == nil {
			content = string(b)
		}
	}

	return ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    content,
		ToolCallID: id,
	}
}
