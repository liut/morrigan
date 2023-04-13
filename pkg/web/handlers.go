package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/liut/morrigan/pkg/models/qas"
	"github.com/liut/morrigan/pkg/services/stores"
	"github.com/liut/morrigan/pkg/settings"
)

const (
	esDone = "[DONE]"

	tplSystemMsg = "You are a helpful assistant.\nCurrent date: %s"
)

func today() string {
	return time.Now().Format("2006-01-02")
}

func getDefaultSystemMsg() string {
	return fmt.Sprintf(tplSystemMsg, today())
}

type ChatCompletionMessage = openai.ChatCompletionMessage

type CompletionRequest struct {
	openai.CompletionRequest

	ConversationID string `json:"csid"`

	cs stores.Conversation
	hi *aigc.HistoryItem
}

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
	Full            bool   `json:"full,omitempty"`

	// deprecated: for github.com/Chanzhaoyu/chatgpt-web only
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
	ID    string `json:"id,omitempty"`
	Delta string `json:"delta,omitempty"`
	Text  string `json:"text,omitempty"`
	Role  string `json:"role,omitempty"`
	Name  string `json:"name,omitempty"`

	FinishRsason string `json:"finishReason,omitempty"`

	// for github.com/Chanzhaoyu/chatgpt-web only
	ConversationId string `json:"conversationId,omitempty"`
}

type CompletionMessage struct {
	ID    string `json:"id,omitempty"`
	Delta string `json:"delta,omitempty"`
	Text  string `json:"text,omitempty"`
	Time  int64  `json:"ts,omitempty"`

	FinishRsason string `json:"finishReason,omitempty"`
}

func (s *server) prepareChatRequest(ctx context.Context, prompt, csid string) *ChatCompletionRequest {
	cs := stores.NewConversation(csid)
	var messages []ChatCompletionMessage

	var hasSystem bool
	if s.preset.Welcome != nil {
		messages = append(messages, ChatCompletionMessage{
			Role: openai.ChatMessageRoleAssistant, Content: s.preset.Welcome.Content})
	}
	for _, msg := range s.preset.BeforeQA {
		if msg.Role == openai.ChatMessageRoleSystem {
			hasSystem = true
			msg.Content += "\nCurrent date: " + today()
		}
		messages = append(messages, ChatCompletionMessage{Role: msg.Role, Content: msg.Content})
	}

	var matched int
	if settings.Current.EmbeddingQA {
		docs, err := stores.Sgt().Qa().MatchDocments(ctx, stores.MatchSpec{
			Question: prompt,
			Limit:    5,
		})
		if err == nil {
			logger().Infow("matches", "docs", len(docs), "prompt", prompt)
			for _, doc := range docs {
				matched++
				logger().Infow("hit", "head", doc.Heading)
				messages = append(messages,
					ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: doc.Heading},
					ChatCompletionMessage{Role: openai.ChatMessageRoleAssistant, Content: doc.Content},
				)
			}
		} else {
			logger().Infow("match fail", "err", err)
		}
	}
	if matched == 0 {
		for _, msg := range s.preset.Messages {
			messages = append(messages, ChatCompletionMessage{Role: msg.Role, Content: msg.Content})
		}
	}

	for _, msg := range s.preset.AfterQA {
		if msg.Role == openai.ChatMessageRoleSystem {
			hasSystem = true
		}
		messages = append(messages, ChatCompletionMessage{Role: msg.Role, Content: msg.Content})
	}

	if !hasSystem {
		messages = append(messages, ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: getDefaultSystemMsg()})
	}

	const historyLimit = 3000

	data, err := cs.ListHistory(ctx)
	if err == nil {
		data = data.RecentlyWith(historyLimit)
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
	messages = append(messages, ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompt,
	})
	logger().Infow("chat", "cid", csid, "msgs", len(messages))
	ccr := new(ChatCompletionRequest)
	ccr.Model = openai.GPT3Dot5Turbo
	ccr.Messages = messages
	ccr.cs = cs
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

	logger().Debugw("chat", "req", &ccr)

	if ccr.Stream {
		answer := s.chatStreamResponse(ccr, w, r)
		if len(answer) > 0 {
			ccr.hi.ChatItem.Assistant = answer
			ccr.cs.AddHistory(r.Context(), ccr.hi)
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
				FinishRsason: res.Choices[0].FinishReason,
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
			cm.FinishRsason = res.Choices[0].FinishReason
		}
	}
	render.JSON(w, r, &cm)
}

