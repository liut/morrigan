package web

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/marcsv/go-binder/binder"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/middleware/stdlib"
	limitRedis "github.com/ulule/limiter/v3/drivers/store/redis"

	staffio "github.com/liut/staffio-client"

	"github.com/liut/morrigan/pkg/services/stores"
	"github.com/liut/morrigan/pkg/settings"
)

type M = render.M

// User online user
type User = staffio.User

// vars from staffio
var (
	UserFromContext = staffio.UserFromContext
)

func (s *server) authMw(redir bool) func(next http.Handler) http.Handler {
	if settings.Current.AuthRequired {
		return s.authzr.MiddlewareWordy(redir)
	}
	needAuth := len(settings.Current.AuthSecret) > 0
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if needAuth {
				tok, err := s.authzr.TokenFromRequest(r)
				if err != nil || tok != settings.Current.AuthSecret {
					w.WriteHeader(401)
					render.JSON(w, r, M{
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

func (s *server) strapRouter() {

	s.ar.Get("/", handleNoContent)
	s.ar.Get("/ping", handlerPing)

	cch := (&staffio.CodeCallback{
		OnTokenGot: s.handleTokenGot,
	}).Handler()

	s.ar.Route("/auth", func(r chi.Router) {
		r.Get("/login", staffio.LoginHandler)
		r.Get("/logout", staffio.LogoutHandler)
		r.Method(http.MethodGet, "/callback", cch)
	})

	rate, err := limiter.NewRateFromFormatted(settings.Current.AskRateLimit)
	if err != nil {
		logger().Fatalw("settings failed", "err", err)
	}
	store, err := limitRedis.NewStoreWithOptions(stores.SgtRC(), limiter.StoreOptions{
		Prefix: "chat-lr-",
	})
	if err != nil {
		logger().Fatalw("limiter with redis option failed", "err", err)
	}
	instance := limiter.New(store, rate)
	middleware := stdlib.NewMiddleware(instance,
		stdlib.WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
			logger().Warnw("failed on", "uri", r.RequestURI, "err", err)
		}))

	s.ar.Route("/api", func(r chi.Router) {
		r.Use(s.authMw(false))
		r.Get("/me", handleMe)

		r.Get("/models", s.getModels)
		r.Get("/welcome", s.getWelcome)
		r.Get("/history/{cid}", s.getHistory)
		r.Group(func(gr chi.Router) {
			gr.Use(middleware.Handler)
			gr.Post("/chat", s.postChat)
			gr.Post("/chat-{suffix}", s.postChat)
			gr.Post("/completions", s.postCompletions)
		})

	})

	s.ar.Get("/api/session", s.handleSession)
	s.ar.Post("/api/session", s.handleSession)
	s.ar.Post("/api/verify", s.handleVerify)

	staffio.SetAdminPath("/")
	s.ar.Group(func(r chi.Router) {
		r.Use(s.authMw(true))
		if s.cfg.DocHandler != nil {
			r.Get("/", s.cfg.DocHandler.ServeHTTP)
		}
	})
	if s.cfg.DocHandler != nil {
		s.ar.NotFound(s.cfg.DocHandler.ServeHTTP)
	}
}

const (
	tokenCN = "token"
)

func (s *server) buildTokenCookie(value string) *http.Cookie {
	return &http.Cookie{
		Name:     tokenCN,
		Value:    value,
		HttpOnly: true,
	}
}

func (s *server) handleTokenGot(ctx context.Context, w http.ResponseWriter, it *staffio.InfoToken) {
	http.SetCookie(w, s.buildTokenCookie(it.AccessToken))
}

// nolint
func handlerHome(w http.ResponseWriter, r *http.Request) {
	render.HTML(w, r, "hi")
}

// nolint
func handleNoContent(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(204)
}

func handlerPing(w http.ResponseWriter, r *http.Request) {
	render.Data(w, r, []byte("Pong\n"))
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
		Auth bool   `json:"auth"`           // need auth
		User *User  `json:"user,omitempty"` // logined user
		URI  string `json:"uri,omitempty"`  // uri of auth
	} `json:"data"`
}

// for github.com/Chanzhaoyu/chatgpt-web
func (s *server) handleSession(w http.ResponseWriter, r *http.Request) {
	user, err := staffio.UserFromRequest(r)
	var res respSession
	res.Status = "Success"

	if settings.Current.AuthRequired {
		if err == nil {
			res.Data.User = user
		} else {
			res.Data.Auth = true
			res.Data.URI = "/auth/login"
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

func apiFail(w http.ResponseWriter, r *http.Request, status int, err interface{}) {
	res := render.M{
		"status": status,
		"error":  err,
	}
	switch ret := err.(type) {
	case error:
		res["message"] = ret.Error()
	case fmt.Stringer:
		res["message"] = ret.String()
	case string, *string, []byte:
		res["message"] = ret
	}
	render.JSON(w, r, res)
}

type RespDone struct {
	Status int `json:"status"`
	Data   any `json:"data,omitempty"`
	Count  int `json:"count,omitempty"`
}

func apiOk(w http.ResponseWriter, r *http.Request, args ...any) {
	res := &RespDone{}
	if len(args) > 0 && args[0] != nil {
		res.Data = args[0]
		if len(args) > 1 {
			if c, ok := args[1].(int); ok {
				res.Count = c
			}
		}
	}

	render.JSON(w, r, res)
}
