// Package updater provides background update checking and self-update logic
// using GitHub Releases as the distribution channel.
package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	goUpdate "github.com/inconshreveable/go-update"
)

const (
	githubRepo    = "MaxRomanov007/smart-pc-agent"
	githubAPI     = "https://api.github.com/repos/" + githubRepo + "/releases/latest"
	checkInterval = 3 * time.Hour
)

// ReleaseInfo is the subset of a GitHub release we care about.
type ReleaseInfo struct {
	Version     string // e.g. "v1.2.3"
	DownloadURL string // direct URL to the platform archive
}

// UpdateFoundFunc is called (from a goroutine) when a newer version is found.
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
// Must be called before Start.
func (s *Service) OnUpdateFound(fn UpdateFoundFunc) {
	s.onFound = fn
}

// start launches the background polling loop; returns when ctx is canceled.
func (s *Service) start(ctx context.Context) {
	s.log.Info("updater started",
		slog.String("current", s.currentVersion),
		slog.String("platform", assetName()),
	)

	s.check() // immediate check on start

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
// Safe to call concurrently — used both by the background ticker and the tray item.
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

	// Find the asset that matches the current OS/arch.
	// Workflow produces: smart-pc-linux-amd64.tar.gz, smart-pc-windows-amd64.zip, etc.
	want := assetName()
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

// assetName returns the archive filename for the current platform,
// matching exactly what the Release workflow produces.
func assetName() string {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("smart-pc-%s-%s%s", runtime.GOOS, runtime.GOARCH, ext)
}

// ── Apply ─────────────────────────────────────────────────────────────────────

// Apply downloads the release archive and atomically replaces the running binary.
// The binary path on disk stays the same, so autostart entries are unaffected:
//   - Linux:   ~/.config/systemd/user/smart-pc.service  (ExecStart path unchanged)
//   - Windows: HKCU\...\Run registry value              (path unchanged)
func (s *Service) Apply(release ReleaseInfo) error {
	s.log.Info("downloading update",
		slog.String("version", release.Version),
		slog.String("url", release.DownloadURL),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, release.DownloadURL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download server returned %d", resp.StatusCode)
	}

	binary, err := extractBinary(resp.Body, release.DownloadURL)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}
	defer binary.Close()

	if err := goUpdate.Apply(binary, goUpdate.Options{}); err != nil {
		return fmt.Errorf("apply: %w", err)
	}

	s.log.Info("update applied", slog.String("version", release.Version))
	return nil
}

// ── archive extraction ────────────────────────────────────────────────────────

// extractBinary unwraps the archive and returns a reader over the binary inside.
// Supports .tar.gz (Linux / macOS) and .zip (Windows).
func extractBinary(body io.Reader, url string) (io.ReadCloser, error) {
	if strings.HasSuffix(url, ".tar.gz") {
		return extractFromTarGz(body)
	}
	return extractFromZip(body)
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
			// Return a closer that also closes the gzip reader.
			return &tarEntry{Reader: tr, gz: gz}, nil
		}
	}
}

type tarEntry struct {
	io.Reader
	gz *gzip.Reader
}

func (e *tarEntry) Close() error { return e.gz.Close() }

// extractFromZip buffers the whole body (zip requires random access),
// then returns the first .exe entry (Windows binary).
func extractFromZip(body io.Reader) (io.ReadCloser, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, ".exe") {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			return rc, nil
		}
	}

	return nil, fmt.Errorf("no .exe found in zip archive")
}
