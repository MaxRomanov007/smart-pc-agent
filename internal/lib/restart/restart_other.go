//go:build !linux && !darwin && !windows

package restart

import (
	"os"
	"os/exec"
)

// now stops the application via stop(), spawns a new copy of the binary
// and exits. Used as a fallback on platforms other than Linux, macOS, Windows.
func now(stop func()) {
	exe, err := os.Executable()
	if err != nil {
		stop()
		os.Exit(0)
	}

	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	stop()

	if err := cmd.Start(); err != nil {
		os.Exit(0)
	}

	os.Exit(0)
}
