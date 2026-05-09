package wakeritems

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"smart-pc-agent/data/assets"
	appi18n "smart-pc-agent/internal/i18n"
	"smart-pc-agent/internal/lib/installer"
	"smart-pc-agent/internal/services/waker"
	"smart-pc-agent/internal/tray/menu"

	"github.com/MaxRomanov007/smart-pc-go-lib/cross-platform/browser"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/getlantern/systray"
	"github.com/ncruces/zenity"
)

const (
	wakerSSHPort      = 22
	wakerRepo         = "MaxRomanov007/smart-pc-waker-agent"
	wakerRemoteBinary = "/usr/bin/smart-pc"
	wakerServiceName  = "smart-pc"

	installStepsCount   = 8
	uninstallStepsCount = 6
)

type Item struct {
	ctx    context.Context
	log    *slog.Logger
	waker  *waker.Service
	events *Events
}

func New(ctx context.Context, log *slog.Logger, w *waker.Service, events *Events) menu.Item {
	return &Item{
		ctx:    ctx,
		log:    log.With(sl.Op("tray.waker")),
		waker:  w,
		events: events,
	}
}

type wakerState struct {
	available    bool
	sshAvailable bool
	authorized   bool
}

func (it *Item) Mount() {
	state, err := it.loadState()
	if err != nil {
		it.log.Error("failed to load waker state", sl.Err(err))
	}

	mInstall := systray.AddMenuItem(appi18n.MsgInstallWaker(), appi18n.MsgInstallWakerTooltip())
	mInstall.SetIcon(assets.GetDownload())

	mUninstall := systray.AddMenuItem(
		appi18n.MsgUninstallWaker(),
		appi18n.MsgUninstallWakerTooltip(),
	)
	mUninstall.SetIcon(assets.GetTrash())

	mAuthorize := systray.AddMenuItem(
		appi18n.MsgAuthorizeWaker(),
		appi18n.MsgAuthorizeWakerTooltip(),
	)
	mAuthorize.SetIcon(assets.GetLogIn())

	it.applyState(state, mInstall, mUninstall, mAuthorize)

	go it.handleClicks(mInstall, mUninstall, mAuthorize)
}

func (it *Item) loadState() (wakerState, error) {
	s := wakerState{}

	available, err := it.waker.IsAvailable()
	if err != nil {
		return s, err
	}
	s.available = available

	sshAvailable, err := it.waker.IsSSHAvailable()
	if err != nil {
		return s, err
	}
	s.sshAvailable = sshAvailable

	if available {
		authorized, err := it.waker.IsAuthorized(it.ctx)
		if err != nil {
			return s, err
		}
		s.authorized = authorized
	}

	return s, nil
}

func (it *Item) applyState(
	s wakerState,
	mInstall, mUninstall, mAuthorize *systray.MenuItem,
) {
	mInstall.Hide()
	mUninstall.Hide()
	mAuthorize.Hide()

	if !s.available {
		mInstall.Show()
		if s.sshAvailable {
			mInstall.SetTitle(appi18n.MsgInstallWaker())
			mInstall.SetTooltip(appi18n.MsgInstallWakerTooltip())
			mInstall.Enable()
		} else {
			mInstall.SetTitle(appi18n.MsgInstallWakerSSHUnavailable())
			mInstall.SetTooltip(appi18n.MsgInstallWakerSSHUnavailableTooltip())
			mInstall.Disable()
		}
		return
	}

	mUninstall.Show()
	if s.sshAvailable {
		mUninstall.SetTitle(appi18n.MsgUninstallWaker())
		mUninstall.SetTooltip(appi18n.MsgUninstallWakerTooltip())
		mUninstall.Enable()
	} else {
		mUninstall.SetTitle(appi18n.MsgUninstallWakerSSHUnavailable())
		mUninstall.SetTooltip(appi18n.MsgUninstallWakerSSHUnavailableTooltip())
		mUninstall.Disable()
	}

	if !s.authorized {
		mAuthorize.Show()
		mAuthorize.Enable()
	}
}

func (it *Item) handleClicks(mInstall, mUninstall, mAuthorize *systray.MenuItem) {
	for {
		select {
		case <-it.ctx.Done():
			return
		case <-it.events.onAuthorized:
			it.log.Info("waker authorized via HTTP callback, hiding authorize button")
			mAuthorize.Hide()
		case <-mInstall.ClickedCh:
			it.onInstall(mInstall, mUninstall, mAuthorize)
		case <-mUninstall.ClickedCh:
			it.onUninstall(mInstall, mUninstall, mAuthorize)
		case <-mAuthorize.ClickedCh:
			it.onAuthorize(mAuthorize)
		}
	}
}

