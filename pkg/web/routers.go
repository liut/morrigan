package web

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/marcsv/go-binder/binder"

	staffio "github.com/liut/staffio-client"

	"github.com/liut/morrigan/pkg/settings"
)

type M = render.M

// User online user
type User = staffio.User

// vars from staffio
var (
	SetLoginPath    = staffio.SetLoginPath
	SetAdminPath    = staffio.SetAdminPath
	UserFromContext = staffio.UserFromContext

	authzr staffio.Authorizer
)

func init() {
	authzr = staffio.NewAuth(staffio.WithRefresh(), staffio.WithURI(staffio.LoginPath), staffio.WithCookie(
		settings.Current.CookieName,
		settings.Current.CookiePath,
		settings.Current.CookieDomain,
	))
}

func (s *server) strapRouter() {

	s.ar.Get("/ping", handlerPing)

	s.ar.Route("/auth", func(r chi.Router) {
		r.Get("/login", staffio.LoginHandler)
		r.Get("/logout", staffio.LogoutHandler)
		r.Method(http.MethodGet, "/callback", staffio.AuthCodeCallback())
	})

	s.ar.Route("/api", func(r chi.Router) {
		r.Use(authzr.Middleware())
		r.Get("/me", handleMe)
		r.Get("/session", handleSession)
		r.Post("/session", handleSession)
		r.Post("/verify", handleVerify)

		r.Get("/models", s.getModels)
		r.Get("/welcome", s.getWelcome)
		r.Get("/history/{cid}", s.getHistory)
		r.Post("/chat", s.postChat)
		r.Post("/chat-{suffix}", s.postChat)
		r.Post("/completions", s.postCompletions)
		// r.Get("/status/{idx}", handlerStatus)
		// r.Post("/client/send", handlerSendClient)
	})

	// s.ar.Get("/", handleNoContent)
	SetAdminPath("/")
	s.ar.Group(func(r chi.Router) {
		r.Use(authzr.MiddlewareWordy(true))
		if s.cfg.DocHandler != nil {
			r.Get("/", s.cfg.DocHandler.ServeHTTP)
		}

		// r.Get("/", handlerHome)
	})
	if s.cfg.DocHandler != nil {
		s.ar.NotFound(s.cfg.DocHandler.ServeHTTP)
	}
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
	if user, ok := UserFromContext(r.Context()); ok {
		apiOk(w, r, user)
	} else {
		apiFail(w, r, 401, "not login")
	}
}

func handleSession(w http.ResponseWriter, r *http.Request) {
	if _, ok := UserFromContext(r.Context()); ok {
		render.JSON(w, r, M{"data": M{
			"auth": true,
			// "user": user,
		}, "status": "Success"})
	} else {
		apiFail(w, r, 401, "not login")
	}
}

type verifyReq struct {
	Token string `json:"token"`
}

func handleVerify(w http.ResponseWriter, r *http.Request) {
	var param verifyReq
	if err := binder.BindBody(r, &param); err != nil {
		apiFail(w, r, 400, err)
		return
	}
	user := new(User)
	if err := user.Decode(param.Token); err != nil {
		apiFail(w, r, 401, err)
		return
	}

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
