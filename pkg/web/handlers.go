package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jpillora/eventsource"
	"github.com/marcsv/go-binder/binder"
	"github.com/sashabaranov/go-openai"

	"github.com/liut/morrigan/pkg/models/aigc"
	"github.com/liut/morrigan/pkg/settings"
	"github.com/liut/morrigan/pkg/sevices/stores"
)

type ChatCompletionMessage = openai.ChatCompletionMessage

type CompletionRequest = openai.CompletionRequest

type ChatCompletionRequest struct {
	openai.ChatCompletionRequest

	isSSE bool
	cs    stores.Conversation
	hi    *aigc.HistoryItem
}

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

type ChatRequest struct {
	Prompt          string `json:"prompt"`
	ConversationID  string `json:"csid"`
	ParentMessageID string `json:"pmid"`
	Stream          bool   `json:"stream"`

	// for github.com/Chanzhaoyu/chatgpt-web only
	Options struct {
		ConversationId string `json:"conversationId,omitempty"`
	}
}

/*
// chatgpt-web:

	interface ConversationResponse {
		conversationId: string
		detail: {
			choices: { finish_reason: string; index: number; logprobs: any; text: string }[]
			created: number
			id: string
			model: string
			object: string
			usage: { completion_tokens: number; prompt_tokens: number; total_tokens: number }
		}
		id: string
		parentMessageId: string
		role: string
		text: string
	}
*/
type ChatCompletionChoice struct {
	FinishRsason string `json:"finishReason,omitempty"`
	Index        int    `json:"index"`
	Text         string `json:"text"`
}
type ConversationResponse struct {
	ConversationID  string `json:"csid"`
	ParentMessageID string `json:"pmid"`
	Detail          struct {
		Choices []ChatCompletionChoice `json:"choices"`

		Created int64  `json:"created"`
		ID      string `json:"id"`
		Model   string `json:"model"`
		Object  string `json:"object"`
		Usage   struct {
			CompletionTokens int `json:"completionTokens"`
			PromptTokens     int `json:"promptTokens"`
			TotalTokens      int `json:"totalTokens"`
		} `json:"usage"`
	} `json:"detail"`
}

type ChatMessage struct {
	ID    string `json:"id"`
	Delta string `json:"delta,omitempty"`
	Text  string `json:"text"`
	Role  string `json:"role,omitempty"`
	Name  string `json:"name,omitempty"`

	// for github.com/Chanzhaoyu/chatgpt-web only
	ConversationId string `json:"conversationId,omitempty"`
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

	var cs stores.Conversation
	if len(param.ConversationID) > 0 {
		cs = stores.NewConversation(param.ConversationID)
	} else {
		// for github.com/Chanzhaoyu/chatgpt-web only
		cs = stores.NewConversation(param.Options.ConversationId)
	}

	var messages []ChatCompletionMessage

	if s.preset != nil {
		if s.preset.Welcome != nil {
			messages = append(messages, ChatCompletionMessage{
				Role: openai.ChatMessageRoleAssistant, Content: s.preset.Welcome.Content})
		}
		for _, msg := range s.preset.Messages {
			messages = append(messages, ChatCompletionMessage{Role: msg.Role, Content: msg.Content})
		}
	}

	data, err := cs.ListHistory(r.Context())
	if err == nil {
		for _, hi := range data {
			if hi.ChatItem != nil {
				if len(hi.ChatItem.User) > 0 {
					messages = append(messages, ChatCompletionMessage{
						Role: openai.ChatMessageRoleUser, Content: hi.ChatItem.User})
				} else if len(hi.ChatItem.Assistant) > 0 {
					messages = append(messages, ChatCompletionMessage{
						Role: openai.ChatMessageRoleAssistant, Content: hi.ChatItem.Assistant})
				}
			}
		}
	}
	messages = append(messages, ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: param.Prompt,
	})
	logger().Infow("chat", "cid", param.ConversationID, "msgs", len(messages))
	ccr := new(ChatCompletionRequest)
	ccr.Model = openai.GPT3Dot5Turbo
	ccr.Messages = messages
	ccr.Stream = isStream
	if settings.Current.AuthRequired {
		if user, ok := UserFromContext(r.Context()); ok {
			ccr.User = "mog-uid-" + user.UID
		}
	}

	ccr.isSSE = isSSE
	ccr.cs = cs
	ccr.hi = &aigc.HistoryItem{
		Time: time.Now().Unix(),
		ChatItem: &aigc.HistoryChatItem{
			User: param.Prompt,
		},
	}

	logger().Debugw("chat", "req", &ccr)

	if ccr.Stream {
		s.chatStreamResponse(ccr, w, r)
		return
	}
	res, err := s.oc.CreateChatCompletion(r.Context(), ccr.ChatCompletionRequest)
	if err != nil {
		apiFail(w, r, 500, err)
		return
	}
	logger().Infow("chat", "res", &res)
	var cr ConversationResponse
	cr.ConversationID = cs.GetID()
	cr.Detail.Created = res.Created
	cr.Detail.ID = res.ID
	cr.Detail.Model = res.Model
	cr.Detail.Object = res.Object
	cr.Detail.Choices = []ChatCompletionChoice{{
		FinishRsason: res.Choices[0].FinishReason,
		Index:        res.Choices[0].Index,
		Text:         res.Choices[0].Message.Content,
	}}
	cr.Detail.Usage.CompletionTokens = res.Usage.CompletionTokens
	cr.Detail.Usage.PromptTokens = res.Usage.PromptTokens
	cr.Detail.Usage.TotalTokens = res.Usage.TotalTokens
	render.JSON(w, r, cr)
}

