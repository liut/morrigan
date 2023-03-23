package web

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	staffio "github.com/liut/staffio-client"
	"github.com/sashabaranov/go-openai"

	"github.com/liut/morrigan/pkg/models/conversatio"
	"github.com/liut/morrigan/pkg/settings"
	"github.com/liut/morrigan/pkg/sevices/stores"
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

	authzr staffio.Authorizer
	oc     *openai.Client
	preset *conversatio.Preset
}

// New return new web server
func New(cfg Config) Service {
	ar := chi.NewMux()
	if cfg.Debug {
		ar.Use(middleware.Logger)
	}
	ar.Use(middleware.Recoverer)

	occ := openai.DefaultConfig(settings.Current.OpenAIAPIKey)
	occ.HTTPClient = &http.Client{
		Timeout:   time.Second * 20,
		Transport: &http.Transport{Proxy: http.ProxyFromEnvironment},
	}

	s := &server{
		Addr: cfg.Addr, ar: ar,
		cfg: cfg,
		oc:  openai.NewClientWithConfig(occ),
	}
	if settings.Current.AuthRequired {
		s.authzr = staffio.NewAuth(staffio.WithRefresh(), staffio.WithURI(staffio.LoginPath), staffio.WithCookie(
			settings.Current.CookieName,
			settings.Current.CookiePath,
			settings.Current.CookieDomain,
		))
	}
	var err error
	s.preset, err = stores.LoadPreset()
	if err == nil {
		logger().Infow("loaded preset", "messages", len(s.preset.Messages))
	}
	s.strapRouter()

	s.hs = &http.Server{
		Addr:    s.Addr,
		Handler: s.ar,
	}

	if cfg.Debug {
		walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
			route = strings.Replace(route, "/*/", "/", -1)
			fmt.Printf("DEBUG: %-6s %-24s --> %s (%d mw)\n", method, route, nameOfFunction(handler), len(middlewares))
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
