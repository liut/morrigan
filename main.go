package main

import (
	"context"
	"embed"
	"flag"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/cupogo/andvari/utils/zlog"

	"github.com/liut/morrigan/pkg/settings"
	"github.com/liut/morrigan/pkg/web"
)

//go:embed htdocs
var static embed.FS

func main() {
	var usage bool
	flag.BoolVar(&usage, "usage", false, "show usage")
	flag.Parse()
	if usage {
		_ = settings.Usage()
		return
	}

	var zlogger *zap.Logger
	if settings.InDevelop() {
		zlogger, _ = zap.NewDevelopment()
	} else {
		zlogger, _ = zap.NewProduction()
	}
	sugar := zlogger.Sugar()
	zlog.Set(sugar)

	fsys := fs.FS(static)
	html, _ := fs.Sub(fsys, "htdocs")
	// srv.StaticFS("/", http.FS(html))
	srv := web.New(web.Config{
		Addr:       settings.Current.HTTPListen,
		Debug:      settings.InDevelop(),
		DocHandler: http.FileServer(http.FS(html)),
	})

	idleClosed := make(chan struct{})
	ctx := context.Background()
	go func() {
		quit := make(chan os.Signal, 2)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		sugar.Info("shuting down server...")
		if err := srv.Stop(ctx); err != nil {
			sugar.Infow("server shutdown:", "err", err)
		}
		close(idleClosed)
	}()

	if err := srv.Serve(ctx); err != nil {
		sugar.Infow("serve fali", "err", err)
	}

	<-idleClosed
}
