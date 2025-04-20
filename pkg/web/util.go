package web

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	htmd "github.com/JohannesKaufmann/html-to-markdown"
)

func nameOfFunction(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

var byteUnits = []string{"B", "KB", "MB", "GB", "TB", "PB"}

// FormatRate formats a floating point b/s as string with closest unit
func FormatRate(rate float64) (str string) {
	return FormatBytes(rate, "/sec")
}

// FormatBytes formats a floating number as string with closest unit
func FormatBytes(num float64, suffix string) (str string) {
	if math.IsInf(num, 0) {
		str = "infinity"
		return
	}
	var idx int
	for num > 1024.0 {
		num /= 1024.0
		idx++
	}
	str = fmt.Sprintf("%.2f%s%s", num, byteUnits[idx], suffix)
	return
}

func patchImageURI(uri, prefix string) string {
	if uri == "" {
		return ""
	}
	if strings.HasPrefix(uri, "http") || strings.HasPrefix(uri, "//") {
		return uri
	}

	return strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(uri, "/")
}

const DEFAULT_USER_AGENT_AUTONOMOUS = "ModelContextProtocol/1.0 (Autonomous; +https://github.com/modelcontextprotocol/servers)"

var converter = htmd.NewConverter("", true, nil)

// Implement HTML to markdown conversion
func extractContentFromHTML(html string) string {
	doc, err := converter.ConvertString(html)
	if err != nil {
		logger().Infow("failed to convert HTML to markdown", "err", err)
		return strings.TrimSpace(html)
	}
	return doc
}

func fetchURL(ctx context.Context, urlStr, userAgent string, forceRaw bool) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger().Infow("failed to fetch URL", "url", urlStr, "err", err)
		return "", "", fmt.Errorf("failed to fetch %s: %v", urlStr, err)
	}
	defer resp.Body.Close()
	contentType := resp.Header.Get("content-type")
	logger().Debugw("fetch URL", "url", urlStr, "status", resp.StatusCode,
		"contentLen", resp.ContentLength, "contentType", contentType)

	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("failed to fetch %s - status code %d", urlStr, resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %v", err)
	}
	content := string(body)
	logger().Debugw("fetch URL", "url", urlStr, "contentLen", len(content))

	isHTML := strings.Contains(contentType, "text/html") || strings.Contains(content, "<html")

	if isHTML && !forceRaw {
		return extractContentFromHTML(content), "", nil
	}

	return content, fmt.Sprintf("Content type %s cannot be simplified to markdown, but here is the raw content:\n", contentType), nil
}
