//go:build linux

package wolcheck

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

func getWoLInfo() ([]InterfaceWoLInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var result []InterfaceWoLInfo

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		out, err := exec.Command("ethtool", iface.Name).Output()
		if err != nil {
			continue
		}

		info := InterfaceWoLInfo{Name: iface.Name}

		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "Wake-on:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					info.Mode = parts[1]
					info.Enabled = parts[1] != "d" && parts[1] != ""
				}
			}
		}

		result = append(result, info)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no interfaces found or ethtool not available")
	}

	return result, nil
}
