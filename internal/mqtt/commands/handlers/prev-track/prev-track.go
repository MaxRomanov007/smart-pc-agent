package prevTrack

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
		const op = "commands.handlers.prevTrack"

		log := log.With(sl.Op(op), sl.MsgID(msg.Publish))

		if err := mediactl.PrevTrack(); err != nil {
			log.Warn("failed to send prev track", sl.Err(err))
			return commands.Error("failed to send prev track")
		}

		return nil
	}
}
