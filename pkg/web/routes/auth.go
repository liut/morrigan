package routes

import (
	"net/http"
	"sync"

	"github.com/go-chi/render"
	staffio "github.com/liut/staffio-client"

	"github.com/liut/morign/pkg/settings"
)

var (
	authzr  staffio.Authorizer
	azonce  sync.Once
)

// Authzr 获取 staffio Authorizer 单例
func Authzr() staffio.Authorizer {
	azonce.Do(func() {
		authzr = staffio.NewAuth(staffio.WithCookie(
			settings.Current.CookieName,
			settings.Current.CookiePath,
			settings.Current.CookieDomain,
		), staffio.WithRefresh(), staffio.WithURI(staffio.LoginPath))
	})
	return authzr
}

// AuthMw 返回认证中间件
func AuthMw(redir bool) func(next http.Handler) http.Handler {
	if settings.Current.AuthRequired {
		return Authzr().MiddlewareWordy(redir)
	}
	needAuth := len(settings.Current.AuthSecret) > 0
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if needAuth {
				tok, err := Authzr().TokenFromRequest(r)
				if err != nil || tok != settings.Current.AuthSecret {
					w.WriteHeader(401)
					render.JSON(w, r, render.M{
						"status":  "Unauthorized",
						"message": "Please authenticate.",
					})
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