func (s *server) chatStreamResponse(ccr *ChatCompletionRequest, w http.ResponseWriter, r *http.Request) (answer string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// w.Header().Set("Cache-Control", "no-cache")
	// w.Header().Set("Connection", "keep-alive")
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
		// for github.com/Chanzhaoyu/chatgpt-web chat-process only
		cm.ConversationId = cm.ID
	}
	var finishReason string
	finishFn := func(id string) {
		logger().Infow("finish", "id", id, "answer", answer, "reason", finishReason)
		if finishReason != "stop" {
			cm.FinishRsason = finishReason
		}
		if ccr.isSSE {
			cm.Text = answer // optional for chatgpt-web
			_ = writeEvent(w, id, esDone)
		} else {
			flusher.Flush()
		}
	}
	for {
		var wrote bool
		ccsr, err := ccs.Recv()
		if errors.Is(err, io.EOF) { // choices is nil at the moment
			logger().Debugw("ccs recv eof", "reason", finishReason)
			finishFn(ccsr.ID)
			break
		}
		if err != nil {
			logger().Infow("ccs recv fail", "err", err)
			break
		}
		if len(ccsr.Choices) > 0 {
			// logger().Debugw("recv", "cohoices", ccsr.Choices)
			finishReason = ccsr.Choices[0].FinishReason
			if len(finishReason) > 0 {
				finishFn(ccsr.ID)
				break
			}
			cm.Delta = ccsr.Choices[0].Delta.Content
			answer += cm.Delta
			if ccr.isSSE {
				if wrote = writeEvent(w, ccsr.ID, &cm); !wrote {
					break
				}
			} else {
				cm.Text += cm.Delta
				if err = json.NewEncoder(w).Encode(&cm); err != nil {
					logger().Infow("json enc fail", "err", err)
					break
				}
				flusher.Flush()
			}
		}
	}
	logger().Infow("ccs recv done", "answer", len(answer))
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

	var useFineTune bool
	if s.preset.Completion != nil {
		header = s.preset.Completion.Header
		if len(s.preset.Completion.Model) > 0 {
			param.Model = s.preset.Completion.Model
			useFineTune = true
		}
	}
	var prompt string
	if s, ok := param.Prompt.(string); ok {
		prompt = s
	} else {
		apiFail(w, r, 400, "invalid prompt")
		return
	}
	if useFineTune {
		param.Prompt = prompt + "\n" + qas.PrefixA
	} else {
		spec := stores.MatchSpec{}
		spec.Question = prompt
		prompt, err := stores.Sgt().Qa().ConstructPrompt(r.Context(), spec)
		if err != nil {
			apiFail(w, r, 503, err)
			return
		}
		param.Prompt = header + "\n\nContext:\n" + prompt + "\n" + qas.PrefixA

	}

	if len(param.Model) == 0 {
		param.Model = openai.GPT3TextDavinci003
	}

	param.MaxTokens = 1024
	logger().Infow("completion", "param", &param)

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

		var answer string
		var finishReason string
		finishFn := func(id string) {
			logger().Infow("finish", "id", id, "answer", answer, "reason", finishReason)
			if finishReason != "stop" {
				cm.FinishRsason = finishReason
			}
			cm.Text = answer // optional for chatgpt-web
			_ = writeEvent(w, id, esDone)
		}
		for {
			var wrote bool
			ccsr, err := ccs.Recv()
			if errors.Is(err, io.EOF) {
				logger().Debugw("ccs recv eof", "reason", finishReason)
				finishFn(ccsr.ID)
				break
			}

			if err != nil {
				logger().Infow("ccs recv fail", "err", err)
				break
			}

			if len(ccsr.Choices) > 0 {
				// logger().Debugw("recv", "cohoices", ccsr.Choices)
				finishReason = ccsr.Choices[0].FinishReason
				if len(finishReason) > 0 {
					finishFn(ccsr.ID)
					break
				}
				cm.Delta = ccsr.Choices[0].Text
				answer += cm.Delta
				if wrote = writeEvent(w, ccsr.ID, &cm); !wrote {
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

const (
	welcomeText = "Hello, I am your virtual assistant. How can I help you?"
)

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
