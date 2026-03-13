package view

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/seoji/ted/internal/syntax"
)

// Tooltip renders a multi-line text popup near the cursor.
type Tooltip struct {
	BaseComponent
	theme   *syntax.Theme
	lines   []string
	visible bool
	anchorX int
	anchorY int
}

// NewTooltip creates a new Tooltip.
func NewTooltip(theme *syntax.Theme) *Tooltip {
	return &Tooltip{theme: theme}
}

// Show displays the tooltip at the given screen position.
func (t *Tooltip) Show(text string, anchorX, anchorY int) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	t.lines = strings.Split(text, "\n")
	// Limit to 10 lines
	if len(t.lines) > 10 {
		t.lines = t.lines[:10]
	}
	t.anchorX = anchorX
	t.anchorY = anchorY
	t.visible = true
}

// Hide hides the tooltip.
func (t *Tooltip) Hide() {
	t.visible = false
	t.lines = nil
}

// IsVisible returns whether the tooltip is shown.
func (t *Tooltip) IsVisible() bool {
	return t.visible
}

// Render draws the tooltip.
func (t *Tooltip) Render(screen tcell.Screen) {
	if !t.visible || len(t.lines) == 0 {
		return
	}

	screenW, screenH := screen.Size()

	// Calculate dimensions
	maxWidth := 0
	for _, line := range t.lines {
		w := runewidth.StringWidth(line)
		if w > maxWidth {
			maxWidth = w
		}
	}
	popupWidth := maxWidth + 4
	if popupWidth > 80 {
		popupWidth = 80
	}
	popupHeight := len(t.lines) + 2 // +2 for top/bottom padding

	// Position: above cursor if possible
	popupX := t.anchorX
	popupY := t.anchorY - popupHeight
	if popupY < 0 {
		popupY = t.anchorY + 1
	}
	if popupX+popupWidth > screenW {
		popupX = screenW - popupWidth
	}
	if popupX < 0 {
		popupX = 0
	}
	if popupY+popupHeight > screenH {
		popupHeight = screenH - popupY
	}

	style := tcell.StyleDefault.
		Background(tcell.NewRGBColor(40, 40, 40)).
		Foreground(tcell.NewRGBColor(212, 212, 212))
	borderStyle := style.Foreground(tcell.NewRGBColor(80, 80, 80))

	// Draw top border
	screen.SetContent(popupX, popupY, '+', nil, borderStyle)
	for x := popupX + 1; x < popupX+popupWidth-1; x++ {
		screen.SetContent(x, popupY, '-', nil, borderStyle)
	}
	screen.SetContent(popupX+popupWidth-1, popupY, '+', nil, borderStyle)

	// Draw content lines
	for i, line := range t.lines {
		y := popupY + 1 + i
		if y >= popupY+popupHeight-1 {
			break
		}
		screen.SetContent(popupX, y, '|', nil, borderStyle)
		x := popupX + 1
		// Clear interior
		for cx := popupX + 1; cx < popupX+popupWidth-1; cx++ {
			screen.SetContent(cx, y, ' ', nil, style)
		}
		// Draw text
		x = popupX + 2
		for _, ch := range line {
			w := runewidth.RuneWidth(ch)
			if x+w > popupX+popupWidth-2 {
				break
			}
			screen.SetContent(x, y, ch, nil, style)
			x += w
		}
		screen.SetContent(popupX+popupWidth-1, y, '|', nil, borderStyle)
	}

	// Draw bottom border
	bottomY := popupY + popupHeight - 1
	if bottomY > popupY {
		screen.SetContent(popupX, bottomY, '+', nil, borderStyle)
		for x := popupX + 1; x < popupX+popupWidth-1; x++ {
			screen.SetContent(x, bottomY, '-', nil, borderStyle)
		}
		screen.SetContent(popupX+popupWidth-1, bottomY, '+', nil, borderStyle)
	}
}

// HandleEvent dismisses the tooltip on any key or mouse event.
func (t *Tooltip) HandleEvent(ev tcell.Event) bool {
	if !t.visible {
		return false
	}
	// Any key dismisses tooltip
	if _, ok := ev.(*tcell.EventKey); ok {
		t.Hide()
		return true
	}
	return false
}