func (it *Item) onInstall(mInstall, mUninstall, mAuthorize *systray.MenuItem) {
	it.log.Info("install waker clicked")

	creds, ok := it.askSSHCredentials(appi18n.MsgSSHCredentialsInstall())
	if !ok {
		return
	}

	dlg, err := zenity.Progress(
		zenity.Title(appi18n.MsgInstallingWaker()),
		zenity.NoCancel(),
	)
	if err != nil {
		it.log.Error("failed to show progress dialog")
		return
	}
	defer func() {
		if err := dlg.Close(); err != nil {
			it.log.Error("failed to close progress dialog", sl.Err(err))
		}
	}()

	if err := installer.InstallWithSteps(
		creds,
		it.installerOptions(),
		it.onInstallStep(dlg),
	); err != nil {
		it.log.Error("failed to install waker", sl.Err(err))
		it.showError(appi18n.MsgFailedToInstallWaker())
		return
	}
	if err := dlg.Text(appi18n.MsgClickOKToAuthorize()); err != nil {
		it.log.Error("failed to set progress dialog text", sl.Err(err))
	}
	if err := dlg.Complete(); err != nil {
		it.log.Error("failed to complete progress dialog", sl.Err(err))
	}

	it.log.Info("waker installed successfully")
	it.refreshMenu(mInstall, mUninstall, mAuthorize)
	<-dlg.Done()

	it.log.Info("opening authorization URL after install")
	it.openAuthorizeURL()
}

func (it *Item) onUninstall(mInstall, mUninstall, mAuthorize *systray.MenuItem) {
	it.log.Info("uninstall waker clicked")

	creds, ok := it.askSSHCredentials(appi18n.MsgSSHCredentialsUninstall())
	if !ok {
		return
	}

	dlg, err := zenity.Progress(zenity.Title(appi18n.MsgUninstallingWaker()), zenity.NoCancel())
	if err != nil {
		it.log.Error("failed to show progress dialog")
		return
	}
	defer func() {
		if err := dlg.Close(); err != nil {
			it.log.Error("failed to close progress dialog", sl.Err(err))
		}
	}()

	if err := installer.UninstallWithSteps(
		creds,
		it.installerOptions(),
		it.unregisterAll,
		it.onUninstallStep(dlg),
	); err != nil {
		it.log.Error("failed to uninstall waker", sl.Err(err))
		it.showError(appi18n.MsgFailedToUninstallWaker())
		return
	}
	if err := dlg.Text(appi18n.MsgWakerUninstalledSuccessfully()); err != nil {
		it.log.Error("failed to set progress dialog text", sl.Err(err))
	}
	if err := dlg.Complete(); err != nil {
		it.log.Error("failed to complete progress dialog", sl.Err(err))
	}

	it.log.Info("waker uninstalled successfully")
	it.refreshMenu(mInstall, mUninstall, mAuthorize)
	<-dlg.Done()
}

func (it *Item) unregisterAll() error {
	const op = "tray.items.wakeritems.unregisterAll"

	isAuthorized, err := it.waker.IsAuthorized(it.ctx)
	if err != nil {
		return fmt.Errorf("%s: failed to get is authorized: %w", op, err)
	}
	if !isAuthorized {
		it.log.Info("not authorized, unregister not needed", sl.Op(op))
		return nil
	}

	return it.waker.UnregisterAll(it.ctx)
}

func (it *Item) onInstallStep(dlg zenity.ProgressDialog) func(step installer.InstallStep) {
	stepFunc := it.dlgStepFunc(dlg, installStepsCount)
	return func(is installer.InstallStep) {
		step := int(is)
		switch is {
		case installer.InstallStepDialSSH:
			stepFunc(step, appi18n.MsgStepDialSSH())
		case installer.InstallStepDetectArch:
			stepFunc(step, appi18n.MsgStepDetectArch())
		case installer.InstallStepGetLatestTag:
			stepFunc(step, appi18n.MsgStepGetLatestTag())
		case installer.InstallStepDownloadBinary:
			stepFunc(step, appi18n.MsgStepDownloadBinary())
		case installer.InstallStepUpload:
			stepFunc(step, appi18n.MsgStepUpload())
		case installer.InstallStepConfigureAutostart:
			stepFunc(step, appi18n.MsgStepConfigureAutostart())
		case installer.InstallStepRemoveTempFile:
			stepFunc(step, appi18n.MsgStepRemoveTempFile())
		case installer.InstallStepCloseConnection:
			stepFunc(step, appi18n.MsgStepCloseConnection())
		}
	}
}

