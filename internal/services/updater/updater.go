// Package updater provides background update checking and self-update logic
// using GitHub Releases as the distribution channel.
package updater

import (
	"context"
	"log/slog"
	"runtime"
	"time"

	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
)

const (
	checkInterval = 3 * time.Hour
)

// ReleaseInfo is the subset of a GitHub release we care about.
type ReleaseInfo struct {
	Version     string // e.g. "v1.2.3"
	DownloadURL string // direct URL to the platform asset
}

// UpdateFoundFunc is called from a goroutine when a newer version is found.
type UpdateFoundFunc func(release ReleaseInfo)

// Service periodically polls GitHub and notifies listeners.
type Service struct {
	log            *slog.Logger
	currentVersion string
	onFound        UpdateFoundFunc
}

func New(ctx context.Context, log *slog.Logger, currentVersion string) *Service {
	s := &Service{
		log:            log.With(sl.Op("updater")),
		currentVersion: currentVersion,
	}
	go s.start(ctx)
	return s
}

// OnUpdateFound registers the callback invoked when a newer version is found.
func (s *Service) OnUpdateFound(fn UpdateFoundFunc) {
	s.onFound = fn
}

func (s *Service) start(ctx context.Context) {
	s.log.Info("updater started",
		slog.String("current", s.currentVersion),
		slog.String("os", runtime.GOOS),
		slog.String("arch", runtime.GOARCH),
	)

	s.check()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.log.Info("updater stopped")
			return
		case <-ticker.C:
			s.check()
		}
	}
}

// Check fetches the latest release and returns it if a newer version exists.
// Returns (release, true, nil) when an update is available,
// (zero, false, nil) when already up to date, or (zero, false, err) on failure.
func (s *Service) Check() (ReleaseInfo, bool, error) {
	release, err := fetchLatestRelease()
	if err != nil {
		s.log.Warn("update check failed", sl.Err(err))
		return ReleaseInfo{}, false, err
	}

	if release.Version == s.currentVersion {
		s.log.Debug("already up to date", slog.String("version", s.currentVersion))
		return ReleaseInfo{}, false, nil
	}

	s.log.Info("update available",
		slog.String("current", s.currentVersion),
		slog.String("latest", release.Version),
	)
	return release, true, nil
}

func (s *Service) check() {
	release, found, _ := s.Check()
	if found && s.onFound != nil {
		s.onFound(release)
	}
}
