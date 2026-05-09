//go:build !windows

package updater

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
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
