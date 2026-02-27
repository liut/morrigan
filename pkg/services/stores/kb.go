package stores

import (
	"context"

	mkb "github.com/liut/morrigan/pkg/models/kb"
	"github.com/liut/morrigan/pkg/models/qas"
)

// KBProvider implements mkb.KnowledgeBase interface.
type KBProvider struct {
	sto Storage
}

// NewKBProvider creates a new KBProvider.
func NewKBProvider(sto Storage) *KBProvider {
	return &KBProvider{sto: sto}
}

// Search performs a similarity search in the knowledge base.
func (p *KBProvider) Search(ctx context.Context, req mkb.SearchRequest) (mkb.SearchResult, error) {
	spec := MatchSpec{
		Question:     req.Query,
		Limit:        req.Limit,
		Threshold:    req.Threshold,
		SkipKeywords: req.SkipKeywords,
	}
	spec.setDefaults()

	docs, err := p.sto.Qa().MatchDocments(ctx, spec)
	if err != nil {
		return mkb.SearchResult{}, err
	}

	result := mkb.SearchResult{
		Documents: make([]mkb.Document, 0, len(docs)),
	}
	for _, d := range docs {
		result.Documents = append(result.Documents, mkb.Document{
			ID:       d.StringID(),
			Title:    d.Title,
			Heading:  d.Heading,
			Content:  d.Content,
			Score:    0, // Score not available from current implementation
		})
	}

	return result, nil
}

// GetContext builds a context string from matched documents for RAG.
func (p *KBProvider) GetContext(ctx context.Context, query string, limit int) (string, error) {
	spec := MatchSpec{
		Question: query,
		Limit:    limit,
	}
	spec.setDefaults()

	return p.sto.Qa().ConstructPrompt(ctx, spec)
}

// Create adds a new document to the knowledge base.
func (p *KBProvider) Create(ctx context.Context, doc mkb.DocumentBasic) (mkb.Document, error) {
	basic := qas.DocumentBasic{
		Title:   doc.Title,
		Heading: doc.Heading,
		Content: doc.Content,
	}
	if doc.Meta != nil {
		if v, ok := doc.Meta["creator"]; ok {
			if creator, ok := v.(string); ok {
				basic.MetaAddKVs("creator", creator)
			}
		}
	}

	obj, err := p.sto.Qa().CreateDocument(ctx, basic)
	if err != nil {
		return mkb.Document{}, err
	}

	return mkb.Document{
		ID:       obj.StringID(),
		Title:    obj.Title,
		Heading:  obj.Heading,
		Content:  obj.Content,
	}, nil
}
