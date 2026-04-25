package api

import (
	"context"
	"net/http"

	"github.com/liut/morign/pkg/services/stores"
	"github.com/liut/morign/pkg/settings"
)

// GetOAuthTokenFunc is a function type for retrieving OAuth tokens.
// Currently reads from cookie, future implementations may use Redis/DB.
type GetOAuthTokenFunc func(ctx context.Context, r *http.Request) string

// GetOAuthTokenFromCookie retrieves token from cookie.
// tokenCN must stay consistent with handle_user.go.
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

// OAuthTokenMiddleware creates a middleware that injects the OAuth token into the request context.
// First tries to get token via getToken func (default: cookie), falls back to SiteTokenKey header.
func OAuthTokenMiddleware(getToken GetOAuthTokenFunc) func(http.Handler) http.Handler {
	if getToken == nil {
		getToken = GetOAuthTokenFromCookie
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok := getToken(r.Context(), r)
			if len(tok) == 0 {
				tok = r.Header.Get(settings.Current.SiteTokenKey)
			}
			if len(tok) > 0 {
				ctx := stores.OAuthContextWithToken(r.Context(), tok)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}
