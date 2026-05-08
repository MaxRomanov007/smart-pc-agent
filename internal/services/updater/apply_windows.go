//go:build windows

package updater

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// Apply downloads the Inno Setup installer into %TEMP%, launches it silently
// and exits the current process immediately.
//
// Inno Setup handles everything:
//   - UAC elevation request
//   - replacing the binary in Program Files
//   - updating DisplayVersion / DisplayName in the uninstall registry key
//   - restarting the agent via the HKCU Run autostart entry
//
// We exit immediately so the binary is not locked when Inno Setup replaces it.
func (s *Service) Apply(release ReleaseInfo) error {
	s.log.Info("downloading installer",
		slog.String("version", release.Version),
		slog.String("url", release.DownloadURL),
	)

	tmpPath := filepath.Join(os.TempDir(), installerAssetName(release.Version))
	if err := downloadToFile(release.DownloadURL, tmpPath); err != nil {
		return err
	}

	s.log.Info("launching installer", slog.String("path", tmpPath))

	// /SILENT            — progress window only, no wizard pages or clicks
	// /CLOSEAPPLICATIONS — Inno Setup closes the running agent before replacing
	// /RESTARTAPPLICATIONS — Inno Setup restarts it after install
	// /NORESTART         — suppress OS reboot prompt
	if err := shellExecute(
		tmpPath,
		"/SILENT /CLOSEAPPLICATIONS /RESTARTAPPLICATIONS /NORESTART",
	); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("launch installer: %w", err)
	}

	// Exit so the binary is unlocked when Inno Setup tries to replace it.
	// The installer will restart the agent via the autostart registry entry.
	os.Exit(0)
	return nil // unreachable
}

// ── helpers ───────────────────────────────────────────────────────────────────

func downloadToFile(url, dst string) error {
	body, err := downloadToReader(url)
	if err != nil {
		return err
	}
	defer body.Close()

	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, body); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	return nil
}

// shellExecute launches file with params without waiting for it to finish.
// Inno Setup will request UAC elevation itself when it runs.
func shellExecute(file, params string) error {
	shell32 := syscall.NewLazyDLL("shell32.dll")
	shellExecEx := shell32.NewProc("ShellExecuteExW")

	type shellExecuteInfo struct {
		cbSize         uint32
		fMask          uint32
		hwnd           uintptr
		lpVerb         *uint16
		lpFile         *uint16
		lpParameters   *uint16
		lpDirectory    *uint16
		nShow          int32
		hInstApp       uintptr
		lpIDList       uintptr
		lpClass        *uint16
		hkeyClass      uintptr
		dwHotKey       uint32
		hIconOrMonitor uintptr
		hProcess       uintptr
	}

	const swNormal = 1

	filePtr, _ := syscall.UTF16PtrFromString(file)
	paramsPtr, _ := syscall.UTF16PtrFromString(params)

	sei := shellExecuteInfo{
		lpFile:       filePtr,
		lpParameters: paramsPtr,
		nShow:        swNormal,
	}
	sei.cbSize = uint32(unsafe.Sizeof(sei))

	r, _, err := shellExecEx.Call(uintptr(unsafe.Pointer(&sei)))
	if r == 0 {
		return fmt.Errorf("ShellExecuteExW: %w", err)
	}
	return nil
}
