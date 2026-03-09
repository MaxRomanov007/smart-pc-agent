//go:build windows

package mediactl

import (
	"fmt"
	"syscall"
)

// Virtual key codes for media keys on Windows
// https://docs.microsoft.com/en-us/windows/win32/inputdev/virtual-key-codes
const (
	vkMediaPlayPause = 0xB3
	vkMediaNextTrack = 0xB0
	vkMediaPrevTrack = 0xB1

	keyeventfKeyup = uintptr(0x0002)
)

var (
	user32     = syscall.NewLazyDLL("user32.dll")
	keybdEvent = user32.NewProc("keybd_event")
)

func sendKey(action Action) error {
	var vk uintptr

	switch action {
	case Play:
		vk = vkMediaPlayPause
	case Next:
		vk = vkMediaNextTrack
	case Prev:
		vk = vkMediaPrevTrack
	default:
		return fmt.Errorf("mediactl: unknown action %d", action)
	}

	// keybd_event is a void function — it has no return value.
	// Go's syscall always populates the error return with GetLastError(),
	// which may contain a stale error code from a previous unrelated call.
	// The correct way to check success is: there is no way — just call it.
	keybdEvent.Call(vk, 0, 0, 0)              // key down
	keybdEvent.Call(vk, 0, keyeventfKeyup, 0) // key up

	return nil
}
