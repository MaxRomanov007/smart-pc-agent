package browser

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
)

// Open opens the specified URL in the default web browser.
// It's a convenience wrapper around OpenContext without context.
func Open(url string) error {
	const op = "lib.cross-platform.browser.open.Open"

	cmd, args, err := getOSRelatedOpenUrlInBrowserCommand(url)
	if err != nil {
		return fmt.Errorf("%s: failed to get command: %w", op, err)
	}

	return exec.Command(cmd, args...).Start()
}

// OpenContext opens the specified URL in the default web browser with context support.
// The context can be used to cancel the command execution.
func OpenContext(ctx context.Context, url string) error {
	const op = "lib.cross-platform.browser.open.OpenContext"

	cmd, args, err := getOSRelatedOpenUrlInBrowserCommand(url)
	if err != nil {
		return fmt.Errorf("%s: failed to get command: %w", op, err)
	}

	return exec.CommandContext(ctx, cmd, args...).Start()
}

// getOSRelatedOpenUrlInBrowserCommand returns the appropriate command and arguments
// to open a URL in the default browser for the current operating system.
// Supports Windows (rundll32), macOS (open), and Linux (xdg-open).
func getOSRelatedOpenUrlInBrowserCommand(url string) (string, []string, error) {
	const op = "lib.cross-platform.browser.open.getOSRelatedOpenUrlInBrowserCommand"

	switch runtime.GOOS {
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}, nil
	case "darwin":
		return "open", []string{url}, nil
	case "linux":
		return "xdg-open", []string{url}, nil
	default:
		return "", nil, fmt.Errorf("%s: unsupported platform", op)
	}
}
