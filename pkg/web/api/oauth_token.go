package api

import (
	"context"
	"net/http"
)

// contextKey for OAuth token
type oauthTokenKeyType struct{}

var oauthTokenKey = oauthTokenKeyType{}

// GetOAuthTokenFunc 用于获取 OAuth token 的函数类型
// 将来可能从 Redis/DB 获取，当前从 cookie 获取
type GetOAuthTokenFunc func(ctx context.Context, r *http.Request) string

// OAuthTokenFromContext 从 context 获取 token
func OAuthTokenFromContext(ctx context.Context) string {
	if tok, ok := ctx.Value(oauthTokenKey).(string); ok {
		return tok
	}
	return ""
}

// OAuthContextWithToken 将 token 添加到 context
func OAuthContextWithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, oauthTokenKey, token)
}

// GetOAuthTokenFromCookie 从 cookie 获取 token（当前实现）
// tokenCN 需要与 handle_user.go 中的保持一致
func GetOAuthTokenFromCookie(ctx context.Context, r *http.Request) string {
	if r == nil {
		return ""
	}
	token, err := r.Cookie(tokenCN)
	if err != nil {
		return ""
	}
	return token.Value
}

// OAuthTokenMiddleware 创建中间件，将 cookie 中的 token 注入 context
func OAuthTokenMiddleware(getToken GetOAuthTokenFunc) func(http.Handler) http.Handler {
	if getToken == nil {
		getToken = GetOAuthTokenFromCookie
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tok := getToken(r.Context(), r); tok != "" {
				ctx := OAuthContextWithToken(r.Context(), tok)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}
