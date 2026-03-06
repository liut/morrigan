package stores

import (
	"github.com/liut/morrigan/pkg/services/llm"
	"github.com/liut/morrigan/pkg/settings"
)

var (
	// 新的 LLM Clients - 按用途分离
	llmEm llm.Client // for Embedding
	llmIt llm.Client // for Interact (chat)
	llmSu llm.Client // for Summarize/Completion
)

func init() {
	// 初始化新的 llm clients
	var err error

	// Embedding Client
	llmEmPtr, err := llm.NewClient(
		llm.WithProvider(settings.Current.Embedding.Type),
		llm.WithAPIKey(settings.Current.Embedding.APIKey),
		llm.WithBaseURL(settings.Current.Embedding.URL),
		llm.WithModel(settings.Current.Embedding.Model),
	)
	if err != nil {
		logger().Fatalw("create llm embedding client failed", "err", err)
	}
	llmEm = llmEmPtr

	// Interact Client (chat)
	llmItPtr, err := llm.NewClient(
		llm.WithProvider(settings.Current.Interact.Type),
		llm.WithAPIKey(settings.Current.Interact.APIKey),
		llm.WithBaseURL(settings.Current.Interact.URL),
		llm.WithModel(settings.Current.Interact.Model),
	)
	if err != nil {
		logger().Fatalw("create llm interact client failed", "err", err)
	}
	llmIt = llmItPtr

	// Summarize/Completion Client (共用)
	llmSuPtr, err := llm.NewClient(
		llm.WithProvider(settings.Current.Summarize.Type),
		llm.WithAPIKey(settings.Current.Summarize.APIKey),
		llm.WithBaseURL(settings.Current.Summarize.URL),
		llm.WithModel(settings.Current.Summarize.Model),
	)
	if err != nil {
		logger().Fatalw("create llm summarize client failed", "err", err)
	}
	llmSu = llmSuPtr
}

// GetLLMClient 获取 LLM Client (默认 Interact)
func GetLLMClient() llm.Client {
	return llmIt
}

// GetLLMEmbeddingClient 获取 Embedding 用 LLM Client
func GetLLMEmbeddingClient() llm.Client {
	return llmEm
}

// GetLLMSummarizeClient 获取 Summarize/Completion 用 LLM Client
func GetLLMSummarizeClient() llm.Client {
	return llmSu
}
