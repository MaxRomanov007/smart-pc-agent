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

	goUpdate "github.com/inconshreveable/go-update"
)

// DoUpdateFlag is the CLI flag the agent checks on startup.
// When present the process applies the pre-downloaded binary and exits.
// main() must call HandleDoUpdateFlag() before anything else.
const DoUpdateFlag = "--do-update"

// Apply is the Windows implementation.
//
// Strategy — "silent UAC":
//  1. Download the new binary into %TEMP%\smart-pc-update.exe
//  2. Re-launch ourselves via ShellExecuteW with verb "runas" and the
//     --do-update flag; Windows shows a single UAC prompt.
//  3. The elevated child calls go-update (which now has write access to
//     Program Files) and exits with code 0.
//  4. The original process (us) waits for the child, then returns nil so
//     the caller (updateritems) proceeds to restart normally.
//
// The HKCU Run / startup folder entry is untouched throughout.
func (s *Service) Apply(release ReleaseInfo) error {
	s.log.Info("downloading update",
		slog.String("version", release.Version),
		slog.String("url", release.DownloadURL),
	)

	// 1. Download into a temp file next to the real binary so go-update's
	//    atomic rename stays on the same volume.
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}

	tmpPath := filepath.Join(os.TempDir(), "smart-pc-update.exe")
	if err := downloadToFile(release, tmpPath); err != nil {
		return err
	}
	// Temp file is cleaned up by the elevated child after it applies the update.

	// 2. Re-launch ourselves elevated with --do-update=<tmpPath>.
	s.log.Info("requesting UAC elevation for update")

	arg := DoUpdateFlag + "=" + tmpPath
	exitCode, err := shellExecuteWait("runas", exe, arg)
	if err != nil {
		return fmt.Errorf("shellexecute: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("elevated updater exited with code %d", exitCode)
	}

	s.log.Info("update applied", slog.String("version", release.Version))
	return nil
}

// HandleDoUpdateFlag must be called at the very top of main().
// If --do-update=<path> is present, the process applies the binary at <path>,
// removes the temp file, and exits. This runs elevated (UAC already granted).
func HandleDoUpdateFlag() {
	path := flagValue(DoUpdateFlag)
	if path == "" {
		return
	}

	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "updater: open temp binary: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	if err := goUpdate.Apply(f, goUpdate.Options{}); err != nil {
		fmt.Fprintf(os.Stderr, "updater: apply: %v\n", err)
		os.Exit(1)
	}

	// Clean up temp file — best effort.
	f.Close()
	os.Remove(path)

	os.Exit(0)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func downloadToFile(release ReleaseInfo, dst string) error {
	binary, err := downloadBinary(release)
	if err != nil {
		return err
	}
	defer binary.Close()

	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, binary); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	return nil
}

// flagValue returns the value of a --flag=value argument, or "".
func flagValue(flag string) string {
	prefix := flag + "="
	for _, arg := range os.Args[1:] {
		if len(arg) > len(prefix) && arg[:len(prefix)] == prefix {
			return arg[len(prefix):]
		}
	}
	return ""
}

// shellExecuteWait calls ShellExecuteExW with the given verb, waits for the
// child process to finish, and returns its exit code.
func shellExecuteWait(verb, exe, args string) (uint32, error) {
	shell32 := syscall.NewLazyDLL("shell32.dll")
	shellExecEx := shell32.NewProc("ShellExecuteExW")
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	waitForSingleObject := kernel32.NewProc("WaitForSingleObject")
	getExitCodeProcess := kernel32.NewProc("GetExitCodeProcess")

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

	const (
		seeMaskNoCloseProcess = 0x00000040
		swHide                = 0
	)

	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	argsPtr, _ := syscall.UTF16PtrFromString(args)

	sei := shellExecuteInfo{
		fMask:        seeMaskNoCloseProcess,
		lpVerb:       verbPtr,
		lpFile:       exePtr,
		lpParameters: argsPtr,
		nShow:        swHide,
	}
	sei.cbSize = uint32(unsafe.Sizeof(sei))

	r, _, err := shellExecEx.Call(uintptr(unsafe.Pointer(&sei)))
	if r == 0 {
		return 0, fmt.Errorf("ShellExecuteExW: %w", err)
	}

	// Wait indefinitely for the elevated process to finish.
	waitForSingleObject.Call(sei.hProcess, 0xFFFFFFFF)

	var exitCode uint32
	getExitCodeProcess.Call(sei.hProcess, uintptr(unsafe.Pointer(&exitCode)))
	syscall.CloseHandle(syscall.Handle(sei.hProcess))

	return exitCode, nil
}
