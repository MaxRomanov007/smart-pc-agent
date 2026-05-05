//go:build darwin

package wolcheck

import (
	"os/exec"
	"strings"
)

func GetWoLInfo() ([]InterfaceWoLInfo, error) {
	out, err := exec.Command(
		"system_profiler", "SPNetworkDataType", "-json",
	).Output()
	if err != nil {
		return nil, err
	}

	// system_profiler не даёт прямого поля WoL —
	// на macOS это управляется через pmset
	pmOut, _ := exec.Command("pmset", "-g").Output()
	wolEnabled := strings.Contains(string(pmOut), "womp                 1")

	// Получаем список интерфейсов из system_profiler
	// (упрощённо, без полного парсинга JSON)
	_ = out

	return []InterfaceWoLInfo{
		{
			Name:    "system",
			Enabled: wolEnabled,
			Mode:    "womp (Wake on Magic Packet)",
		},
	}, nil
}