func (it *Item) onUninstallStep(dlg zenity.ProgressDialog) func(step installer.UninstallStep) {
	stepFunc := it.dlgStepFunc(dlg, uninstallStepsCount)
	return func(us installer.UninstallStep) {
		step := int(us)
		switch us {
		case installer.UninstallStepUnregisterAllPcs:
			stepFunc(step, appi18n.MsgStepUnregisterAllPcs())
		case installer.UninstallStepDialSSH:
			stepFunc(step, appi18n.MsgStepDialSSH())
		case installer.UninstallStepRemoveAutostart:
			stepFunc(step, appi18n.MsgStepRemoveAutostart())
		case installer.UninstallStepRemoveBinary:
			stepFunc(step, appi18n.MsgStepRemoveBinary())
		case installer.UninstallStepRemoveCache:
			stepFunc(step, appi18n.MsgStepRemoveCache())
		case installer.UninstallStepCloseConnection:
			stepFunc(step, appi18n.MsgStepCloseConnection())
		}
	}
}

func (it *Item) dlgStepFunc(
	dlg zenity.ProgressDialog,
	stepCount int,
) func(step int, message string) {
	stepValue := 100 / (stepCount - 1)
	return func(step int, message string) {
		log := it.log.With(slog.Int("step", step))
		log.Info(message)

		value := stepValue * step

		if err := dlg.Text(message); err != nil {
			log.Error("failed to set dialog message", sl.Err(err), slog.String("message", message))
		}
		if err := dlg.Value(value); err != nil {
			log.Error("failed to set dialog value", sl.Err(err), slog.Int("value", value))
		}
	}
}

func (it *Item) onAuthorize(mAuthorize *systray.MenuItem) {
	it.log.Info("authorize waker clicked")
	if it.openAuthorizeURL() {
		mAuthorize.Hide()
	}
}

func (it *Item) refreshMenu(mInstall, mUninstall, mAuthorize *systray.MenuItem) {
	state, err := it.loadState()
	if err != nil {
		it.log.Error("failed to refresh waker state", sl.Err(err))
		return
	}
	it.applyState(state, mInstall, mUninstall, mAuthorize)
}

func (it *Item) askSSHCredentials(title string) (installer.Credentials, bool) {
	wakerIP, err := it.waker.IP()
	if err != nil {
		it.log.Error("failed to get waker IP", sl.Err(err))
		return installer.Credentials{}, false
	}

	user, password, err := zenity.Password(
		zenity.Title(title),
		zenity.Username(),
	)
	if errors.Is(err, zenity.ErrCanceled) {
		it.log.Info("SSH credentials dialog cancelled")
		return installer.Credentials{}, false
	}
	if err != nil {
		it.log.Error("SSH credentials dialog failed", sl.Err(err))
		return installer.Credentials{}, false
	}

	return installer.Credentials{
		Host:     wakerIP.String(),
		Port:     wakerSSHPort,
		User:     user,
		Password: password,
	}, true
}

func (it *Item) openAuthorizeURL() bool {
	url, err := it.waker.AuthorizeURL(it.ctx)
	if err != nil {
		it.log.Error("failed to get authorize URL", sl.Err(err))
		return false
	}
	if err := browser.OpenContext(it.ctx, url); err != nil {
		it.log.Error("failed to open authorize URL in browser", sl.Err(err))
		return false
	}
	return true
}

func (it *Item) installerOptions() installer.Options {
	return installer.Options{
		Repo:             wakerRepo,
		RemoteBinaryPath: wakerRemoteBinary,
		ServiceName:      wakerServiceName,
	}
}

func (it *Item) showError(msg string) {
	_ = zenity.Error(msg, zenity.Title("Error"), zenity.ErrorIcon)
}

func (it *Item) showInfo(msg string) {
	_ = zenity.Info(msg, zenity.Title("Success"), zenity.InfoIcon)
}
