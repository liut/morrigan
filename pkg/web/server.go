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
	staffio "github.com/liut/staffio-client"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sashabaranov/go-openai"

	"github.com/liut/morrigan/pkg/models/aigc"
	"github.com/liut/morrigan/pkg/services/stores"
	"github.com/liut/morrigan/pkg/settings"
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

	sto stores.Storage

	ar *chi.Mux     // app router
	hs *http.Server // http server

	cmodel string // openAI chat model
	authzr staffio.Authorizer
	oc     *openai.Client
	preset aigc.Preset
	mcpcs  map[string]client.MCPClient // with token key, like session
	tools  []mcp.Tool                  // preset tools
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
		cfg:    cfg,
		sto:    stores.Sgt(),
		oc:     stores.GetInteractAIClient(),
		cmodel: settings.Current.ChatModel,
		mcpcs:  make(map[string]client.MCPClient),
	}
	s.initTools()

	s.authzr = staffio.NewAuth(staffio.WithCookie(
		settings.Current.CookieName,
		settings.Current.CookiePath,
		settings.Current.CookieDomain,
	), staffio.WithRefresh(), staffio.WithURI(staffio.LoginPath))

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

func (s *server) initTools() {
	s.tools = append(s.tools,
		mcp.NewTool(ToolNameKBSearch,
			mcp.WithDescription("Search knowledge base with text of keywords or subject"),
			mcp.WithString("subject", mcp.Required(), mcp.Description("text of keywords or subject")),
		),
	)
	logger().Debugw("init tools", "tools", s.tools)
}
