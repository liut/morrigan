package stores

import (
	"context"

	mkb "github.com/liut/morrigan/pkg/models/kb"
	"github.com/liut/morrigan/pkg/settings"
)

// RAGService provides retrieval-augmented generation capabilities.
type RAGService struct {
	kb mkb.KnowledgeBase
}

// NewRAGService creates a new RAG service.
func NewRAGService(kb mkb.KnowledgeBase) *RAGService {
	return &RAGService{kb: kb}
}

// BuildContext retrieves relevant documents and builds context for LLM.
func (s *RAGService) BuildContext(ctx context.Context, query string) (string, error) {
	if s.kb == nil {
		return "", nil
	}
	return s.kb.GetContext(ctx, query, settings.Current.VectorLimit)
}

// Search performs a search in the knowledge base using MatchSpec.
func (s *RAGService) Search(ctx context.Context, spec MatchSpec) (mkb.SearchResult, error) {
	if s.kb == nil {
		return mkb.SearchResult{}, nil
	}
	spec.setDefaults()
	return s.kb.Search(ctx, mkb.SearchRequest{
		Query:        spec.Question,
		Limit:        spec.Limit,
		Threshold:    spec.Threshold,
		SkipKeywords: spec.SkipKeywords,
	})
}

// Search performs a search in the knowledge base.
func (s *RAGService) SearchKB(ctx context.Context, req mkb.SearchRequest) (mkb.SearchResult, error) {
	if s.kb == nil {
		return mkb.SearchResult{}, nil
	}
	return s.kb.Search(ctx, req)
}

// KB returns the underlying knowledge base.
func (s *RAGService) KB() mkb.KnowledgeBase {
	return s.kb
}
