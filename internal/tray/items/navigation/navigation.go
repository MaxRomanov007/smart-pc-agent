package navigation

import (
	"context"
	"log/slog"
	"smart-pc-agent/data/assets"
	"smart-pc-agent/internal/config"
	appi18n "smart-pc-agent/internal/i18n"
	"smart-pc-agent/internal/tray/menu"

	"github.com/MaxRomanov007/smart-pc-go-lib/cross-platform/browser"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/getlantern/systray"
)

type Item struct {
	ctx   context.Context
	log   *slog.Logger
	uiCfg config.UI
}

func New(ctx context.Context, log *slog.Logger, uiCfg config.UI) menu.Item {
	return &Item{
		ctx:   ctx,
		log:   log.With(sl.Op("tray.navigation")),
		uiCfg: uiCfg,
	}
}

func (it *Item) Mount() {
	mDashboard := systray.AddMenuItem(appi18n.MsgOpenDashboard(), appi18n.MsgOpenDashboard())
	mDashboard.SetIcon(assets.GetHouse())

	mInterface := systray.AddMenuItem(appi18n.MsgOpenInterface(), appi18n.MsgOpenInterface())
	mInterface.SetIcon(assets.GetPcCase())

	go it.handleClicks(mDashboard, mInterface)
}

func (it *Item) handleClicks(mDashboard, mInterface *systray.MenuItem) {
	for {
		select {
		case <-it.ctx.Done():
			return
		case <-mDashboard.ClickedCh:
			it.log.Info("open dashboard clicked")
			if err := browser.Open(it.uiCfg.BaseURL + "/dashboard"); err != nil {
				it.log.Error("failed to open dashboard", sl.Err(err))
			}
		case <-mInterface.ClickedCh:
			it.log.Info("open interface clicked")
			if err := browser.Open(it.uiCfg.BaseURL + "/this-pc"); err != nil {
				it.log.Error("failed to open interface", sl.Err(err))
			}
		}
	}
}
