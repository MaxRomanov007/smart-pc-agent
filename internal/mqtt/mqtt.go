package mqtt

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"smart-pc-agent/internal/config"
	luaApi "smart-pc-agent/internal/lib/lua-api"
	"smart-pc-agent/internal/lib/random"
	executeScript "smart-pc-agent/internal/mqtt/commands/handlers/execute-script"
	"smart-pc-agent/internal/mqtt/commands/handlers/mute"
	nextTrack "smart-pc-agent/internal/mqtt/commands/handlers/next-track"
	playPause "smart-pc-agent/internal/mqtt/commands/handlers/play-pause"
	prevTrack "smart-pc-agent/internal/mqtt/commands/handlers/prev-track"
	setVolume "smart-pc-agent/internal/mqtt/commands/handlers/set-volume"
	"smart-pc-agent/internal/mqtt/commands/handlers/unmute"

	"github.com/MaxRomanov007/smart-pc-go-lib/authorization"
	"github.com/MaxRomanov007/smart-pc-go-lib/commands"
	mqttAuth "github.com/MaxRomanov007/smart-pc-go-lib/mqtt-auth"
	"github.com/eclipse/paho.golang/paho"
)

const clientIDPostfixLength = 6

type MQTT struct {
	Connection *mqttAuth.Connection
}

type PcIDGetter interface {
	GetPcID(ctx context.Context) (string, error)
}

func New(
	ctx context.Context,
	log *slog.Logger,
	mqttCfg config.MQTT,
	auth *authorization.Auth,
	registry *luaApi.Registry,
	pcIDGetter PcIDGetter,
	commandGetter executeScript.CommandGetter,
	commandParamsGetter executeScript.CommandParamsGetter,
) (*MQTT, error) {
	const op = "mqtt.New"

	localCtx, cancel := context.WithCancel(context.Background())

	mqttConnCfg, router, err := createMQTTConfig(localCtx, mqttCfg, auth)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("%s: failed to create mqtt config: %w", op, err)
	}

	connection, err := mqttAuth.NewConnection(localCtx, mqttConnCfg)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("%s: failed to create mqtt connection: %w", op, err)
	}

	pcID, err := pcIDGetter.GetPcID(localCtx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("%s: failed to get pc id: %w", op, err)
	}

	startSendState(ctx, localCtx, pcID, log, connection, cancel)

	executor := commands.NewExecutor(connection, router)
	executor.SetDefault(executeScript.New(log, commandGetter, commandParamsGetter, registry))
	executor.Set("mute", mute.New(log))
	executor.Set("unmute", unmute.New(log))
	executor.Set("set-volume", setVolume.New(log))
	executor.Set("play-pause", playPause.New(log))
	executor.Set("next-track", nextTrack.New(log))
	executor.Set("prev-track", prevTrack.New(log))

	if err := executor.StartListen(localCtx, &commands.StartListenOptions{
		CommandTopic:       fmt.Sprintf("pcs/%s/command", pcID),
		CommandMessageType: "command",
		LogTopic:           fmt.Sprintf("pcs/%s/log", pcID),
		LogMessageType:     "pc-command-log",
		Log:                log,
	}); err != nil {
		cancel()
		return nil, fmt.Errorf("%s: failed to start listening commands: %w", op, err)
	}

	return &MQTT{
		Connection: connection,
	}, nil
}

func createMQTTConfig(
	ctx context.Context,
	mqttCfg config.MQTT,
	auth *authorization.Auth,
) (*mqttAuth.ClientConfig, *mqttAuth.Router, error) {
	const op = "mqtt.createMQTTConfig"

	cfg, router, err := mqttAuth.NewClientConfigWithRouter(ctx, auth)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"%s: failed to create new client config with router: %w",
			op,
			err,
		)
	}

	broker, err := url.Parse(mqttCfg.BrokerURL)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: failed to parse broker url: %w", op, err)
	}

	cfg.ClientConfig.ClientID = mqttCfg.ClientIDPrefix + random.String(clientIDPostfixLength)
	cfg.ServerUrls = []*url.URL{broker}
	cfg.CleanStartOnInitialConnection = false
	cfg.SessionExpiryInterval = mqttCfg.SessionExpiryInterval
	cfg.KeepAlive = mqttCfg.KeepAlive

	cfg.SetWill(&paho.WillMessage{
		QoS:     1,
		Retain:  true,
		Topic:   "pcs/hello/status",
		Payload: []byte("{\"type\":\"pc-status\",\"data\":{\"status\":\"offline\"}}"),
	})

	return cfg, router, nil
}

func (m *MQTT) Done() <-chan struct{} {
	return m.Connection.Done()
}
