package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

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

func importDocs(cc *cli.Context) error {
	input := cc.Args().First()
	file, err := os.Open(input)
	if err != nil {
		logger().Warnw("open fail", "input", input, "err", err)
		return err
	}
	defer file.Close()
	difflog := cc.String("diff")
	var lw *os.File // log write dat
	if len(difflog) > 0 {
		lw, err = os.Create(difflog)
		if err != nil {
			logger().Warnw("create fail", "difflog", difflog, "err", err)
			return err
		}
		defer lw.Close()
	} else {
		lw = os.Stderr
	}
	err = stores.Sgt().Qa().ImportDocs(context.Background(), file, lw)
	if err != nil {
		logger().Warnw("import fail", "input", input, "err", err)
		return err
	}
	return nil
}

func exportDocs(cc *cli.Context) error {
	output := cc.Args().First() // csv
	file, err := os.OpenFile(output, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		logger().Warnw("open fail", "output", output, "err", err)
		return err
	}
	defer file.Close()
	ctx := context.Background()
	spec := &stores.QaDocumentSpec{}
	spec.Limit = 90
	spec.Sort = "id"
	ea := stores.ExportArg{
		Spec:   spec,
		Out:    file,
		Format: cc.String("format"),
	}
	return stores.Sgt().Qa().ExportDocs(ctx, ea)
}

func embeddingDocVector(cc *cli.Context) error {
	ctx := context.Background()
	spec := &stores.QaDocumentSpec{}
	spec.Limit = 90
	spec.Sort = "id"
	return stores.Sgt().Qa().EmbeddingDocVector(ctx, spec)
	// return nil
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
		Usage:                  "A Backend for OpenAI/ChatGPT",
		UseShortOptionHandling: true,
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
				Name:   "import",
				Usage:  "import documents from a csv",
				Action: importDocs,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "diff", Aliases: []string{"diff-log"}, Value: "", Usage: "a filename of diff"},
				},
			},
			{
				Name:    "export",
				Usage:   "export documents to a csv",
				Aliases: []string{"exportDocs"},
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "format", Aliases: []string{"t"}, Value: "csv", Usage: "csv|jsonl"},
				},
				Action: exportDocs,
			},
			{
				Name:    "embedding",
				Usage:   "read prompt documents and embedding",
				Aliases: []string{"embedding-doc-vec"},
				Action:  embeddingDocVector,
			},
			{
				Name:    "web",
				Aliases: []string{"run"},
				Usage:   "run a web server",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "listen", Aliases: []string{"l"}, Value: settings.Current.HTTPListen, Usage: "http listen address"},
				},
				Action: webRun,
			},
			{
				Name: "version", Aliases: []string{"ver"},
				Usage: "show build version",
				Action: func(ctx *cli.Context) error {
					sugar.Infow("", "version", settings.Version(), "runtime", runtime.Version())
					return nil
				},
			},
		},
	}
	// if len(os.Args) < 2 {
	// 	webRun()
	// 	return
	// }
	if err := app.Run(os.Args); err != nil {
		logger().Fatalw("app run fail", "err", err)
	}
}

func webRun(cc *cli.Context) error {
	srv := web.New(web.Config{
		Addr:       cc.String("listen"),
		Debug:      settings.InDevelop(),
		DocHandler: http.FileServer(http.FS(htdocs.FS())),
	})

	ctx := context.Background()
	go func() {
		quit := make(chan os.Signal, 2)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		logger().Info("shuting down server...")
		if err := srv.Stop(ctx); err != nil {
			logger().Infow("server shutdown:", "err", err)
		}
	}()

	return srv.Serve(ctx)
}
