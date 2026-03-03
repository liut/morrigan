package web

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/middleware/stdlib"
	limitRedis "github.com/ulule/limiter/v3/drivers/store/redis"

	staffio "github.com/liut/staffio-client"

	"github.com/liut/morrigan/pkg/services/stores"
	"github.com/liut/morrigan/pkg/settings"
	"github.com/liut/morrigan/pkg/web/routes"
)

type M = render.M

func (s *server) strapRouter(ar chi.Router) {

	ar.Get("/", handleNoContent)
	ar.Get("/api/ping", handlerPing)

	staffio.SetAdminPath("/")

	cch := (&staffio.CodeCallback{
		OnTokenGot: s.handleTokenGot,
	}).Handler()

	ar.Get(authLoginPath, staffio.LoginHandler)
	ar.Get(authLogoutPath, staffio.LogoutHandler)
	ar.Method(http.MethodGet, authCallbackPath, cch)

	ar.Route("/api", func(r chi.Router) {

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

		r.Use(s.authMw(false))
		r.Get("/me", handleMe)

		r.Get("/tools", s.getTools)
		r.Get("/welcome", s.getWelcome)
		r.Get("/history/{cid}", s.getHistory)
		r.Group(func(gr chi.Router) {
			gr.Use(middleware.Handler)
			gr.Post("/chat", s.postChat)
			gr.Post("/chat-{suffix}", s.postChat)
			gr.Post("/completions", s.postCompletions)
		})

		// 遍历 handles 注册路由
		pr := r.With(routes.AuthMw(false))
		for _, hi := range handles {
			if hi.auth {
				if len(hi.rid) > 0 {
					pr.With(authPerm(hi.rid)).Method(hi.method, hi.path, hi.hafn(s))
				} else {
					pr.Method(hi.method, hi.path, hi.hafn(s))
				}
			} else {
				r.Method(hi.method, hi.path, hi.hafn(s))
			}
		}
	})

	ar.Get("/api/session", s.handleSession)
	ar.Post("/api/session", s.handleSession)
	ar.Post("/api/verify", s.handleVerify)

	ar.Group(func(r chi.Router) {
		r.Use(s.authMw(true))
		if s.cfg.DocHandler != nil {
			r.Get("/", s.cfg.DocHandler.ServeHTTP)
		}
	})
	if s.cfg.DocHandler != nil {
		ar.NotFound(s.cfg.DocHandler.ServeHTTP)
	}
}

func apiFail(w http.ResponseWriter, r *http.Request, status int, err any) {
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
