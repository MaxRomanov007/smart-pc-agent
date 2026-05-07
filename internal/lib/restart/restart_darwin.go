//go:build darwin

package restart

import (
	"os"
	"path/filepath"
	"syscall"
)

// now stops the application via stop(), then replaces the current process
// image with a fresh copy of the binary using execve.
// filepath.EvalSymlinks is used because os.Executable() may return a symlink
// on macOS (e.g. when the binary is installed under /usr/local/bin).
func now(stop func()) {
	exe, err := os.Executable()
	if err != nil {
		stop()
		os.Exit(0)
	}

	// Resolve symlinks before exec — required on macOS.
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		stop()
		os.Exit(0)
	}

	stop()

	// execve — never returns on success.
	if err := syscall.Exec(exe, os.Args, os.Environ()); err != nil {
		os.Exit(0)
	}
}
