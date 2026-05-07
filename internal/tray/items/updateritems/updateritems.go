// Package updateritems provides the systray menu item for in-app updates.
package updateritems

import (
	"context"
	"log/slog"
	"smart-pc-agent/data/assets"
	"smart-pc-agent/internal/lib/restart"
	"smart-pc-agent/internal/services/updater"
	"smart-pc-agent/internal/tray/menu"

	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/getlantern/systray"
	"github.com/ncruces/zenity"
)

// Item implements menu.Item and wires the updater into the systray.
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
	mUpdate := systray.AddMenuItem("Check for updates", "Check for updates")
	mUpdate.SetIcon(assets.GetUpdate())

	// Background ticker found an update — switch button to "Update to vX.X.X"
	// and send a desktop notification. The click handler already running below
	// will pick up the release on the next click via the captured variable.
	it.updater.OnUpdateFound(func(release updater.ReleaseInfo) {
		mUpdate.SetTitle("Update to " + release.Version)
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
	mUpdate.SetTitle("Checking…")
	mUpdate.Disable()

	release, found, err := it.updater.Check()
	if err != nil {
		it.log.Error("manual update check failed", sl.Err(err))
		notifyError("Could not reach update server:\n" + err.Error())
		mUpdate.SetTitle("Check for updates")
		mUpdate.Enable()
		return
	}

	if !found {
		it.log.Info("already up to date")
		_ = zenity.Info(
			"You are running the latest version.",
			zenity.Title("Smart PC Agent"),
		)
		mUpdate.SetTitle("Check for updates")
		mUpdate.Enable()
		return
	}

	// Update found — ask for confirmation.
	mUpdate.SetTitle("Update to " + release.Version)
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

	mUpdate.SetTitle("Updating…")
	mUpdate.Disable()

	if err := it.updater.Apply(release); err != nil {
		it.log.Error("update failed", sl.Err(err))
		notifyError(err.Error())
		mUpdate.SetTitle("Update to " + release.Version)
		mUpdate.Enable()
		return
	}

	restart.Now(it.stop)
}

// ── zenity helpers ────────────────────────────────────────────────────────────

func notifyAvailable(version string) {
	_ = zenity.Notify(
		"Update "+version+" is available. Click the tray icon to install.",
		zenity.Title("Smart PC Agent"),
	)
}

func notifyError(msg string) {
	_ = zenity.Error(
		msg,
		zenity.Title("Update failed"),
	)
}

func askConfirmation(version string) bool {
	err := zenity.Question(
		"Update agent to "+version+"?",
		zenity.Title("Update available"),
		zenity.OKLabel("Update"),
		zenity.CancelLabel("Cancel"),
	)
	return err == nil
}
