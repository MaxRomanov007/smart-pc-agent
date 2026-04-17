package setVolume

import (
	"context"
	"log/slog"

	"github.com/MaxRomanov007/smart-pc-go-lib/commands"
	"github.com/MaxRomanov007/smart-pc-go-lib/domain/models/message"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/itchyny/volume-go"
)

type Parameter struct {
	Volume int `json:"volume"`
}

func New(log *slog.Logger) commands.CommandFunc {
	return func(ctx context.Context, msg *message.Message) error {
		const op = "commands.handlers.set-volume"

		log := log.With(sl.Op(op), sl.MsgId(msg.Publish))

		parameter, err := message.Parameter[Parameter](msg)
		if err != nil {
			log.Warn(
				"failed to parse message parameter",
				slog.Any("parameter", msg.Data.Parameter),
				sl.Err(err),
			)
			return commands.Error("failed to get volume")
		}

		log.Info("got volume level", slog.Int("volume", parameter.Volume))

		if err := volume.SetVolume(parameter.Volume); err != nil {
			log.Warn("failed to set volume", slog.Int("volume", parameter.Volume), sl.Err(err))
			return commands.Error("failed to set volume")
		}

		return nil
	}
}
