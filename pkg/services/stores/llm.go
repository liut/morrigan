package stores

import (
	"context"
	"fmt"
	"strings"

	"github.com/liut/morign/pkg/models/aigc"
	"github.com/liut/morign/pkg/services/llm"
	"github.com/liut/morign/pkg/settings"
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

// GetKeywords extracts keywords from text using LLM
func GetKeywords(ctx context.Context, text string) (kw string, err error) {
	if len(text) == 0 {
		err = ErrEmptyParam
		return
	}
	prompt := fmt.Sprintf(tplKeyword, text)
	result, _, err := llmSu.Generate(ctx, prompt)
	if err != nil {
		logger().Infow("summarize fail", "text", text, "err", err)
		return
	}
	kw = strings.TrimSpace(result)
	logger().Infow("summarize ok", "text", text, "kw", kw)
	return
}

// GetSummary 生成聊天记录的简短标题
// text 参数为聊天记录文本，tips 参数为自定义提示内容（可选）
func GetSummary(ctx context.Context, text, tips string) (summary string, err error) {
	if len(text) == 0 {
		err = ErrEmptyParam
		return
	}
	// 使用自定义提示或默认提示
	if tips == "" {
		tips = "请根据以下聊天记录生成一个简短的标题（不超过10个字），这个标题只针对聊天的主题，且只返回标题，不要其他内容:"
	}
	prompt := fmt.Sprintf("%s\n\n%s\n\n标题:", tips, text)
	result, _, err := llmSu.Generate(ctx, prompt)
	if err != nil {
		logger().Infow("summary fail", "text", text, "err", err)
		return
	}
	summary = strings.TrimSpace(result)
	logger().Infow("summary ok", "text", len(text), "summary", summary)
	return
}

func GetHistorySummary(ctx context.Context, history aigc.HistoryItems, tips string) (summary string, err error) {
	summary, err = GetSummary(ctx, history.ToText(), tips)
	if err != nil {
		return
	}
	logger().Infow("history summary ok", "history", aigc.HiLogged(history))
	return
}
