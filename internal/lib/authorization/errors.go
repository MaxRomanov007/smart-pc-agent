package authorization

import "errors"

// Common authorization errors
var (
	// ErrNoToken indicates that no token is available for authentication
	ErrNoToken = errors.New("no token")
	// ErrTokenLocked indicates that the token is currently locked by another goroutine
	// and TryToken() cannot acquire the lock
	ErrTokenLocked = errors.New("token is locked")
)
