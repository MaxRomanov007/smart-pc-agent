// Package mediactl provides cross-platform media control (play/pause, next, previous).
// It sends system media key events using native OS APIs.
package mediactl

// Action represents a media control action.
type Action int

const (
	Play Action = iota // Play / Pause toggle
	Next               // Next track
	Prev               // Previous track
)

// PlayPause sends a Play/Pause media key event.
func PlayPause() error {
	return sendKey(Play)
}

// NextTrack sends a Next Track media key event.
func NextTrack() error {
	return sendKey(Next)
}

// PrevTrack sends a Previous Track media key event.
func PrevTrack() error {
	return sendKey(Prev)
}
