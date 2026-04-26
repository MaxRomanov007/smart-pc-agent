package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"smart-pc-agent/data/assets"
	authorization "smart-pc-agent/internal/auth"
	"smart-pc-agent/internal/config"
	httpServer "smart-pc-agent/internal/http-server"
	"smart-pc-agent/internal/lib/logger"
	"smart-pc-agent/internal/lib/waitable"
	"smart-pc-agent/internal/mqtt"
	pcsService "smart-pc-agent/internal/services/pcs-service"
	"smart-pc-agent/internal/storage/sqlite"
	"syscall"

	"github.com/MaxRomanov007/smart-pc-go-lib/cross-platform/browser"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/getlantern/systray"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.MustLoad()

	logCtx, cancelLogCtx := context.WithCancel(context.Background())
	defer cancelLogCtx()
	log := logger.MustSetupLogger(logCtx, cfg.Env, cfg.LogPath)

	log.Debug("debug messages are enabled")

	storage, err := sqlite.New(ctx, log, cfg.Storage)
	if err != nil {
		log.Error("failed to create sqlite storage", sl.Err(err))
		os.Exit(1)
	}

	auth, err := authorization.New(ctx, cfg.Auth, storage.AppStorage, storage.AppStorage)
	if err != nil {
		log.Error("failed to create auth", sl.Err(err))
		os.Exit(1)
	}

	pcs, err := pcsService.New(ctx, auth, cfg.Services.Pcs, storage.AppStorage, storage.AppStorage)
	if err != nil {
		log.Error("failed to create pcs service", sl.Err(err))
		os.Exit(1)
	}

	mqttConn, err := mqtt.New(
		ctx,
		log,
		cfg.MQTT,
		auth,
		storage.AppStorage,
		storage.Commands,
		storage.CommandParameters,
	)
	if err != nil {
		log.Error("failed to create mqtt connection", sl.Err(err))
		os.Exit(1)
	}

	srv := httpServer.New(ctx, log, cfg.HTTPServer, storage, pcs, stop)
	go func() {
		if err := srv.Run(ctx); err != nil {
			log.Error("http server error", sl.Err(err))
			os.Exit(1)
		}
	}()

	go systray.Run(onTrayReady(ctx, log), onTrayExit(stop))

	waitable.WaitAll(mqttConn, srv)
}

func onTrayReady(ctx context.Context, log *slog.Logger) func() {
	return func() {
		systray.SetIcon(assets.GetIcon())
		systray.SetTitle("Smart PC")
		systray.SetTooltip("Control Agent")

		mOpenDashboard := systray.AddMenuItem("Open dashboard", "Open dashboard")
		mOpenDashboard.SetIcon(assets.GetHouse())
		mOpenInterface := systray.AddMenuItem("Open interface", "Open interface")
		mOpenInterface.SetIcon(assets.GetPcCase())
		mQuit := systray.AddMenuItem("Quit", "Quit")
		mQuit.SetIcon(assets.GetExit())

		go func() {
			const op = "tray"
			log := log.With(sl.Op(op))

			for {
				select {
				case <-ctx.Done():
					log.Info("context canceled")
					systray.Quit()
					return
				case <-mQuit.ClickedCh:
					log.Info("quit clicked")
					systray.Quit()
					return
				case <-mOpenInterface.ClickedCh:
					log.Info("open interface clicked")
					if err := browser.Open("http://localhost:3003/this-pc"); err != nil {
						log.Error("failed to open browser", sl.Err(err))
					}
				case <-mOpenDashboard.ClickedCh:
					log.Info("open dashboard clicked")
					if err := browser.Open("http://localhost:3003/dashboard"); err != nil {
						log.Error("failed to open browser", sl.Err(err))
					}
				}
			}
		}()
	}
}

func onTrayExit(stop context.CancelFunc) func() {
	return func() {
		stop()
	}
}