func (s *server) chatStreamResponse(ccr *ChatCompletionRequest, w http.ResponseWriter, r *http.Request) {

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

	ccs, err := s.oc.CreateChatCompletionStream(r.Context(), ccr.ChatCompletionRequest)
	if err != nil {
		logger().Infow("call chat stream fail", "err", err)
		apiFail(w, r, 500, err)
		return
	}
	defer ccs.Close()

	var cm ChatMessage
	cm.ID = ccr.cs.GetID()
	if !ccr.isSSE {
		// for github.com/Chanzhaoyu/chatgpt-web only
		cm.ConversationId = cm.ID
	}
	var answer string
	for {
		var wrote bool
		ccsr, err := ccs.Recv()
		if errors.Is(err, io.EOF) {
			logger().Debug("ccs recv end")
			if ccr.isSSE {
				if err = eventsource.WriteEvent(w, eventsource.Event{
					ID:   ccsr.ID,
					Data: []byte("[DONE]"),
				}); err == nil {
					wrote = true
				} else {
					logger().Infow("eventsource write fail", "err", err)
				}
			}
			break
		}
		if err != nil {
			logger().Infow("ccs recv fail", "err", err)
			break
		}
		// logger().Debugw("ccs recv", "res", &ccsr)
		// if wrote>0 {
		// 	w.Write([]byte("\n"))
		// }
		if len(ccsr.Choices) > 0 {
			cm.Delta = ccsr.Choices[0].Delta.Content
			answer += cm.Delta
			// logger().Debugw("cm", "delta", cm.Delta)
			if ccr.isSSE {
				b, err := json.Marshal(&cm)
				if err != nil {
					logger().Infow("json marshal fail", "cm", &cm, "err", err)
					break
				}
				if err = eventsource.WriteEvent(w, eventsource.Event{
					ID:   ccsr.ID,
					Data: b,
				}); err == nil {
					wrote = true
				} else {
					logger().Infow("eventsource write fail", "err", err)
				}
			} else {
				cm.Text += cm.Delta
				if err = json.NewEncoder(w).Encode(&cm); err != nil {
					logger().Infow("json enc fail", "err", err)
					break
				}
			}
		}
		if !wrote {
			flusher.Flush()
		}

	}
	ccr.hi.ChatItem.Assistant = answer
	ccr.cs.AddHistory(r.Context(), ccr.hi)
	logger().Infow("ccs recv done", "answer", len(answer))
}

func (s *server) postCompletions(w http.ResponseWriter, r *http.Request) {
	var param CompletionRequest
	if err := binder.BindBody(r, &param); err != nil {
		apiFail(w, r, 400, err)
		return
	}
	res, err := s.oc.CreateCompletion(r.Context(), param)
	if err != nil {
		apiFail(w, r, 400, err)
		return
	}
	apiOk(w, r, res, 0)
}

const (
	welcomeText = "Hello, I am your virtual assistant. How can I help you?"
)

func (s *server) getWelcome(w http.ResponseWriter, r *http.Request) {
	msg := new(aigc.Message)

	if s.preset != nil && s.preset.Welcome != nil {
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
