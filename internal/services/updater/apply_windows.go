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
	"golang.org/x/sys/windows/registry"
)

// DoUpdateFlag is the CLI flag the agent checks on startup.
const DoUpdateFlag = "--do-update"

// Inno Setup writes the app version under this registry key.
// The elevated process updates it after applying the binary.
const innoSetupRegKey = `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Smart PC Agent_is1`

// Apply is the Windows implementation.
//
// Strategy — "silent UAC":
//  1. Download the new binary into %TEMP%\smart-pc-update.exe
//  2. Re-launch ourselves via ShellExecuteExW with verb "runas" and the
//     --do-update=<path>,<version> flag; Windows shows a single UAC prompt.
//  3. The elevated child applies go-update, updates the registry version,
//     schedules deletion of the .old file via MoveFileExW (DELAY_UNTIL_REBOOT)
//     as a fallback, deletes the temp binary, then exits 0.
//  4. The original process waits, gets exit code 0, returns nil — caller
//     proceeds to restart.Now() which spawns a fresh non-elevated agent.
//     The fresh agent calls CleanupOldBinary() on startup to remove .old if
//     it was not already cleaned up.
func (s *Service) Apply(release ReleaseInfo) error {
	s.log.Info("downloading update",
		slog.String("version", release.Version),
		slog.String("url", release.DownloadURL),
	)

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}

	tmpPath := filepath.Join(os.TempDir(), "smart-pc-update.exe")
	if err := downloadToFile(release, tmpPath); err != nil {
		return err
	}

	s.log.Info("requesting UAC elevation for update")

	// Pass both the temp path and the new version string to the elevated child.
	arg := DoUpdateFlag + "=" + tmpPath + "," + release.Version
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
// If --do-update=<tmpPath>,<version> is present the process:
//  1. Applies the new binary via go-update  (has admin rights via UAC)
//  2. Tries to remove the .old leftover immediately; if Windows still holds a
//     lock on it (the current process IS that old binary), schedules deletion
//     on next reboot via MoveFileExW(MOVEFILE_DELAY_UNTIL_REBOOT).
//  3. Updates the DisplayVersion in the Inno Setup uninstall registry key
//  4. Removes the temp binary
//  5. Exits 0
//
// After this process exits, the fresh (non-elevated) agent that restart.Now()
// spawns will call CleanupOldBinary() on startup, which removes the .old file
// before Windows places a lock on the new process image — covering the common
// case where the reboot-scheduled deletion is not needed.
func HandleDoUpdateFlag() {
	val := flagValue(DoUpdateFlag)
	if val == "" {
		return
	}

	// Parse "tmpPath,version"
	sep := lastIndex(val, ',')
	if sep < 0 {
		fmt.Fprintln(os.Stderr, "updater: malformed --do-update value")
		os.Exit(1)
	}
	tmpPath := val[:sep]
	newVersion := val[sep+1:]

	// 1. Apply the new binary.
	f, err := os.Open(tmpPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "updater: open temp binary: %v\n", err)
		os.Exit(1)
	}

	if err := goUpdate.Apply(f, goUpdate.Options{}); err != nil {
		f.Close()
		fmt.Fprintf(os.Stderr, "updater: apply: %v\n", err)
		os.Exit(1)
	}
	f.Close()

	// 2. Remove the .old file go-update leaves behind.
	//
	// Root cause: go-update renames the current exe to <exe>.old before
	// writing the new binary. The elevated child process IS that old binary,
	// so Windows keeps a file lock on <exe>.old for as long as the process
	// runs. os.Remove therefore fails here.
	//
	// Strategy:
	//   a) Try immediate removal — succeeds once the process has exited, but
	//      we try anyway in case go-update's behaviour changes.
	//   b) Schedule deletion at next OS reboot via MoveFileExW as a safety net.
	//   c) CleanupOldBinary() (called by the freshly-restarted agent on startup)
	//      handles the normal case: by the time the new process starts, the
	//      elevated child has exited and the lock is gone.
	exe, err := os.Executable()
	if err == nil {
		oldPath := exe + ".old"
		if removeErr := os.Remove(oldPath); removeErr != nil {
			// Lock still held — schedule deletion on next Windows boot.
			scheduleDeleteOnReboot(oldPath) // best effort, requires admin (we have it here)
		}
	}

	// 3. Update DisplayVersion in the Inno Setup uninstall registry key.
	// Strip leading "v" — Inno Setup stores "0.0.5" not "v0.0.5".
	displayVersion := newVersion
	if len(displayVersion) > 0 && displayVersion[0] == 'v' {
		displayVersion = displayVersion[1:]
	}
	updateRegistryVersion(displayVersion) // best effort — don't fail the update

	// 4. Remove temp binary.
	os.Remove(tmpPath)

	os.Exit(0)
}

// CleanupOldBinary removes the <exe>.old file left by go-update, if present.
// Call this once near the top of main(), before the application starts doing
// meaningful work. By the time the freshly-updated process reaches main(),
// the elevated child that held the lock on .old has already exited, so the
// removal succeeds without UAC.
func CleanupOldBinary() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	os.Remove(exe + ".old") // best effort — ignore errors
}

func updateRegistryVersion(version string) {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		innoSetupRegKey,
		registry.SET_VALUE,
	)
	if err != nil {
		return
	}
	defer k.Close()

	k.SetStringValue("DisplayVersion", version)                //nolint:errcheck
	k.SetStringValue("DisplayName", "Smart PC Agent "+version) //nolint:errcheck
}

// scheduleDeleteOnReboot marks path for deletion on the next Windows reboot
// using MoveFileExW(path, NULL, MOVEFILE_DELAY_UNTIL_REBOOT).
// Requires administrator privileges — safe to call from the UAC-elevated child.
func scheduleDeleteOnReboot(path string) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	moveFileEx := kernel32.NewProc("MoveFileExW")

	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return
	}

	const movefileDelayUntilReboot = 0x00000004

	// lpNewFileName = NULL means "delete on reboot".
	moveFileEx.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0, // NULL
		movefileDelayUntilReboot,
	)
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

// flagValue returns the value of --flag=value, or "".
func flagValue(flag string) string {
	prefix := flag + "="
	for _, arg := range os.Args[1:] {
		if len(arg) > len(prefix) && arg[:len(prefix)] == prefix {
			return arg[len(prefix):]
		}
	}
	return ""
}

// lastIndex returns the index of the last occurrence of sep in s, or -1.
func lastIndex(s string, sep byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == sep {
			return i
		}
	}
	return -1
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

	waitForSingleObject.Call(sei.hProcess, 0xFFFFFFFF)

	var exitCode uint32
	getExitCodeProcess.Call(sei.hProcess, uintptr(unsafe.Pointer(&exitCode)))
	syscall.CloseHandle(syscall.Handle(sei.hProcess))

	return exitCode, nil
}
