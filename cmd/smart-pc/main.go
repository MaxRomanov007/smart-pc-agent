package main

import (
	"context"
	"database/sql"
	"net/url"
	"os/signal"
	executeScript "smart-pc-agent/internal/commands/handlers/execute-script"
	"smart-pc-agent/internal/commands/handlers/mute"
	nextTrack "smart-pc-agent/internal/commands/handlers/next-track"
	playPause "smart-pc-agent/internal/commands/handlers/play-pause"
	prevTrack "smart-pc-agent/internal/commands/handlers/prev-track"
	setVolume "smart-pc-agent/internal/commands/handlers/set-volume"
	"smart-pc-agent/internal/commands/handlers/unmute"
	"smart-pc-agent/internal/config"
	"smart-pc-agent/internal/lib/logger"
	"smart-pc-agent/internal/storage/sqlite/dbqueries"
	"syscall"

	"github.com/MaxRomanov007/smart-pc-go-lib/authorization"
	"github.com/MaxRomanov007/smart-pc-go-lib/commands"
	mqttAuth "github.com/MaxRomanov007/smart-pc-go-lib/mqtt-auth"
	"github.com/eclipse/paho.golang/paho"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	cfg := config.MustLoad()
	log := logger.MustSetupLogger(cfg.Env)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Debug("debug messages are enabled")

	db, err := sql.Open("sqlite3", "./data/database/db.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	queries := dbqueries.New(db)

	auth, err := createAuth(log, queries)
	if err != nil {
		panic(err)
	}

	mqttCfg, router, err := createMQTTConfig(ctx, auth)
	if err != nil {
		panic(err)
	}
	mqttCfg.SetWill(&paho.WillMessage{
		QoS:     1,
		Retain:  true,
		Topic:   "pcs/hello/status",
		Payload: []byte("{\"type\":\"pc-status\",\"data\":{\"status\":\"offline\"}}"),
	})

	connCtx, cancel := context.WithCancel(context.Background())

	connection, err := mqttAuth.NewConnection(connCtx, mqttCfg)
	if err != nil {
		panic(err)
	}

	if _, err := connection.Publish(connCtx, &paho.Publish{
		QoS:     1,
		Retain:  true,
		Topic:   "pcs/hello/status",
		Payload: []byte("{\"type\":\"pc-status\",\"data\":{\"status\":\"online\"}}"),
	}); err != nil {
		panic(err)
	}

	startSendState(connCtx, log, connection)

	executor := commands.NewExecutor(connection, router)
	executor.SetDefault(executeScript.New(log, queries))
	executor.Set("mute", mute.New(log))
	executor.Set("unmute", unmute.New(log))
	executor.Set("set-volume", setVolume.New(log))
	executor.Set("play-pause", playPause.New(log))
	executor.Set("next-track", nextTrack.New(log))
	executor.Set("prev-track", prevTrack.New(log))

	if err := executor.StartListen(connCtx, &commands.StartListenOptions{
		CommandTopic:       "pcs/hello/command",
		CommandMessageType: "command",
		LogTopic:           "pcs/hello/log",
		LogMessageType:     "pc-command-log",
		Log:                log,
	}); err != nil {
		panic(err)
	}

	<-ctx.Done()

	if _, err := connection.Publish(connCtx, &paho.Publish{
		QoS:     1,
		Retain:  true,
		Topic:   "pcs/hello/status",
		Payload: []byte("{\"type\":\"pc-status\",\"data\":{\"status\":\"offline\"}}"),
	}); err != nil {
		panic(err)
	}

	cancel()

	<-connection.Done()
}

func createMQTTConfig(
	ctx context.Context,
	auth *authorization.Auth,
) (*mqttAuth.ClientConfig, *mqttAuth.Router, error) {
	cfg, router, err := mqttAuth.NewClientConfigWithRouter(ctx, auth)
	if err != nil {
		return nil, nil, err
	}

	broker, err := url.Parse("mqtt://localhost:1883")
	if err != nil {
		return nil, nil, err
	}

	cfg.ClientConfig.ClientID = "smart-pc-cmd"
	cfg.ServerUrls = []*url.URL{broker}
	cfg.CleanStartOnInitialConnection = false
	cfg.SessionExpiryInterval = 60
	cfg.KeepAlive = 20

	return cfg, router, nil
}
