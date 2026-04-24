package mqtt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	mqttMessage "smart-pc-agent/internal/domain/models/mqtt-message"
	"time"

	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	mqttAuth "github.com/MaxRomanov007/smart-pc-go-lib/mqtt-auth"
	"github.com/eclipse/paho.golang/paho"
	"github.com/itchyny/volume-go"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

func startSendState(ctx context.Context, pcID string, log *slog.Logger, conn *mqttAuth.Connection) {
	const op = "mqtt.sendState"

	go func() {
		log := log.With(sl.Op(op))

		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		ticker := time.NewTicker(1 * time.Second)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				state, err := getState()
				if err != nil {
					log.Warn("error occurred while getting state", sl.Err(err))
				}

				message := mqttMessage.Message[*State]{
					Type: "pc-state",
					Data: state,
				}
				jsonMessage, err := json.Marshal(message)
				if err != nil {
					log.Warn("error occurred while marshaling mqtt message", sl.Err(err))
				}

				if _, err := conn.Publish(ctx, &paho.Publish{
					QoS:     1,
					Retain:  true,
					Topic:   fmt.Sprintf("pcs/%s/state", pcID),
					Payload: jsonMessage,
				}); err != nil {
					log.Warn("error occurred while sending state", sl.Err(err))
					continue
				}

				if _, err := conn.Publish(ctx, &paho.Publish{
					QoS:     1,
					Retain:  true,
					Topic:   fmt.Sprintf("pcs/%s/status", pcID),
					Payload: []byte("{\"type\":\"pc-status\",\"data\":{\"status\":\"online\"}}"),
				}); err != nil {
					log.Warn("error occurred while sending status", sl.Err(err))
					continue
				}
			}
		}
	}()
}

type State struct {
	CPUPercent    float64            `json:"cpuPercent"`
	VirtualMemory VirtualMemoryState `json:"virtualMemory"`
	Volume        *VolumeState       `json:"volume,omitempty"`
}

type VirtualMemoryState struct {
	Total     uint64 `json:"total"`
	Available uint64 `json:"available"`
}

type VolumeState struct {
	Current *int  `json:"current,omitempty"`
	Muted   *bool `json:"muted,omitempty"`
}

func getState() (*State, error) {
	const op = "getState"

	errs := make([]error, 0, 4)
	percent, err := cpu.Percent(100*time.Millisecond, false)
	if err != nil {
		errs = append(errs, fmt.Errorf("%s: error getting used cpu percent: %w", op, err))
	}
	vm, err := mem.VirtualMemory()
	if err != nil {
		errs = append(errs, fmt.Errorf("%s: error getting memory stats: %w", op, err))
	}

	currentVolume := new(int)
	*currentVolume, err = volume.GetVolume()
	if err != nil {
		currentVolume = nil
		errs = append(errs, fmt.Errorf("%s: error getting current volume: %w", op, err))
	}
	muted := new(bool)
	*muted, err = volume.GetMuted()
	if err != nil {
		muted = nil
		errs = append(errs, fmt.Errorf("%s: error getting muted status: %w", op, err))
	}

	var volumeState *VolumeState
	if currentVolume != nil || muted != nil {
		volumeState = &VolumeState{
			Current: currentVolume,
			Muted:   muted,
		}
	}

	return &State{
		CPUPercent: percent[0],
		VirtualMemory: VirtualMemoryState{
			Total:     vm.Total,
			Available: vm.Available,
		},
		Volume: volumeState,
	}, errors.Join(errs...)
}
