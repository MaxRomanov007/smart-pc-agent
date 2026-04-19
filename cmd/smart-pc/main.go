package main

import (
	"context"
	"os"
	"os/signal"
	authorization "smart-pc-agent/internal/auth"
	"smart-pc-agent/internal/config"
	httpServer "smart-pc-agent/internal/http-server"
	"smart-pc-agent/internal/lib/logger"
	"smart-pc-agent/internal/lib/waitable"
	"smart-pc-agent/internal/mqtt"
	pcsService "smart-pc-agent/internal/services/pcs-service"
	"smart-pc-agent/internal/storage/sqlite"
	"syscall"

	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
)

func main() {
	cfg := config.MustLoad()
	log := logger.MustSetupLogger(cfg.Env)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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

	mqttConn, err := mqtt.New(ctx, log, cfg.MQTT, auth, storage.Commands, storage.CommandParameters)
	if err != nil {
		log.Error("failed to create mqtt connection", sl.Err(err))
		os.Exit(1)
	}

	pcs, err := pcsService.New(ctx, auth, cfg.Services.Pcs, storage.AppStorage, storage.AppStorage)
	if err != nil {
		log.Error("failed to create pcs service", sl.Err(err))
		os.Exit(1)
	}

	srv := httpServer.New(ctx, log, cfg.HTTPServer, pcs, pcs, storage.Commands)
	go func() {
		if err := srv.Run(ctx); err != nil {
			log.Error("http server error", sl.Err(err))
			os.Exit(1)
		}
	}()

	waitable.WaitAll(mqttConn, srv)
}
