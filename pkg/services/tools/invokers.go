package tools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/liut/morrigan/pkg/models/qas"
	"github.com/liut/morrigan/pkg/services/stores"
)

const (
	DEFAULT_USER_AGENT_AUTONOMOUS = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// KB Search implementation
func (r *Registry) callKBSearch(ctx context.Context, args map[string]any) (mcp.Content, error) {
	logger().Infow("mcp call qa search", "args", args)
	subjectArg, ok := args["subject"]
	if !ok {
		return nil, errors.New("missing required argument: subject")
	}
	subject, ok := subjectArg.(string)
	if !ok {
		return nil, errors.New("subject argument must be a string")
	}

	docs, err := r.sto.Qa().MatchDocments(ctx, stores.MatchSpec{
		Question:     subject,
		Limit:        5,
		SkipKeywords: true,
	})
	if err != nil {
		return nil, err
	}
	logger().Infow("matched", "docs", len(docs))
	if len(docs) == 0 {
		return mcp.NewTextContent("No relevant information found"), nil
	}

	return mcp.NewTextContent(docs.MarkdownText()), nil
}

// KB Create implementation
func (r *Registry) callKBCreate(ctx context.Context, args map[string]any) (mcp.Content, error) {
	user, ok := stores.UserFromContext(ctx)
	if !ok {
		return nil, errors.New("only admin can create document")
	}
	logger().Infow("mcp call qa create", "args", args, "user", user)

	titleArg, ok := args["title"]
	if !ok {
		return nil, errors.New("missing required argument: title")
	}
	headingArg, ok := args["heading"]
	if !ok {
		return nil, errors.New("missing required argument: heading")
	}
	contentArg, ok := args["content"]
	if !ok {
		return nil, errors.New("missing required argument: content")
	}

	docBasic := qas.DocumentBasic{
		Title:   titleArg.(string),
		Heading: headingArg.(string),
		Content: contentArg.(string),
	}
	docBasic.MetaAddKVs("creator", user.Name)
	obj, err := r.sto.Qa().CreateDocument(ctx, docBasic)
	if err != nil {
		logger().Infow("create document fail", "title", docBasic.Title, "heading", docBasic.Heading,
			"content", len(docBasic.Content), "err", err)
		return mcp.NewTextContent(fmt.Sprintf(
			"Create KB document with title %q and heading %q is failed, %s", docBasic.Title, docBasic.Heading, err)), nil
	}
	return mcp.NewTextContent(fmt.Sprintf("Created KB document with ID %s", obj.StringID())), nil
}

// Fetch implementation
func (r *Registry) callFetch(ctx context.Context, args map[string]any) (mcp.Content, error) {
	var (
		urlStr     string
		maxLength  int
		startIndex int
		raw        bool
	)
	if s, ok := args["url"]; ok {
		urlStr = s.(string)
	}
	if s, ok := args["max_length"]; ok {
		maxLength = int(s.(float64))
	}
	if s, ok := args["start_index"]; ok {
		startIndex = int(s.(float64))
	}
	if s, ok := args["raw"]; ok {
		raw = s.(bool)
	}

	// Fetch URL
	content, prefix, err := fetchURL(ctx, urlStr, DEFAULT_USER_AGENT_AUTONOMOUS, raw)
	if err != nil {
		logger().Infow("fetch", "url", urlStr, "err", err)
		return mcp.NewTextContent(err.Error()), nil
	}
	logger().Debugw("fetch", "url", urlStr, "content", content, "prefix", prefix)

	// Handle truncation
	originalLength := len(content)
	if startIndex >= originalLength {
		content = "<error>No more content available.</error>"
	} else {
		endIndex := min(startIndex+maxLength, originalLength)
		truncatedContent := content[startIndex:endIndex]
		if len(truncatedContent) == 0 {
			content = "<error>No more content available.</error>"
		} else {
			content = truncatedContent
			actualContentLength := len(truncatedContent)
			remainingContent := originalLength - (startIndex + actualContentLength)
			if actualContentLength == maxLength && remainingContent > 0 {
				nextStart := startIndex + actualContentLength
				content += fmt.Sprintf("\n\n<error>Content truncated. Call the fetch tool with a start_index of %d to get more content.</error>", nextStart)
			}
		}
	}
	logger().Debugw("fetch", "url", urlStr, "content", content, "prefix", prefix)

	return mcp.NewTextContent(fmt.Sprintf("%s\nContents of %s:\n%s", prefix, urlStr, content)), nil
}

func fetchURL(ctx context.Context, urlStr, userAgent string, raw bool) (content, prefix string, err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("HTTP %d", resp.StatusCode)
		return
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	content = string(b)
	prefix = "Markdown"
	if raw {
		prefix = "HTML"
	}
	return
}
