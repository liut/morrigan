package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/cupogo/andvari/utils/zlog"

	"github.com/liut/morrigan/htdocs"
	"github.com/liut/morrigan/pkg/services/stores"
	"github.com/liut/morrigan/pkg/settings"
	"github.com/liut/morrigan/pkg/web"
)

func main() {
	var usage bool
	flag.BoolVar(&usage, "usage", false, "show usage")
	var initdb bool
	flag.BoolVar(&initdb, "initdb", false, "init database")
	var input string
	flag.StringVar(&input, "input", "", "input file")
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

	if initdb {
		_ = stores.InitDB()
		return
	}

	if len(input) > 0 {
		file, err := os.Open(input)
		if err != nil {
			sugar.Warnw("open fail", "input", input, "err", err)
			return
		}
		defer file.Close()
		err = stores.Sgt().Qa().ImportFromCSV(context.Background(), file)
		if err != nil {
			sugar.Warnw("import fail", "input", input, "err", err)
			return
		}
		return
	}

	srv := web.New(web.Config{
		Addr:       settings.Current.HTTPListen,
		Debug:      settings.InDevelop(),
		DocHandler: http.FileServer(http.FS(htdocs.FS())),
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
