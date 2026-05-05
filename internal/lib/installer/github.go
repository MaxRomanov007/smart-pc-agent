package installer

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// latestTag returns the tag name of the latest GitHub release, e.g. "v1.3.0".
func latestTag(repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %s for %s", resp.Status, url)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decode release JSON: %w", err)
	}
	if release.TagName == "" {
		return "", fmt.Errorf("empty tag_name in GitHub response")
	}
	return release.TagName, nil
}

// downloadBinary fetches the .tar.gz asset for the given arch from GitHub
// Releases, extracts the binary into a temp file and returns its path.
// The caller is responsible for removing the temp file.
func downloadBinary(repo, tag, arch string) (string, error) {
	assetName := arch + ".tar.gz"
	url := fmt.Sprintf(
		"https://github.com/%s/releases/download/%s/%s",
		repo, tag, assetName,
	)

	resp, err := http.Get(url) //nolint:gosec,noctx
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s returned %s", url, resp.Status)
	}

	tmp, err := os.CreateTemp("", "router-agent-*")
	if err != nil {
		return "", err
	}

	if err := extractBinary(resp.Body, arch, tmp); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}

	tmp.Close()
	return tmp.Name(), nil
}

// extractBinary reads a .tar.gz stream and writes the entry named binaryName
// into dst.
func extractBinary(r io.Reader, binaryName string, dst *os.File) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}
		if filepath.Base(hdr.Name) == binaryName {
			if _, err := io.Copy(dst, tr); err != nil {
				return fmt.Errorf("extract: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("binary %q not found inside archive", binaryName)
}

func removeTempFile(path string) {
	_ = os.Remove(path)
}
