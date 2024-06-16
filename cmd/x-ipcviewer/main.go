package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ItsNotGoodName/x-ipcviewer/internal/build"
	"github.com/ItsNotGoodName/x-ipcviewer/internal/bus"
	"github.com/ItsNotGoodName/x-ipcviewer/internal/config"
	"github.com/ItsNotGoodName/x-ipcviewer/internal/xwm"
	"github.com/danielgtaylor/huma/v2/humacli"
	_ "github.com/gen2brain/go-mpv"
	"github.com/jezek/xgb"
	"github.com/joho/godotenv"
	"github.com/phsym/console-slog"
)

type Options struct {
	Debug  bool   `doc:"enable debug"`
	Host   string `doc:"host to listen on"`
	Port   int    `doc:"port to listen on" default:"8080"`
	Config string `doc:"config file" default:".x-ipcviewer.yaml"`
}

func main() {
	godotenv.Load()

	cli := humacli.New(func(hooks humacli.Hooks, options *Options) {
		if options.Debug {
			InitLogger(slog.LevelDebug)
		} else {
			InitLogger(slog.LevelInfo)
		}

		OnServe(hooks, func(ctx context.Context) error {
			bus.SetContext(ctx)

			configFilePath, err := filepath.Abs(options.Config)
			if err != nil {
				return err
			}

			provider, err := config.NewProvider(configFilePath)
			if err != nil {
				return err
			}

			if err := xwm.NormalizeConfig(provider); err != nil {
				return err
			}

			conn, err := xgb.NewConn()
			if err != nil {
				return err
			}
			defer conn.Close()

			state, err := xwm.SetupState(conn, provider)
			if err != nil {
				return err
			}

			eventC := make(chan any)
			go xwm.ReceiveEvents(ctx, conn, eventC)

			return xwm.HandleEvents(ctx, conn, state, eventC)
		})
	})

	cli.Root().Version = build.Current.Version

	cli.Run()
}

func InitLogger(level slog.Level) {
	slog.SetDefault(slog.New(console.NewHandler(os.Stderr, &console.HandlerOptions{
		Level: level,
	})))
}

func OnServe(hooks humacli.Hooks, serveFn func(ctx context.Context) error) {
	stopC := make(chan struct{})
	hooks.OnStart(func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		errC := make(chan error, 1)

		go func() { errC <- serveFn(ctx) }()

		select {
		case <-stopC:
			cancel()
		case err := <-errC:
			if err != nil && !errors.Is(err, context.Canceled) {
				log.Fatal(err)
			}
			return
		}

		<-errC
		<-stopC
	})
	hooks.OnStop(func() {
		stopC <- struct{}{}
		stopC <- struct{}{}
	})
}
