package powerOff

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
	appi18n "smart-pc-agent/internal/i18n"
	"time"

	"github.com/MaxRomanov007/smart-pc-go-lib/commands"
	commandMessage "github.com/MaxRomanov007/smart-pc-go-lib/domain/models/command-message"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/ncruces/zenity"
)

const (
	seconds       = 10
	stepsBySecond = 2
)

func New(log *slog.Logger) commands.CommandFunc {
	return func(ctx context.Context, msg *commandMessage.Message) error {
		const op = "commands.handlers.power-off"
		log := log.With(sl.Op(op), sl.MsgID(msg.Publish))

		dlg, err := zenity.Progress(
			zenity.Title(appi18n.MsgPowerOffTitle()),
			zenity.CancelLabel(appi18n.MsgPowerOffCancelLabel()),
		)
		if err != nil {
			log.Error("failed to initialize progress dialog")
			return fmt.Errorf("%s: failed to initialize progress dialog: %w", op, err)
		}
		defer func(dlg zenity.ProgressDialog) {
			_ = dlg.Close()
		}(dlg)

		for i := 0.0; i < seconds; i += 1.0 / stepsBySecond {
			_ = dlg.Text(appi18n.MsgPowerOffCountdown(int(seconds - i)))

			if err := dlg.Value(
				int((i + 1.0/stepsBySecond) / seconds * 100),
			); errors.Is(
				err,
				zenity.ErrCanceled,
			) {
				log.Info("canceled by user")
				return commands.Error("canceled by user")
			}

			time.Sleep(time.Second / stepsBySecond)
		}

		_ = dlg.Text(appi18n.MsgPowerOffShuttingDown())
		if err := dlg.Complete(); errors.Is(
			err,
			zenity.ErrCanceled,
		) {
			log.Info("canceled by user (on completion)")
			return commands.Error("canceled by user")
		}
		time.Sleep(500 * time.Millisecond)
		_ = dlg.Close()

		if err := shutdown(); err != nil {
			_ = zenity.Error(
				appi18n.MsgPowerOffError(err),
				zenity.Title(appi18n.MsgPowerOffErrorTitle()),
			)
		}

		return nil
	}
}

func shutdown() error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("shutdown", "/s", "/t", "0")
	case "darwin":
		cmd = exec.Command("osascript", "-e", `tell app "System Events" to shut down`)
	default: // linux, freebsd, etc.
		cmd = exec.Command("shutdown", "-h", "now")
	}
	return cmd.Run()
}
