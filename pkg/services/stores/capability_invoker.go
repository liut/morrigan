package stores

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cast"
)

// CapabilityInvoker invokes Bus API calls with path parameter substitution.
type CapabilityInvoker struct {
	httpClient *http.Client
	baseURL    string
}

// NewCapabilityInvoker creates a CapabilityInvoker with the given Bus API base URL.
func NewCapabilityInvoker(baseURL string) *CapabilityInvoker {
	return &CapabilityInvoker{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    strings.TrimSuffix(baseURL, "/"),
	}
}

// Invoke makes an HTTP request to the Bus API.
func (inv *CapabilityInvoker) Invoke(ctx context.Context, method, endpoint string, params map[string]any) (*http.Response, error) {
	method = strings.ToUpper(method)

	// Build URL and body
	reqURL, body, err := inv.buildRequestData(method, endpoint, params)
	if err != nil {
		return nil, err
	}

	logger().Infow("invoking api", "method", method, "url", reqURL, "params", params)

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("X-Ai-Agent", "morign")
	req.Header.Set("Content-Type", "application/json")
	if tk := OAuthTokenFromContext(ctx); len(tk) > 0 {
		req.Header.Set("Authorization", "Bearer "+tk)
	}

	return inv.httpClient.Do(req)
}

// buildRequestData handles path parameter substitution, query string construction, and JSON body generation.
func (inv *CapabilityInvoker) buildRequestData(method, endpoint string, params map[string]any) (string, io.Reader, error) {
	fullURL := inv.baseURL + endpoint
	if len(params) == 0 {
		return fullURL, nil, nil
	}

	u, err := url.Parse(fullURL)
	if err != nil {
		return "", nil, fmt.Errorf("parse url: %w", err)
	}

	// 1. Substitute path parameters, collect remaining params
	remaining := make(map[string]any)
	path := u.Path
	for k, v := range params {
		placeholder := "{" + k + "}"
		if strings.Contains(path, placeholder) {
			path = strings.ReplaceAll(path, placeholder, cast.ToString(v))
		} else {
			remaining[k] = v
		}
	}
	u.Path = path

	if len(remaining) == 0 {
		return u.String(), nil, nil
	}

	// 2. Route remaining params based on HTTP method
	var body io.Reader
	switch method {
	case http.MethodGet, http.MethodDelete:
		// GET and DELETE: remaining params become query string
		q := u.Query()
		for k, v := range remaining {
			q.Add(k, cast.ToString(v))
		}
		u.RawQuery = q.Encode()

	case http.MethodPost, http.MethodPut, http.MethodPatch:
		// POST, PUT, PATCH: remaining params become JSON body
		data, err := json.Marshal(remaining)
		if err != nil {
			return "", nil, fmt.Errorf("marshal params: %w", err)
		}
		body = bytes.NewReader(data)
	}

	return u.String(), body, nil
}
