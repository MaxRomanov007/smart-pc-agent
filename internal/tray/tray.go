package tray

import (
	"context"
	"log/slog"
	"smart-pc-agent/data/assets"
	"smart-pc-agent/internal/config"
	"smart-pc-agent/internal/services/updater"
	"smart-pc-agent/internal/services/waker"
	"smart-pc-agent/internal/tray/items/navigation"
	"smart-pc-agent/internal/tray/items/quit"
	"smart-pc-agent/internal/tray/items/updateritems"
	"smart-pc-agent/internal/tray/items/wakeritems"
	"smart-pc-agent/internal/tray/menu"

	"github.com/getlantern/systray"
)

func Start(
	ctx context.Context,
	log *slog.Logger,
	uiCfg config.UI,
	stop context.CancelFunc,
	waker *waker.Service,
	upd *updater.Service,
) *wakeritems.Events {
	events := wakeritems.NewEvents()

	items := []menu.Item{
		navigation.New(ctx, log, uiCfg),
		menu.NewSeparator(),
		updateritems.New(ctx, log, upd, stop),
		menu.NewSeparator(),
		wakeritems.New(ctx, log, waker, events),
		menu.NewSeparator(),
		quit.New(ctx, log),
	}

	go systray.Run(
		onReady(items),
		onExit(stop),
	)

	return events
}

func onReady(items []menu.Item) func() {
	return func() {
		systray.SetIcon(assets.GetIcon())
		systray.SetTitle("Smart PC")
		systray.SetTooltip("Smart PC Control Agent")

		for _, item := range items {
			item.Mount()
		}
	}
}

func onExit(stop context.CancelFunc) func() {
	return func() {
		stop()
	}
}
