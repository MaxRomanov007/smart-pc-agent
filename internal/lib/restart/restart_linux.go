//go:build linux

package restart

import (
	"os"
	"syscall"
)

// now stops the application via stop(), then replaces the current process
// image with a fresh copy of the binary using execve.
// The PID stays the same, so systemd never notices the restart —
// the user service unit keeps working as-is.
func now(stop func()) {
	exe, err := os.Executable()
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
