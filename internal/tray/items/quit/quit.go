package quit

import (
	"context"
	"log/slog"
	"smart-pc-agent/data/assets"
	appi18n "smart-pc-agent/internal/i18n"
	"smart-pc-agent/internal/tray/menu"

	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/getlantern/systray"
)

type Item struct {
	ctx context.Context
	log *slog.Logger
}

func New(ctx context.Context, log *slog.Logger) menu.Item {
	return &Item{
		ctx: ctx,
		log: log.With(sl.Op("tray.quit")),
	}
}

func (it *Item) Mount() {
	mQuit := systray.AddMenuItem(appi18n.MsgQuit(), appi18n.MsgQuitTooltip())
	mQuit.SetIcon(assets.GetExit())

	go it.handleClicks(mQuit)
}

func (it *Item) handleClicks(mQuit *systray.MenuItem) {
	for {
		select {
		case <-it.ctx.Done():
			systray.Quit()
			return
		case <-mQuit.ClickedCh:
			it.log.Info("quit clicked")
			systray.Quit()
			return
		}
	}
}
