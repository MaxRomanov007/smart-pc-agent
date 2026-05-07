//go:build !windows && !darwin && !linux

package mediactl

import "fmt"

func sendKey(action Action) error {
	return fmt.Errorf("mediactl: unsupported platform")
}
