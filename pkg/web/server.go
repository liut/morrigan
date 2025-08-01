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

type ToolCallFunc = func(ctx context.Context, params map[string]any) (mcp.Content, error)

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

	invokers map[string]ToolCallFunc
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
		cfg:      cfg,
		sto:      stores.Sgt(),
		oc:       stores.GetInteractAIClient(),
		cmodel:   settings.Current.ChatModel,
		mcpcs:    make(map[string]client.MCPClient),
		invokers: make(map[string]ToolCallFunc),
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
		logger().Infow("loaded preset", "mcps", len(s.preset.MCPServers))
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
			mcp.WithDescription("Search documents in knowledge base with keywords or subject"),
			mcp.WithString("subject", mcp.Required(), mcp.Description("text of keywords or subject")),
		),
		mcp.NewTool(ToolNameKBCreate,
			mcp.WithDescription("Create new document of knowledge base. Note, this is a write operation, all parameters are required."),
			mcp.WithString("title", mcp.Required(), mcp.Description("title of document, like a main name or topic")),
			mcp.WithString("heading", mcp.Required(), mcp.Description("heading of document, like a sub name or property")),
			mcp.WithString("content", mcp.Required(), mcp.Description("long text of content of document")),
		),
		mcp.NewTool(ToolNameFetch,
			mcp.WithDescription("Fetches a URL from the internet and optionally extracts its contents as markdown"),
			mcp.WithString("url",
				mcp.Required(),
				mcp.Description("URL to fetch"),
			),
			mcp.WithNumber("max_length",
				mcp.DefaultNumber(5000),
				mcp.Description("Maximum number of characters to return, default 5000"),
				mcp.Min(0),
				mcp.Max(1000000),
			),
			mcp.WithNumber("start_index",
				mcp.DefaultNumber(0),
				mcp.Description("On return output starting at this character index, default 0"),
				mcp.Min(0),
			),
			mcp.WithBoolean("raw",
				mcp.DefaultBool(false),
				mcp.Description("Get the actual HTML content without simplification, dfault false"),
			),
		),
	)
	logger().Debugw("init tools", "tools", s.tools)
	s.invokers = map[string]ToolCallFunc{
		ToolNameKBSearch: s.callKBSearch,
		ToolNameKBCreate: s.callKBCreate,
		ToolNameFetch:    s.callFetch,
	}
}

func (s *server) invokeTool(ctx context.Context, toolName string, params map[string]any) (mcp.Content, error) {
	if toolName == "" {
		return mcp.NewTextContent("tool name is empty"), nil
	}
	logger().Debugw("invoking", "toolName", toolName, "params", params)
	for key, vfn := range s.invokers {
		if strings.EqualFold(key, toolName) {
			return vfn(ctx, params)
		}
	}
	// TODO: check if tool is in s.tools
	return mcp.NewTextContent("tool not found"), nil
}
