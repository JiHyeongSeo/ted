package view

import (
	"fmt"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/seoji/ted/internal/syntax"
	"github.com/seoji/ted/internal/types"
)

// StatusBar displays file information at the bottom of the editor.
type StatusBar struct {
	BaseComponent
	theme    *syntax.Theme
	filename string
	language string
	line     int
	col      int
	dirty    bool
	encoding string
	message  string // temporary message to display instead of normal info
}

// NewStatusBar creates a new StatusBar.
func NewStatusBar(theme *syntax.Theme) *StatusBar {
	return &StatusBar{
		theme:    theme,
		encoding: "UTF-8",
	}
}

// Update refreshes the status bar with current editor state.
func (sb *StatusBar) Update(filePath string, language string, line, col int, dirty bool) {
	if filePath != "" {
		sb.filename = filepath.Base(filePath)
	} else {
		sb.filename = "[No Name]"
	}
	sb.language = language
	sb.line = line
	sb.col = col
	sb.dirty = dirty
}

// Render draws the status bar.
func (sb *StatusBar) Render(screen tcell.Screen) {
	bounds := sb.Bounds()
	style := sb.theme.UIStyle("statusbar")

	// Clear the status bar area
	for x := bounds.X; x < bounds.X+bounds.Width; x++ {
		screen.SetContent(x, bounds.Y, ' ', nil, style)
	}

	// If there's a message, show it instead
	if sb.message != "" {
		x := bounds.X
		msgStyle := style.Foreground(tcell.ColorYellow)
		for _, ch := range " " + sb.message {
			if x >= bounds.X+bounds.Width {
				break
			}
			screen.SetContent(x, bounds.Y, ch, nil, msgStyle)
			x++
		}
		return
	}

	// Left side: filename + modified indicator
	left := " " + sb.filename
	if sb.dirty {
		left += " [+]"
	}

	// Right side: language, line:col, encoding
	right := fmt.Sprintf("%s  Ln %d, Col %d  %s ", sb.language, sb.line+1, sb.col+1, sb.encoding)

	// Draw left
	x := bounds.X
	for _, ch := range left {
		if x >= bounds.X+bounds.Width {
			break
		}
		screen.SetContent(x, bounds.Y, ch, nil, style)
		x++
	}

	// Draw right (right-aligned)
	rightStart := bounds.X + bounds.Width - len(right)
	if rightStart > x {
		x = rightStart
		for _, ch := range right {
			if x >= bounds.X+bounds.Width {
				break
			}
			screen.SetContent(x, bounds.Y, ch, nil, style)
			x++
		}
	}
}

// SetPosition is a convenience method to update cursor position.
func (sb *StatusBar) SetPosition(pos types.Position) {
	sb.line = pos.Line
	sb.col = pos.Col
}

// SetMessage sets a temporary message to display in the status bar.
func (sb *StatusBar) SetMessage(msg string) {
	sb.message = msg
}

// ClearMessage clears the temporary message.
func (sb *StatusBar) ClearMessage() {
	sb.message = ""
}
