package stores

import (
	"net/http"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/liut/morrigan/pkg/settings"
)

var (
	ocEm *openai.Client // for embedding
	ocIt *openai.Client // for Interact
	ocSu *openai.Client // for summary
)

const (
	openaiTimeout = time.Second * 90
)

func init() {
	ocEm = NewOpenAIClient(
		settings.Current.Embedding.APIKey,
		settings.Current.Embedding.URL,
	)
	ocIt = NewOpenAIClient(
		settings.Current.Interact.APIKey,
		settings.Current.Interact.URL,
	)
	ocSu = NewOpenAIClient(
		settings.Current.Summarize.APIKey,
		settings.Current.Summarize.URL,
	)
}

func GetInteractAIClient() *openai.Client {
	return ocIt
}

func NewOpenAIClient(args ...string) *openai.Client {
	apikey := settings.Current.OpenAIAPIKey
	if len(args) > 0 && len(args[0]) > 0 {
		apikey = args[0]
	}
	occ := openai.DefaultConfig(apikey)
	occ.HTTPClient = &http.Client{
		Timeout:   openaiTimeout,
		Transport: &http.Transport{Proxy: http.ProxyFromEnvironment},
	}
	if len(args) > 1 && len(args[1]) > 0 {
		occ.BaseURL = args[1]
	}
	return openai.NewClientWithConfig(occ)
}
