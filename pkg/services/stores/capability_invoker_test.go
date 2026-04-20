package stores

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRequestData(t *testing.T) {
	inv := &CapabilityInvoker{baseURL: "http://localhost:8080"}

	tests := []struct {
		name     string
		method   string
		endpoint string
		params   map[string]any
		wantURL  string
		wantBody string
	}{
		{
			name:     "GET with path param only",
			method:   http.MethodGet,
			endpoint: "/api/a1/companies/{id}",
			params:   map[string]any{"id": "co-123"},
			wantURL:  "http://localhost:8080/api/a1/companies/co-123",
			wantBody: "",
		},
		{
			name:     "GET with path param and query param",
			method:   http.MethodGet,
			endpoint: "/api/a1/companies/{id}",
			params:   map[string]any{"id": "co-123", "name": "test"},
			wantURL:  "http://localhost:8080/api/a1/companies/co-123?name=test",
			wantBody: "",
		},
		{
			name:     "GET with query param only",
			method:   http.MethodGet,
			endpoint: "/api/a1/companies",
			params:   map[string]any{"name": "test"},
			wantURL:  "http://localhost:8080/api/a1/companies?name=test",
			wantBody: "",
		},
		{
			name:     "DELETE with path param and query param",
			method:   http.MethodDelete,
			endpoint: "/api/a1/companies/{id}",
			params:   map[string]any{"id": "co-123", "name": "test"},
			wantURL:  "http://localhost:8080/api/a1/companies/co-123?name=test",
			wantBody: "",
		},
		{
			name:     "DELETE with query param only",
			method:   http.MethodDelete,
			endpoint: "/api/a1/companies",
			params:   map[string]any{"name": "test", "status": "active"},
			wantURL:  "http://localhost:8080/api/a1/companies?name=test&status=active",
			wantBody: "",
		},
		{
			name:     "POST with path param and body",
			method:   http.MethodPost,
			endpoint: "/api/a1/companies/{id}/users",
			params:   map[string]any{"id": "co-123", "name": "test", "role": "admin"},
			wantURL:  "http://localhost:8080/api/a1/companies/co-123/users",
			wantBody: `{"name":"test","role":"admin"}`,
		},
		{
			name:     "POST with body only",
			method:   http.MethodPost,
			endpoint: "/api/a1/companies",
			params:   map[string]any{"name": "test", "status": "active"},
			wantURL:  "http://localhost:8080/api/a1/companies",
			wantBody: `{"name":"test","status":"active"}`,
		},
		{
			name:     "PUT with path param and body",
			method:   http.MethodPut,
			endpoint: "/api/a1/companies/{id}",
			params:   map[string]any{"id": "co-123", "name": "updated"},
			wantURL:  "http://localhost:8080/api/a1/companies/co-123",
			wantBody: `{"name":"updated"}`,
		},
		{
			name:     "PATCH with body only",
			method:   http.MethodPatch,
			endpoint: "/api/a1/companies/{id}",
			params:   map[string]any{"id": "co-123", "name": "patched"},
			wantURL:  "http://localhost:8080/api/a1/companies/co-123",
			wantBody: `{"name":"patched"}`,
		},
		{
			name:     "multiple path params",
			method:   http.MethodGet,
			endpoint: "/api/a1/companies/{id}/users/{userId}",
			params:   map[string]any{"id": "co-123", "userId": "u-456"},
			wantURL:  "http://localhost:8080/api/a1/companies/co-123/users/u-456",
			wantBody: "",
		},
		{
			name:     "multiple path params with remaining",
			method:   http.MethodPost,
			endpoint: "/api/a1/companies/{id}/users/{userId}",
			params:   map[string]any{"id": "co-123", "userId": "u-456", "name": "test"},
			wantURL:  "http://localhost:8080/api/a1/companies/co-123/users/u-456",
			wantBody: `{"name":"test"}`,
		},
		{
			name:     "empty params",
			method:   http.MethodGet,
			endpoint: "/api/a1/companies",
			params:   map[string]any{},
			wantURL:  "http://localhost:8080/api/a1/companies",
			wantBody: "",
		},
		{
			name:     "no matching path param - params go to query for GET",
			method:   http.MethodGet,
			endpoint: "/api/a1/companies/{id}",
			params:   map[string]any{"name": "test"},
			wantURL:  "http://localhost:8080/api/a1/companies/%7Bid%7D?name=test",
			wantBody: "",
		},
		{
			name:     "lowercase method normalized",
			method:   "get",
			endpoint: "/api/a1/companies/{id}",
			params:   map[string]any{"id": "co-123"},
			wantURL:  "http://localhost:8080/api/a1/companies/co-123",
			wantBody: "",
		},
		{
			name:     "boolean and number params - GET",
			method:   http.MethodGet,
			endpoint: "/api/a1/companies",
			params:   map[string]any{"active": true, "limit": 10},
			wantURL:  "http://localhost:8080/api/a1/companies?active=true&limit=10",
			wantBody: "",
		},
		{
			name:     "boolean and number params - POST",
			method:   http.MethodPost,
			endpoint: "/api/a1/companies",
			params:   map[string]any{"active": true, "limit": 10},
			wantURL:  "http://localhost:8080/api/a1/companies",
			wantBody: `{"active":true,"limit":10}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotBody, err := inv.buildRequestData(tt.method, tt.endpoint, tt.params)
			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, gotURL)

			if tt.wantBody == "" {
				assert.Nil(t, gotBody)
			} else {
				require.NotNil(t, gotBody)
				bodyBytes, err := io.ReadAll(gotBody)
				require.NoError(t, err)
				assert.JSONEq(t, tt.wantBody, string(bodyBytes))
			}
		})
	}
}

func TestBuildRequestData_EdgeCases(t *testing.T) {
	inv := &CapabilityInvoker{baseURL: "http://localhost:8080"}

	t.Run("path param with special characters", func(t *testing.T) {
		urlStr, body, err := inv.buildRequestData(http.MethodGet, "/api/test/{id}", map[string]any{"id": "co-123/abc"})
		require.NoError(t, err)
		assert.Equal(t, "http://localhost:8080/api/test/co-123/abc", urlStr)
		assert.Nil(t, body)
	})

	t.Run("path param with empty string value", func(t *testing.T) {
		urlStr, body, err := inv.buildRequestData(http.MethodGet, "/api/test/{id}", map[string]any{"id": ""})
		require.NoError(t, err)
		assert.Equal(t, "http://localhost:8080/api/test/", urlStr)
		assert.Nil(t, body)
	})

	t.Run("query param with special characters", func(t *testing.T) {
		urlStr, _, err := inv.buildRequestData(http.MethodGet, "/api/test", map[string]any{"q": "hello world"})
		require.NoError(t, err)
		assert.Equal(t, "http://localhost:8080/api/test?q=hello+world", urlStr)
	})

	t.Run("nil params treated as empty", func(t *testing.T) {
		urlStr, body, err := inv.buildRequestData(http.MethodGet, "/api/test", nil)
		require.NoError(t, err)
		assert.Equal(t, "http://localhost:8080/api/test", urlStr)
		assert.Nil(t, body)
	})

	t.Run("array params in body", func(t *testing.T) {
		urlStr, body, err := inv.buildRequestData(http.MethodPost, "/api/test", map[string]any{"ids": []int{1, 2, 3}})
		require.NoError(t, err)
		assert.Equal(t, "http://localhost:8080/api/test", urlStr)
		bodyBytes, _ := io.ReadAll(body)
		assert.JSONEq(t, `{"ids":[1,2,3]}`, string(bodyBytes))
	})

	t.Run("map params in body", func(t *testing.T) {
		urlStr, body, err := inv.buildRequestData(http.MethodPost, "/api/test", map[string]any{"metadata": map[string]any{"key": "value"}})
		require.NoError(t, err)
		assert.Equal(t, "http://localhost:8080/api/test", urlStr)
		bodyBytes, _ := io.ReadAll(body)
		assert.JSONEq(t, `{"metadata":{"key":"value"}}`, string(bodyBytes))
	})

	t.Run("empty body for POST when no remaining params", func(t *testing.T) {
		urlStr, body, err := inv.buildRequestData(http.MethodPost, "/api/test/{id}", map[string]any{"id": "123"})
		require.NoError(t, err)
		assert.Equal(t, "http://localhost:8080/api/test/123", urlStr)
		assert.Nil(t, body)
	})

	t.Run("PUT with query string-like path", func(t *testing.T) {
		urlStr, body, err := inv.buildRequestData(http.MethodPut, "/api/test", map[string]any{"name": "test"})
		require.NoError(t, err)
		assert.Equal(t, "http://localhost:8080/api/test", urlStr)
		require.NotNil(t, body)
		bodyBytes, _ := io.ReadAll(body)
		assert.JSONEq(t, `{"name":"test"}`, string(bodyBytes))
	})

	t.Run("all HTTP methods supported", func(t *testing.T) {
		methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
		for _, method := range methods {
			_, _, err := inv.buildRequestData(method, "/api/test", map[string]any{"key": "value"})
			require.NoError(t, err, "method %s should not error", method)
		}
	})
}
