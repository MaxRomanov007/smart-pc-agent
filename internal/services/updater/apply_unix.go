//go:build !windows

package updater

import (
	"fmt"
	"log/slog"

	goUpdate "github.com/inconshreveable/go-update"
)

// Apply downloads the release archive and atomically replaces the running
// binary via go-update. The path on disk stays the same, so autostart
// entries (systemd user service on Linux, launchd plist on macOS) are
// unaffected.
func (s *Service) Apply(release ReleaseInfo) error {
	s.log.Info("downloading update",
		slog.String("version", release.Version),
		slog.String("url", release.DownloadURL),
	)

	binary, err := downloadBinary(release)
	if err != nil {
		return err
	}
	defer binary.Close()

	if err := goUpdate.Apply(binary, goUpdate.Options{}); err != nil {
		return fmt.Errorf("apply: %w", err)
	}

	s.log.Info("update applied", slog.String("version", release.Version))
	return nil
}

// HandleDoUpdateFlag is a no-op on non-Windows platforms.
// On Windows it is defined in apply_windows.go and handles the --do-update flag.
func HandleDoUpdateFlag() {}
