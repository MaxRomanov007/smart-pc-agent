//go:build linux

package mediactl

import (
	"fmt"
	"os/exec"
)

// On Linux we prefer playerctl (D-Bus / MPRIS2).
// If playerctl is not installed we fall back to xdotool (X11 media keys).
// Both are widely available on modern desktop distros.

func sendKey(action Action) error {
	if err := sendViaPlayerctl(action); err == nil {
		return nil
	}
	return sendViaXdotool(action)
}

// sendViaPlayerctl uses the playerctl CLI which speaks MPRIS2 over D-Bus.
// Works with Spotify, VLC, Rhythmbox, Chromium, Firefox, etc.
func sendViaPlayerctl(action Action) error {
	var cmd string
	switch action {
	case Play:
		cmd = "play-pause"
	case Next:
		cmd = "next"
	case Prev:
		cmd = "previous"
	default:
		return fmt.Errorf("mediactl: unknown action %d", action)
	}

	path, err := exec.LookPath("playerctl")
	if err != nil {
		return fmt.Errorf("mediactl: playerctl not found: %w", err)
	}

	if out, err := exec.Command(path, cmd).CombinedOutput(); err != nil {
		return fmt.Errorf("mediactl: playerctl %s: %w — %s", cmd, err, out)
	}
	return nil
}

// sendViaXdotool simulates XF86 media key presses (X11 only).
// Useful when playerctl is unavailable or no MPRIS2 player is running.
func sendViaXdotool(action Action) error {
	var keyName string
	switch action {
	case Play:
		keyName = "XF86AudioPlay"
	case Next:
		keyName = "XF86AudioNext"
	case Prev:
		keyName = "XF86AudioPrev"
	default:
		return fmt.Errorf("mediactl: unknown action %d", action)
	}

	path, err := exec.LookPath("xdotool")
	if err != nil {
		return fmt.Errorf("mediactl: neither playerctl nor xdotool found; install one of them")
	}

	if out, err := exec.Command(path, "key", keyName).CombinedOutput(); err != nil {
		return fmt.Errorf("mediactl: xdotool key %s: %w — %s", keyName, err, out)
	}
	return nil
}
