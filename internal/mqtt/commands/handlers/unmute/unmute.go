package unmute

import (
	"context"
	"log/slog"

	"github.com/MaxRomanov007/smart-pc-go-lib/commands"
	"github.com/MaxRomanov007/smart-pc-go-lib/domain/models/message"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/itchyny/volume-go"
)

func New(log *slog.Logger) commands.CommandFunc {
	return func(ctx context.Context, msg *message.Message) error {
		const op = "commands.handlers.unmute"

		log := log.With(sl.Op(op), sl.MsgID(msg.Publish))

		if err := volume.Unmute(); err != nil {
			log.Warn("failed to unmute", sl.Err(err))
			return commands.Error("failed to unmute")
		}

		return nil
	}
}
