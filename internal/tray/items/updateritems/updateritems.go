// Package updateritems provides the systray menu item for in-app updates.
package updateritems

import (
	"context"
	"log/slog"
	"smart-pc-agent/data/assets"
	appi18n "smart-pc-agent/internal/i18n"
	"smart-pc-agent/internal/lib/restart"
	"smart-pc-agent/internal/services/updater"
	"smart-pc-agent/internal/tray/menu"

	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/getlantern/systray"
	"github.com/ncruces/zenity"
)

// Item implements' menu.Item and wires the updater into the systray.
type Item struct {
	ctx     context.Context
	log     *slog.Logger
	updater *updater.Service
	stop    context.CancelFunc
}

func New(
	ctx context.Context,
	log *slog.Logger,
	upd *updater.Service,
	stop context.CancelFunc,
) menu.Item {
	return &Item{
		ctx:     ctx,
		log:     log.With(sl.Op("tray.updater")),
		updater: upd,
		stop:    stop,
	}
}

func (it *Item) Mount() {
	mUpdate := systray.AddMenuItem(appi18n.MsgCheckForUpdates(), appi18n.MsgCheckForUpdates())
	mUpdate.SetIcon(assets.GetUpdate())

	// Background ticker found an update — switch button to "Update to vX.X.X"
	// and send a desktop notification. The click handler already running below
	// will pick up the release on the next click via the captured variable.
	it.updater.OnUpdateFound(func(release updater.ReleaseInfo) {
		mUpdate.SetTitle(appi18n.MsgUpdateTo(release.Version))
		go notifyAvailable(release.Version)
	})

	go it.handleClicks(mUpdate)
}

// handleClicks is the single, persistent click handler for the lifetime of
// the tray. Each click triggers a fresh Check() so the user always gets the
// current state, regardless of whether the background ticker has fired yet.
func (it *Item) handleClicks(mUpdate *systray.MenuItem) {
	for {
		select {
		case <-it.ctx.Done():
			return
		case <-mUpdate.ClickedCh:
			it.log.Info("update menu item clicked")
			it.checkAndPrompt(mUpdate)
		}
	}
}

func (it *Item) checkAndPrompt(mUpdate *systray.MenuItem) {
	// Disable button while checking to prevent double-clicks.
	mUpdate.SetTitle(appi18n.MsgChecking())
	mUpdate.Disable()

	release, found, err := it.updater.Check()
	if err != nil {
		it.log.Error("manual update check failed", sl.Err(err))
		notifyError(appi18n.MsgCouldNotReachUpdateServer(err.Error()))
		mUpdate.SetTitle(appi18n.MsgCheckForUpdates())
		mUpdate.Enable()
		return
	}

	if !found {
		it.log.Info("already up to date")
		_ = zenity.Info(
			appi18n.MsgAlreadyUpToDate(),
			zenity.Title("Smart PC Agent"),
		)
		mUpdate.SetTitle(appi18n.MsgCheckForUpdates())
		mUpdate.Enable()
		return
	}

	// Update found — ask for confirmation.
	mUpdate.SetTitle(appi18n.MsgUpdateTo(release.Version))
	mUpdate.Enable()

	it.promptAndApply(mUpdate, release)
}

func (it *Item) promptAndApply(mUpdate *systray.MenuItem, release updater.ReleaseInfo) {
	confirmed := askConfirmation(release.Version)
	if !confirmed {
		it.log.Info("user cancelled update")
		return
	}

	it.log.Info("applying update", slog.String("version", release.Version))

	mUpdate.SetTitle(appi18n.MsgUpdating())
	mUpdate.Disable()

	if err := it.updater.Apply(release); err != nil {
		it.log.Error("update failed", sl.Err(err))
		notifyError(err.Error())
		mUpdate.SetTitle(appi18n.MsgUpdateTo(release.Version))
		mUpdate.Enable()
		return
	}

	restart.Now(it.stop)
}

// zenity helpers

func notifyAvailable(version string) {
	_ = zenity.Notify(
		appi18n.MsgUpdateAvailableNotify(version),
		zenity.Title("Smart PC Agent"),
	)
}

func notifyError(msg string) {
	_ = zenity.Error(
		msg,
		zenity.Title(appi18n.MsgUpdateFailed()),
	)
}

func askConfirmation(version string) bool {
	err := zenity.Question(
		appi18n.MsgUpdateAvailableQuestion(version),
		zenity.Title(appi18n.MsgUpdateAvailableTitle()),
		zenity.OKLabel(appi18n.MsgUpdateOKLabel()),
		zenity.CancelLabel(appi18n.MsgCancelLabel()),
	)
	return err == nil
}
