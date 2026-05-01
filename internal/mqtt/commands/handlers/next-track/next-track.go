package nextTrack

import (
	"context"
	"log/slog"
	"smart-pc-agent/internal/lib/cross-platform/mediactl"

	"github.com/MaxRomanov007/smart-pc-go-lib/commands"
	commandMessage "github.com/MaxRomanov007/smart-pc-go-lib/domain/models/command-message"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
)

func New(log *slog.Logger) commands.CommandFunc {
	return func(ctx context.Context, msg *commandMessage.Message) error {
		const op = "commands.handlers.nextTrack"

		log := log.With(sl.Op(op), sl.MsgID(msg.Publish))

		if err := mediactl.NextTrack(); err != nil {
			log.Warn("failed to send next track", sl.Err(err))
			return commands.Error("failed to send next track")
		}

		return nil
	}
}
