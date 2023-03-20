package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/render"
	"github.com/marcsv/go-binder/binder"
	"github.com/sashabaranov/go-openai"
)

type ChatCompletionRequest = openai.ChatCompletionRequest
type CompletionRequest = openai.CompletionRequest

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
}

/*
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
}

func (s *server) postChat(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		apiFail(w, r, 401, "not login")
	}
	var param ChatRequest
	if err := binder.BindBody(r, &param); err != nil {
		apiFail(w, r, 400, err)
		return
	}
	var isStream bool
	if strings.HasSuffix(r.URL.Path, "-process") {
		isStream = true
	}
	ccr := ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: param.Prompt,
			},
		},
		Stream: isStream,
		User:   "mog-uid-" + user.UID,
	}
	logger().Infow("chat", "req", &ccr)
	if isStream {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Add("Content-type", "application/octet-stream")

		ccs, err := s.oc.CreateChatCompletionStream(r.Context(), ccr)
		if err != nil {
			apiFail(w, r, 500, err)
			return
		}
		defer ccs.Close()

		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		var cm ChatMessage
		var wrote int
		for {
			ccsr, err := ccs.Recv()
			if errors.Is(err, io.EOF) {
				logger().Debug("ccs recv end")
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
			cm.ID = ccsr.ID
			if len(ccsr.Choices) > 0 {
				cm.Delta = ccsr.Choices[0].Delta.Content
				cm.Text += cm.Delta
				if err = json.NewEncoder(w).Encode(&cm); err != nil {
					logger().Infow("json enc fail", "err", err)
					break
				}
			}

			wrote++
			flusher.Flush()
		}
		logger().Infow("ccs recv done", "wrote", wrote)
		return

	}
	res, err := s.oc.CreateChatCompletion(r.Context(), ccr)
	if err != nil {
		apiFail(w, r, 500, err)
		return
	}
	logger().Infow("chat", "res", &res)
	var cr ConversationResponse
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
