//go:build !windows

package updater

import (
	"fmt"
	"log/slog"
	"os"

	goUpdate "github.com/inconshreveable/go-update"
)

// Apply downloads the .tar.gz archive, extracts the binary and atomically
// replaces the running executable via go-update.
// The path on disk stays the same, so systemd/launchd autostart is unaffected.
func (s *Service) Apply(release ReleaseInfo) error {
	s.log.Info("downloading update",
		slog.String("version", release.Version),
		slog.String("url", release.DownloadURL),
	)

	body, err := downloadToReader(release.DownloadURL)
	if err != nil {
		return err
	}
	defer body.Close()

	binary, err := extractFromTarGz(body)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}
	defer binary.Close()

	if err := goUpdate.Apply(binary, goUpdate.Options{}); err != nil {
		return fmt.Errorf("apply: %w", err)
	}

	// go-update leaves a .old backup — remove it best-effort.
	if exe, err := os.Executable(); err == nil {
		os.Remove(exe + ".old")
	}

	s.log.Info("update applied", slog.String("version", release.Version))
	return nil
}
