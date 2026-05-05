//go:build !windows && !darwin && !linux

package wolcheck

import "fmt"

func getWoLInfo() ([]InterfaceWoLInfo, error) {
	return nil, fmt.Errorf("wolcheck: unsupported platform")
}
