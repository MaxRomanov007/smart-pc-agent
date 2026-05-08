// Package updater provides background update checking and self-update logic
// using GitHub Releases as the distribution channel.
package updater

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
)

const (
	githubRepo    = "MaxRomanov007/smart-pc-agent"
	githubAPI     = "https://api.github.com/repos/" + githubRepo + "/releases/latest"
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

// ── GitHub API ────────────────────────────────────────────────────────────────

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func fetchLatestRelease() (ReleaseInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubAPI, nil)
	if err != nil {
		return ReleaseInfo{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ReleaseInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ReleaseInfo{}, fmt.Errorf("github api returned %d", resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return ReleaseInfo{}, err
	}

	want := assetName(rel.TagName)
	for _, a := range rel.Assets {
		if a.Name == want {
			return ReleaseInfo{
				Version:     rel.TagName,
				DownloadURL: a.BrowserDownloadURL,
			}, nil
		}
	}

	return ReleaseInfo{}, fmt.Errorf("no asset %q in release %s", want, rel.TagName)
}

// assetName returns the asset filename for the current platform.
// On Windows — the Inno Setup installer; on Linux/macOS — the tar.gz archive.
// For Windows the version is not known at call time, so we pass it explicitly.
func assetName(version string) string {
	switch runtime.GOOS {
	case "windows":
		return installerAssetName(version)
	default:
		return fmt.Sprintf("smart-pc-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	}
}

// ── archive helpers (unix only) ───────────────────────────────────────────────

// downloadToReader fetches url and returns a reader over the raw response body.
func downloadToReader(url string) (io.ReadCloser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// extractFromTarGz streams the first regular file from a .tar.gz archive.
func extractFromTarGz(body io.Reader) (io.ReadCloser, error) {
	gz, err := gzip.NewReader(body)
	if err != nil {
		return nil, err
	}

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err != nil {
			return nil, fmt.Errorf("binary not found in archive: %w", err)
		}
		if hdr.Typeflag == tar.TypeReg {
			return &tarEntry{Reader: tr, gz: gz}, nil
		}
	}
}

type tarEntry struct {
	io.Reader
	gz *gzip.Reader
}

func (e *tarEntry) Close() error { return e.gz.Close() }

// installerAssetName returns the Inno Setup installer filename for a given version tag
func installerAssetName(version string) string {
	return fmt.Sprintf("smart-pc-agent-setup-%s.exe", version)
}
