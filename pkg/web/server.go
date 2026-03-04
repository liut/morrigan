package web

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/liut/morrigan/pkg/web/routes"
)

type Service interface {
	Serve(ctx context.Context) error
	Stop(ctx context.Context) error
}

type Config struct {
	Addr  string
	Debug bool

	DocHandler http.Handler
}

type server struct {
	Addr string
	cfg  Config

	ar *chi.Mux     // app router
	hs *http.Server // http server

}

// New return new web server
func New(cfg Config) Service {
	ar := chi.NewMux()
	if cfg.Debug {
		ar.Use(middleware.Logger)
	}
	ar.Use(middleware.Recoverer, middleware.RealIP)

	s := &server{
		Addr: cfg.Addr, ar: ar,
		cfg: cfg,
	}

	s.strapRouter(ar)

	s.hs = &http.Server{
		Addr:    s.Addr,
		Handler: s.ar,
	}

	if cfg.Debug {
		logger().Infow("routes:")
		walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
			route = strings.Replace(route, "/*/", "/", -1)
			fmt.Fprintf(os.Stderr, "DEBUG: %-6s %-24s --> %s (%d mw)\n", method, route, nameOfFunction(handler), len(middlewares))
			return nil
		}

		if err := chi.Walk(ar, walkFunc); err != nil {
			logger().Infow("router walk fail", "err", err)
		}
	}
	return s
}

func (s *server) Serve(ctx context.Context) error {
	// Run HTTP server
	runErrChan := make(chan error)
	t := time.AfterFunc(time.Millisecond*200, func() {
		runErrChan <- s.hs.ListenAndServe()
	})

	defer t.Stop()
	logger().Infow("Listen on", "addr", s.hs.Addr)

	// Wait
	for {
		select {
		case runErr := <-runErrChan:
			if runErr != nil {
				logger().Infow("run http server failed",
					"err", runErr,
				)
				return runErr
			}
		case <-ctx.Done():
			//TODO Graceful shutdown
			logger().Info("http server has been stopped")
			return ctx.Err()
		}
	}
}

func (s *server) Stop(ctx context.Context) error {
	if err := s.hs.Shutdown(ctx); err != nil {
		logger().Fatalw("Server Shutdown", "err", err)
		return err
	}
	return nil
}

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
