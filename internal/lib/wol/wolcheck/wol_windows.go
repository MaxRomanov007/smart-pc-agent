//go:build windows

package wolcheck

import (
	"encoding/json"
	"os/exec"
)

type psAdapter struct {
	Name              string `json:"Name"`
	WakeOnMagicPacket bool   `json:"WakeOnMagicPacket"`
}

func getWoLInfo() ([]InterfaceWoLInfo, error) {
	out, err := exec.Command(
		"powershell", "-NoProfile", "-Command",
		"Get-NetAdapter | Select-Object Name, WakeOnMagicPacket | ConvertTo-Json",
	).Output()
	if err != nil {
		return nil, err
	}

	// PowerShell возвращает объект вместо массива, если адаптер один
	var adapters []psAdapter
	if err := json.Unmarshal(out, &adapters); err != nil {
		var single psAdapter
		if err2 := json.Unmarshal(out, &single); err2 != nil {
			return nil, err
		}
		adapters = []psAdapter{single}
	}

	var result []InterfaceWoLInfo
	for _, a := range adapters {
		result = append(result, InterfaceWoLInfo{
			Name:    a.Name,
			Enabled: a.WakeOnMagicPacket,
			Mode:    "",
		})
	}

	return result, nil
}
