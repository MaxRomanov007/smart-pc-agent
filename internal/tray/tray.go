package tray

import (
	"context"
	"errors"
	"log/slog"
	"smart-pc-agent/data/assets"
	"smart-pc-agent/internal/config"
	"smart-pc-agent/internal/lib/installer"
	"smart-pc-agent/internal/services/waker"

	"github.com/MaxRomanov007/smart-pc-go-lib/cross-platform/browser"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/getlantern/systray"
	"github.com/ncruces/zenity"
)

func Start(
	ctx context.Context,
	log *slog.Logger,
	uiCfg config.UI,
	stop context.CancelFunc,
	waker *waker.Service,
) {
	go systray.Run(onTrayReady(ctx, log, uiCfg, waker), onTrayExit(stop))
}

func onTrayReady(
	ctx context.Context,
	log *slog.Logger,
	uiCfg config.UI,
	waker *waker.Service,
) func() {
	return func() {
		systray.SetIcon(assets.GetIcon())
		systray.SetTitle("Smart PC")
		systray.SetTooltip("Control Agent")

		mOpenDashboard := systray.AddMenuItem("Open dashboard", "Open dashboard")
		mOpenDashboard.SetIcon(assets.GetHouse())
		mOpenInterface := systray.AddMenuItem("Open interface", "Open interface")
		mOpenInterface.SetIcon(assets.GetPcCase())
		mQuit := systray.AddMenuItem("Quit", "Quit")
		mQuit.SetIcon(assets.GetExit())
		mInstallWaker := systray.AddMenuItem("Install waker", "Install waker")
		mInstallWaker.Hide()
		mInstallWaker.SetIcon(assets.GetDownload())

		isWakerAvailable, err := waker.IsAvailable()
		if err != nil {
			log.Error("failed to check if waker is available", sl.Err(err))
		}
		if !isWakerAvailable {
			mInstallWaker.Show()
		}

		go func() {
			const op = "tray"
			log := log.With(sl.Op(op))

			for {
				select {
				case <-ctx.Done():
					systray.Quit()
					return
				case <-mQuit.ClickedCh:
					log.Info("quit clicked")
					systray.Quit()
					return
				case <-mOpenInterface.ClickedCh:
					log.Info("open interface clicked")
					if err := browser.Open(uiCfg.BaseURL + "/this-pc"); err != nil {
						log.Error("failed to open browser", sl.Err(err))
					}
				case <-mOpenDashboard.ClickedCh:
					log.Info("open dashboard clicked")
					if err := browser.Open(uiCfg.BaseURL + "/dashboard"); err != nil {
						log.Error("failed to open browser", sl.Err(err))
					}
				case <-mInstallWaker.ClickedCh:
					log.Info("install waker clicked")

					user, password, err := zenity.Password(
						zenity.Title("Type your username and password"),
						zenity.Username(),
					)
					if errors.Is(err, zenity.ErrCanceled) {
						log.Info("install waker cancelled")
						continue
					}
					if err != nil {
						log.Error("dialog failed", sl.Err(err))
						continue
					}

					wakerIP, err := waker.IP()
					if err != nil {
						log.Error("failed to get waker ip", sl.Err(err))
						continue
					}

					err = installer.Install(
						installer.Credentials{
							Host:     wakerIP.String(),
							Port:     22,
							User:     user,
							Password: password,
						},
						installer.Options{
							Repo:             "MaxRomanov007/smart-pc-waker-agent",
							RemoteBinaryPath: "/usr/bin/smart-pc",
							ServiceName:      "smart-pc",
						},
					)
					if err != nil {
						log.Error("failed to install waker", sl.Err(err))
					}
					log.Info("install waker succeeded")
				}
			}
		}()
	}
}

func onTrayExit(stop context.CancelFunc) func() {
	return func() {
		stop()
	}
}
