package api

import (
	"context"
	"net/http"

	"github.com/go-chi/render"
	staffio "github.com/liut/staffio-client"
	"github.com/marcsv/go-binder/binder"

	"github.com/liut/morrigan/pkg/settings"
)

func init() {
	// 注册 session/verify 路由
	regHI(false, "GET", "/session", "", func(a *api) http.HandlerFunc {
		return a.handleSession
	})
	regHI(false, "POST", "/session", "", func(a *api) http.HandlerFunc {
		return a.handleSession
	})
	regHI(false, "POST", "/verify", "", func(a *api) http.HandlerFunc {
		return a.handleVerify
	})
	regHI(true, "GET", "/me", "", func(a *api) http.HandlerFunc {
		return handleMe
	})
}

type User = staffio.User

// vars from staffio
var (
	UserFromContext = staffio.UserFromContext
)

// staffio 认证相关
const (
	authLoginPath    = "/api/auth/login"
	authLogoutPath   = "/api/auth/logout"
	authCallbackPath = "/api/auth/callback"

	tokenCN = "o_token" // from oauth2 provider
)

func buildTokenCookie(value string) *http.Cookie {
	return &http.Cookie{
		Name:     tokenCN,
		Value:    value,
		HttpOnly: true,
		Path:     settings.Current.CookiePath,
		MaxAge:   settings.Current.CookieMaxAge,
	}
}

func handleTokenGot(ctx context.Context, w http.ResponseWriter, it *staffio.InfoToken) {
	ot := staffio.TokenFromContext(ctx)
	if ot != nil {
		logger().Infow("got token", "it", it, "ot", staffio.TokenFromContext(ctx))
		http.SetCookie(w, buildTokenCookie(ot.AccessToken))
	}
}

// @Tags 用户 认证
// @Summary 获取当前用户
// @Accept json
// @Produce json
// @Success 200 {object} Done{result=User}
// @Failure 401 {object} Failure "未登录"
// @Router /api/me [get]
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

// @Tags 用户 认证
// @Summary 获取会话信息
// @Description 获取当前登录状态和用户信息 for github.com/Chanzhaoyu/chatgpt-web
// @Accept json
// @Produce json
// @Success 200 {object} Done{result=respSession}
// @Router /api/session [get]
// @Router /api/session [post]
func (a *api) handleSession(w http.ResponseWriter, r *http.Request) {
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

// @Tags 用户 认证
// @Summary 验证Token
// @Description 验证用户Token for github.com/Chanzhaoyu/chatgpt-web
// @Accept json
// @Produce json
// @Param token body verifyReq true "验证Token"
// @Success 200 {object} Done{result=M}
// @Failure 400 {object} Failure "请求或参数错误"
// @Failure 401 {object} Failure "Token验证失败"
// @Router /api/verify [post]
func (a *api) handleVerify(w http.ResponseWriter, r *http.Request) {
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

// patchImageURI 修复图片 URI
func patchImageURI(uri, prefix string) string {
	if uri == "" {
		return ""
	}
	if prefix == "" {
		return uri
	}
	// 如果已经是完整 URL，直接返回
	if len(uri) > 7 && (uri[:7] == "http://" || uri[:8] == "https://") {
		return uri
	}
	// 如果是相对路径，添加 prefix
	return prefix + uri
}
