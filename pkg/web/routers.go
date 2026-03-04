package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/liut/morrigan/pkg/web/routes"
)

type M = render.M

func (s *server) strapRouter(ar chi.Router) {

	ar.Get("/", handleNoContent)

	// API 路由 - 通过 routes.Routers 注册
	routes.Routers(ar)

	ar.Group(func(r chi.Router) {
		r.Use(routes.AuthMw(true))
		if s.cfg.DocHandler != nil {
			r.Get("/", s.cfg.DocHandler.ServeHTTP)
		}
	})
	if s.cfg.DocHandler != nil {
		ar.NotFound(s.cfg.DocHandler.ServeHTTP)
	}
}

// nolint
func handleNoContent(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(204)
}
