package stores

import (
	"net/http"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/liut/morrigan/pkg/settings"
)

const (
	openaiTimeout = time.Second * 30
)

func NewOpenAIClient() *openai.Client {
	occ := openai.DefaultConfig(settings.Current.OpenAIAPIKey)
	occ.HTTPClient = &http.Client{
		Timeout:   openaiTimeout,
		Transport: &http.Transport{Proxy: http.ProxyFromEnvironment},
	}
	return openai.NewClientWithConfig(occ)
}
