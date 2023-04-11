package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/cupogo/andvari/models/oid"
	"github.com/cupogo/andvari/utils/zlog"

	"github.com/liut/morrigan/htdocs"
	"github.com/liut/morrigan/pkg/services/stores"
	"github.com/liut/morrigan/pkg/settings"
	"github.com/liut/morrigan/pkg/web"
)

func usage(cc *cli.Context) error {
	return settings.Usage()
}

func initdb(cc *cli.Context) error {
	return stores.InitDB()
}

func embedding(cc *cli.Context) error {
	input := cc.Args().First()
	file, err := os.Open(input)
	if err != nil {
		logger().Warnw("open fail", "input", input, "err", err)
		return err
	}
	defer file.Close()
	err = stores.Sgt().Qa().ImportFromCSV(context.Background(), file)
	if err != nil {
		logger().Warnw("import fail", "input", input, "err", err)
		return err
	}
	return nil
}

func fillQAs(cc *cli.Context) error {
	ctx := context.Background()
	spec := &stores.DocumentSpec{}
	if cc.Args().Len() > 0 {
		spec.IDsStr = oid.OIDsStr(strings.Join(cc.Args().Slice(), ","))
	}
	spec.Limit = 90
	spec.Sort = "id"
	return stores.Sgt().Qa().FillQAs(ctx, spec)
}

func exportQAs(cc *cli.Context) error {
	output := cc.Args().First() // csv
	file, err := os.OpenFile(output, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		logger().Warnw("open fail", "output", output, "err", err)
		return err
	}
	defer file.Close()
	ctx := context.Background()
	spec := &stores.DocumentSpec{}
	spec.Limit = 90
	spec.Sort = "id"
	return stores.Sgt().Qa().ExportQAs(ctx, spec, file)
}

func logger() zlog.Logger {
	return zlog.Get()
}

func main() {

	var zlogger *zap.Logger
	if settings.InDevelop() {
		zlogger, _ = zap.NewDevelopment()
	} else {
		zlogger, _ = zap.NewProduction()
	}
	sugar := zlogger.Sugar()
	zlog.Set(sugar)

	app := &cli.App{
		Usage: "A Backend for OpenAI/ChatGPT",
		Commands: []*cli.Command{
			{
				Name:    "usage",
				Aliases: []string{"env"},
				Usage:   "show usage",
				Action:  usage,
			},
			{
				Name:   "initdb",
				Usage:  "init database schema",
				Action: initdb,
			},
			{
				Name:   "embedding",
				Usage:  "input a csv to embedding",
				Action: embedding,
			},
			{
				Name:    "fill-qa",
				Usage:   "fill QA in documents",
				Aliases: []string{"fillQAs"},
				Action:  fillQAs,
			},
			{
				Name:    "export-qa",
				Usage:   "export QA from documents",
				Aliases: []string{"exportQAs"},
				Action:  exportQAs,
			},
			{
				Name:    "web",
				Aliases: []string{"run"},
				Usage:   "run a web server",
				Action: func(cCtx *cli.Context) error {
					webRun()
					return nil
				},
			},
		},
	}
	if len(os.Args) < 2 {
		webRun()
		return
	}
	if err := app.Run(os.Args); err != nil {
		logger().Fatalw("app run fail", "err", err)
	}
}

func webRun() {
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
		logger().Info("shuting down server...")
		if err := srv.Stop(ctx); err != nil {
			logger().Infow("server shutdown:", "err", err)
		}
		close(idleClosed)
	}()

	if err := srv.Serve(ctx); err != nil {
		logger().Infow("serve fali", "err", err)
	}

	<-idleClosed
}
