package main

import (
	"context"
	"os"
	"os/signal"
	authorization "smart-pc-agent/internal/auth"
	"smart-pc-agent/internal/config"
	httpServer "smart-pc-agent/internal/http-server"
	appi18n "smart-pc-agent/internal/i18n"
	"smart-pc-agent/internal/lib/logger"
	luaApi "smart-pc-agent/internal/lib/lua-api"
	"smart-pc-agent/internal/mqtt"
	luaLog "smart-pc-agent/internal/mqtt/commands/lua-api/log"
	luaMessages "smart-pc-agent/internal/mqtt/commands/lua-api/messages"
	pcsService "smart-pc-agent/internal/services/pcs-service"
	"smart-pc-agent/internal/services/updater"
	wakerService "smart-pc-agent/internal/services/waker"
	"smart-pc-agent/internal/storage/sqlite"
	"smart-pc-agent/internal/tray"
	"syscall"

	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/MaxRomanov007/smart-pc-go-lib/waitable"
)

// version is set at build time via -ldflags "-X main.version=v1.2.3"
var version = "v0.0.0"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.MustLoad()

	logCtx, cancelLogCtx := context.WithCancel(context.Background())
	defer cancelLogCtx()
	log := logger.MustSetupLogger(logCtx, cfg.Env, string(cfg.LogPath))

	log.Debug("debug messages are enabled")

	appi18n.Init(log)

	waker, err := wakerService.New(ctx, cfg.Services.Waker, log)
	if err != nil {
		log.Error("failed to create waker service", sl.Err(err))
		os.Exit(1)
	}

	upd := updater.New(ctx, log, version)

	wakerEvents := tray.Start(ctx, log, cfg.UI, stop, waker, upd)

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

	waker.SetPcID(pcs.PcID)

	registry := luaApi.NewRegistry(version).
		Register("log", luaLog.New(log)).
		Register("messages", luaMessages.New())

	mqttConn, err := mqtt.New(
		ctx,
		log,
		cfg.MQTT,
		auth,
		registry,
		storage.AppStorage,
		storage.Commands,
		storage.CommandParameters,
	)
	if err != nil {
		log.Error("failed to create mqtt connection", sl.Err(err))
		os.Exit(1)
	}
	go func() {
		<-mqttConn.Done()
		log.Info("mqtt connection closed")
	}()

	srv := httpServer.New(
		ctx,
		log,
		cfg.HTTPServer,
		storage,
		pcs,
		waker,
		registry,
		stop,
		wakerEvents,
	)
	go func() {
		if err := srv.Run(ctx); err != nil {
			log.Error("http server error", sl.Err(err))
			os.Exit(1)
		}
	}()

	waitable.WaitAll(mqttConn, srv, waker)
}
