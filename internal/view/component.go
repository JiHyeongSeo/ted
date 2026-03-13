package view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/seoji/ted/internal/types"
)

// Component is the interface all view components must implement.
type Component interface {
	// Render draws the component to the screen within its bounds.
	Render(screen tcell.Screen)
	// SetBounds sets the rectangular region this component occupies.
	SetBounds(bounds types.Rect)
	// Bounds returns the current bounds.
	Bounds() types.Rect
	// HandleEvent processes a tcell event. Returns true if handled.
	HandleEvent(ev tcell.Event) bool
	// SetFocused sets whether this component has input focus.
	SetFocused(focused bool)
	// IsFocused returns whether this component has input focus.
	IsFocused() bool
}

// BaseComponent provides a default implementation of Component.
type BaseComponent struct {
	bounds  types.Rect
	focused bool
}

func (b *BaseComponent) SetBounds(bounds types.Rect) { b.bounds = bounds }
func (b *BaseComponent) Bounds() types.Rect           { return b.bounds }
func (b *BaseComponent) SetFocused(focused bool)      { b.focused = focused }
func (b *BaseComponent) IsFocused() bool               { return b.focused }
func (b *BaseComponent) HandleEvent(ev tcell.Event) bool { return false }
func (b *BaseComponent) Render(screen tcell.Screen) {}
