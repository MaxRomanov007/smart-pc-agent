// Package menu provides the registry and base types for system tray menu items.
package menu

import "github.com/getlantern/systray"

// Item is a self-contained menu handler that knows how to add itself
// to the tray and react to user interactions.
type Item interface {
	// Mount adds all systray menu items for this handler and starts any
	// background goroutines it needs. The goroutines must stop when ctx is done.
	Mount()
}

// SeparatorItem is a simple Item that renders a systray separator.
type SeparatorItem struct{}

func NewSeparator() Item { return &SeparatorItem{} }

func (s *SeparatorItem) Mount() { systray.AddSeparator() }
