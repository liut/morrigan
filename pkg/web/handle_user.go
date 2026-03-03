package web

import (
	"context"
	"net/http"

	"github.com/go-chi/render"
	staffio "github.com/liut/staffio-client"
	"github.com/marcsv/go-binder/binder"

	"github.com/liut/morrigan/pkg/settings"
	"github.com/liut/morrigan/pkg/web/routes"
)

// User online user
type User = staffio.User

// vars from staffio
var (
	UserFromContext = staffio.UserFromContext
)

func (s *server) authMw(redir bool) func(next http.Handler) http.Handler {
	return routes.AuthMw(redir)
}

const authLoginPath = "/api/auth/login"
const authLogoutPath = "/api/auth/logout"
const authCallbackPath = "/api/auth/callback"

const (
	tokenCN = "o_token" // from oauth2 provider
)

func (s *server) buildTokenCookie(value string) *http.Cookie {
	return &http.Cookie{
		Name:     tokenCN,
		Value:    value,
		HttpOnly: true,
		Path:     settings.Current.CookiePath,
	}
}

func (s *server) handleTokenGot(ctx context.Context, w http.ResponseWriter, it *staffio.InfoToken) {
	ot := staffio.TokenFromContext(ctx)
	if ot != nil {
		logger().Infow("got token", "it", it, "ot", staffio.TokenFromContext(ctx))
		http.SetCookie(w, s.buildTokenCookie(ot.AccessToken))
	}
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	if !settings.Current.AuthRequired {
		apiOk(w, r, &User{})
		return
	}
	if user, ok := UserFromContext(r.Context()); ok {
		apiOk(w, r, user)
	} else {
		apiFail(w, r, 401, "not login")
	}
}

type respSession struct {
	Status string `json:"status"`
	Data   struct {
		Auth  bool   `json:"auth"`            // need auth
		User  *User  `json:"user,omitempty"`  // logined user
		URI   string `json:"uri,omitempty"`   // uri of auth
		Model string `json:"model,omitempty"` // for chatgpt-web
		Token string `json:"token,omitempty"` // token from oauth2 provider
	} `json:"data"`
}

// for github.com/Chanzhaoyu/chatgpt-web
func (s *server) handleSession(w http.ResponseWriter, r *http.Request) {
	user, err := staffio.UserFromRequest(r)
	var res respSession
	res.Status = "Success"
	// res.Data.Model = "ChatGPTAPI"
	logger().Debugw("handle session", "user", user, "err", err)
	if settings.Current.AuthRequired {
		if err == nil {
			user.Avatar = patchImageURI(user.Avatar, staffio.GetPrefix())
			res.Data.User = user
			if token, err := r.Cookie(tokenCN); err == nil {
				res.Data.Token = token.Value
			}
		} else {
			res.Data.Auth = true
			res.Data.URI = authLoginPath
		}
	} else {
		res.Data.Auth = len(settings.Current.AuthSecret) > 0
	}
	render.JSON(w, r, &res)
}

type verifyReq struct {
	Token string `json:"token"`
}

// for github.com/Chanzhaoyu/chatgpt-web
func (s *server) handleVerify(w http.ResponseWriter, r *http.Request) {
	var param verifyReq
	if err := binder.BindBody(r, &param); err != nil {
		apiFail(w, r, 400, err)
		return
	}
	if param.Token != settings.Current.AuthSecret {
		apiFail(w, r, 401, "mismatch token")
	}
	// user := new(User)
	// if err := user.Decode(param.Token); err != nil {
	// 	apiFail(w, r, 401, err)
	// 	return
	// }

	render.JSON(w, r, M{"status": "Success"})
}
