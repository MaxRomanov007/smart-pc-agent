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

		mAuthorizeWaker := new(systray.MenuItem)
		mUninstallWaker := new(systray.MenuItem)
		mInstallWaker := new(systray.MenuItem)
		isWakerAvailable, err := waker.IsAvailable()
		if err != nil {
			log.Error("failed to check if waker is available", sl.Err(err))
		}
		isSSHAvailable, err := waker.IsSSHAvailable()
		if err != nil {
			log.Error("failed to check if waker is SSH available", sl.Err(err))
		}
		if isWakerAvailable {
			isAuthorized, err := waker.IsAuthorized(ctx)
			if err != nil {
				log.Error("failed to check if waker is authorized", sl.Err(err))
			}

			if !isAuthorized {
				mAuthorizeWaker = systray.AddMenuItem(
					"Authorize Waker",
					"Authorize waker so it can receive commands",
				)
				mAuthorizeWaker.SetIcon(assets.GetLogIn())
			}

			if isSSHAvailable {
				mUninstallWaker = systray.AddMenuItem("Uninstall waker", "Uninstall waker")
				mUninstallWaker.SetIcon(assets.GetTrash())
			} else {
				mUninstallWaker = systray.AddMenuItem(
					"Uninstall waker",
					"Can not uninstall waker because the device at waker IP does not support SSH",
				)
				mUninstallWaker.Disable()
				mUninstallWaker.SetIcon(assets.GetDownload())
			}
		} else {
			if isSSHAvailable {
				mInstallWaker = systray.AddMenuItem("Install waker", "Install waker")
				mInstallWaker.SetIcon(assets.GetDownload())
			} else {
				mInstallWaker = systray.AddMenuItem(
					"Install waker",
					"Can not install waker because the device at waker IP does not support SSH",
				)
				mInstallWaker.Disable()
				mInstallWaker.SetIcon(assets.GetDownload())
			}
		}

		mQuit := systray.AddMenuItem("Quit", "Quit")
		mQuit.SetIcon(assets.GetExit())

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
						zenity.Title("Type SSH username and password"),
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
						_ = zenity.Error("Failed to install waker, see logs for more details",
							zenity.Title("Error"),
							zenity.ErrorIcon,
						)
						log.Error("failed to install waker", sl.Err(err))
						continue
					}

					log.Info("install waker succeeded")
					_ = zenity.Info("Waker installed successfully",
						zenity.Title("Success"),
						zenity.InfoIcon)

					log.Info("authorizing waker")

					url, err := waker.AuthorizeURL(ctx)
					if err != nil {
						log.Error("failed to get authorize url", sl.Err(err))
						continue
					}

					if err := browser.OpenContext(ctx, url); err != nil {
						log.Error("failed to open url in browser", sl.Err(err))
						continue
					}
				case <-mUninstallWaker.ClickedCh:
					log.Info("uninstall waker clicked")

					user, password, err := zenity.Password(
						zenity.Title("Type SSH username and password"),
						zenity.Username(),
					)
					if errors.Is(err, zenity.ErrCanceled) {
						log.Info("uninstall waker cancelled")
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

					err = installer.Uninstall(
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
						_ = zenity.Error("Failed to uninstall waker, see logs for more details",
							zenity.Title("Error"),
							zenity.ErrorIcon,
						)
						log.Error("failed to uninstall waker", sl.Err(err))
						continue
					}

					log.Info("uninstall waker succeeded")
					_ = zenity.Info("Waker uninstalled successfully",
						zenity.Title("Success"),
						zenity.InfoIcon)
				case <-mAuthorizeWaker.ClickedCh:
					log.Info("authorize waker clicked")

					url, err := waker.AuthorizeURL(ctx)
					if err != nil {
						log.Error("failed to get authorize url", sl.Err(err))
						continue
					}

					if err := browser.OpenContext(ctx, url); err != nil {
						log.Error("failed to open url in browser", sl.Err(err))
						continue
					}
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
