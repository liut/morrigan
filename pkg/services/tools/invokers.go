package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	htmd "github.com/JohannesKaufmann/html-to-markdown"

	"github.com/liut/morrigan/pkg/models/cob"
	"github.com/liut/morrigan/pkg/models/mcps"
	"github.com/liut/morrigan/pkg/services/stores"
)

const (
	DEFAULT_USER_AGENT_AUTONOMOUS = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// KB Search implementation
func (r *Registry) callKBSearch(ctx context.Context, args map[string]any) (map[string]any, error) {
	logger().Infow("mcp call qa search", "args", args)
	subjectArg, ok := args["subject"]
	if !ok {
		return mcps.BuildToolErrorResult("missing required argument: subject"), nil
	}
	subject, ok := subjectArg.(string)
	if !ok {
		return mcps.BuildToolErrorResult("subject argument must be a string"), nil
	}

	docs, err := r.sto.Cob().MatchDocments(ctx, stores.MatchSpec{
		Question:     subject,
		Limit:        5,
		SkipKeywords: true,
	})
	if err != nil {
		return mcps.BuildToolErrorResult(err.Error()), nil
	}
	logger().Infow("matched", "docs", len(docs))
	if len(docs) == 0 {
		return mcps.BuildToolSuccessResult("No relevant information found"), nil
	}

	return mcps.BuildToolSuccessResult(docs.MarkdownText()), nil
}

// KB Create implementation
func (r *Registry) callKBCreate(ctx context.Context, args map[string]any) (map[string]any, error) {
	if !stores.IsKeeper(ctx) {
		return mcps.BuildToolErrorResult("permission denied: keeper role required"), nil
	}

	user, _ := stores.UserFromContext(ctx)
	logger().Infow("mcp call qa create", "args", args, "user", user)

	titleArg, ok := args["title"]
	if !ok {
		return mcps.BuildToolErrorResult("missing required argument: title"), nil
	}
	headingArg, ok := args["heading"]
	if !ok {
		return mcps.BuildToolErrorResult("missing required argument: heading"), nil
	}
	contentArg, ok := args["content"]
	if !ok {
		return mcps.BuildToolErrorResult("missing required argument: content"), nil
	}

	docBasic := cob.DocumentBasic{
		Title:   titleArg.(string),
		Heading: headingArg.(string),
		Content: contentArg.(string),
	}
	docBasic.MetaAddKVs("creator", user.Name)
	obj, err := r.sto.Cob().CreateDocument(ctx, docBasic)
	if err != nil {
		logger().Infow("create document fail", "title", docBasic.Title, "heading", docBasic.Heading,
			"content", len(docBasic.Content), "err", err)
		return mcps.BuildToolSuccessResult(fmt.Sprintf(
			"Create KB document with title %q and heading %q is failed, %s", docBasic.Title, docBasic.Heading, err)), nil
	}
	return mcps.BuildToolSuccessResult(fmt.Sprintf("Created KB document with ID %s", obj.StringID())), nil
}

// Fetch implementation
func (r *Registry) callFetch(ctx context.Context, args map[string]any) (map[string]any, error) {
	var (
		urlStr     string
		maxLength  int
		startIndex int
		raw        bool
	)

	urlStr = mcps.StringArg(args, "url")
	maxLength, _, _ = mcps.IntArg(args, "max_length")
	if maxLength == 0 {
		maxLength = 5000
	}
	startIndex, _, _ = mcps.IntArg(args, "start_index")
	raw, _, _ = mcps.BoolArg(args, "raw")

	// Fetch URL
	content, prefix, err := fetchURL(ctx, urlStr, DEFAULT_USER_AGENT_AUTONOMOUS, raw)
	if err != nil {
		logger().Infow("fetch", "url", urlStr, "err", err)
		return mcps.BuildToolSuccessResult(err.Error()), nil
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

	return mcps.BuildToolSuccessResult(fmt.Sprintf("%s\nContents of %s:\n%s", prefix, urlStr, content)), nil
}

var converter = htmd.NewConverter("", true, nil)

func extractContentFromHTML(html string) string {
	doc, err := converter.ConvertString(html)
	if err != nil {
		logger().Infow("failed to convert HTML to markdown", "err", err)
		return strings.TrimSpace(html)
	}
	return doc
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

	contentType := resp.Header.Get("content-type")
	isHTML := strings.Contains(contentType, "text/html") || strings.Contains(content, "<html")

	if isHTML && !raw {
		content = extractContentFromHTML(content)
		prefix = "Markdown"
	} else {
		prefix = fmt.Sprintf("Content type %s, raw content:", contentType)
	}
	return
}
