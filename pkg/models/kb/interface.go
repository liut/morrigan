package kb

import (
	"context"
	"strings"
)

// KnowledgeBase defines the interface for knowledge base operations.
type KnowledgeBase interface {
	// Search performs a similarity search in the knowledge base.
	Search(ctx context.Context, req SearchRequest) (SearchResult, error)

	// GetContext builds a context string from matched documents for RAG.
	GetContext(ctx context.Context, query string, limit int) (string, error)

	// Create adds a new document to the knowledge base.
	Create(ctx context.Context, doc DocumentBasic) (Document, error)
}

// SearchRequest represents a search query.
type SearchRequest struct {
	Query        string
	Limit        int
	Threshold    float32
	SkipKeywords bool
}

// SearchResult contains matched documents.
type SearchResult struct {
	Documents []Document
}

// MarkdownText converts documents to markdown format.
func (sr SearchResult) MarkdownText() string {
	var sections []string
	for _, doc := range sr.Documents {
		sections = append(sections, doc.Heading+": "+doc.Content)
	}
	return "\n* " + strings.Join(sections, "\n* ")
}

// Document represents a knowledge base document.
type Document struct {
	ID       string
	Title    string
	Heading  string
	Content  string
	Score    float32
}

// DocumentBasic represents a document to be created.
type DocumentBasic struct {
	Title   string
	Heading string
	Content string
	Meta    map[string]any
}

// DocumentList is a container for iterating documents.
type DocumentList []Document

func (dl DocumentList) Iterate(fn func(Document)) {
	for _, d := range dl {
		fn(d)
	}
}
