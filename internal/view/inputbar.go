package view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/seoji/ted/internal/syntax"
	"github.com/seoji/ted/internal/types"
)

// InputBar is a single-line input overlay for prompts (e.g., "Go to line:").
type InputBar struct {
	BaseComponent
	theme    *syntax.Theme
	prompt   string
	value    []rune
	visible  bool
	onSubmit func(value string)
	onCancel func()
}

// NewInputBar creates a new InputBar.
func NewInputBar(theme *syntax.Theme) *InputBar {
	return &InputBar{theme: theme}
}

// Show displays the input bar with the given prompt.
func (ib *InputBar) Show(prompt string) {
	ib.visible = true
	ib.prompt = prompt
	ib.value = nil
}

// Hide hides the input bar.
func (ib *InputBar) Hide() {
	ib.visible = false
	ib.value = nil
}

// IsVisible returns whether the input bar is shown.
func (ib *InputBar) IsVisible() bool {
	return ib.visible
}

// SetOnSubmit sets the callback when Enter is pressed.
func (ib *InputBar) SetOnSubmit(fn func(value string)) {
	ib.onSubmit = fn
}

// SetOnCancel sets the callback when Escape is pressed.
func (ib *InputBar) SetOnCancel(fn func()) {
	ib.onCancel = fn
}

// SetBoundsFromScreen positions the input bar centered near the top.
func (ib *InputBar) SetBoundsFromScreen(width, height int) {
	barWidth := width * 2 / 5
	if barWidth < 30 {
		barWidth = 30
	}
	if barWidth > width-4 {
		barWidth = width - 4
	}
	startX := (width - barWidth) / 2
	ib.SetBounds(types.Rect{X: startX, Y: 2, Width: barWidth, Height: 1})
}

// Render draws the input bar.
func (ib *InputBar) Render(screen tcell.Screen) {
	if !ib.visible {
		return
	}
	bounds := ib.Bounds()
	style := ib.theme.UIStyle("panel")

	// Clear row
	for x := bounds.X; x < bounds.X+bounds.Width; x++ {
		screen.SetContent(x, bounds.Y, ' ', nil, style)
	}

	text := []rune(ib.prompt)
	text = append(text, ib.value...)
	x := bounds.X + 1
	for _, ch := range text {
		w := runewidth.RuneWidth(ch)
		if x+w > bounds.X+bounds.Width-1 {
			break
		}
		screen.SetContent(x, bounds.Y, ch, nil, style)
		x += w
	}
}

// HandleEvent processes key events.
func (ib *InputBar) HandleEvent(ev tcell.Event) bool {
	if !ib.visible {
		return false
	}

	keyEv, ok := ev.(*tcell.EventKey)
	if !ok {
		return false
	}

	switch keyEv.Key() {
	case tcell.KeyEscape:
		ib.Hide()
		if ib.onCancel != nil {
			ib.onCancel()
		}
		return true
	case tcell.KeyEnter:
		val := string(ib.value)
		ib.Hide()
		if ib.onSubmit != nil {
			ib.onSubmit(val)
		}
		return true
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(ib.value) > 0 {
			ib.value = ib.value[:len(ib.value)-1]
		}
		return true
	case tcell.KeyRune:
		ib.value = append(ib.value, keyEv.Rune())
		return true
	}

	return false
}
