package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"
)

const (
	githubRepo = "MaxRomanov007/smart-pc-agent"
	githubAPI  = "https://api.github.com/repos/" + githubRepo + "/releases/latest"
)

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

// installerAssetName returns the Inno Setup installer filename for a given version tag
func installerAssetName(version string) string {
	return fmt.Sprintf("smart-pc-agent-setup-%s.exe", version)
}
