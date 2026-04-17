package playPause

import (
	"context"
	"log/slog"
	"smart-pc-agent/internal/lib/cross-platform/mediactl"

	"github.com/MaxRomanov007/smart-pc-go-lib/commands"
	"github.com/MaxRomanov007/smart-pc-go-lib/domain/models/message"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
)

func New(log *slog.Logger) commands.CommandFunc {
	return func(ctx context.Context, msg *message.Message) error {
		const op = "commands.handlers.playPause"

		log := log.With(sl.Op(op), sl.MsgId(msg.Publish))

		if err := mediactl.PlayPause(); err != nil {
			log.Warn("failed to send play/pause", sl.Err(err))
			return commands.Error("failed to send play/pause")
		}

		return nil
	}
}
