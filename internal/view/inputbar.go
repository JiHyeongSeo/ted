package view

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/JiHyeongSeo/ted/internal/syntax"
	"github.com/JiHyeongSeo/ted/internal/types"
)

// InputBar is a single-line input overlay for prompts (e.g., "Go to line:").
type InputBar struct {
	BaseComponent
	theme     *syntax.Theme
	prompt    string
	value     []rune
	visible   bool
	onSubmit  func(value string)
	onCancel  func()
	onChange  func(value string)
	pasteFunc func() string // returns clipboard text for Ctrl+V
}

// SetPasteFunc wires a clipboard reader so Ctrl+V works in the input bar.
func (ib *InputBar) SetPasteFunc(fn func() string) {
	ib.pasteFunc = fn
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

// SetOnChange sets the callback when input value changes.
func (ib *InputBar) SetOnChange(fn func(value string)) {
	ib.onChange = fn
}

// SetValue pre-fills the input bar with a value.
func (ib *InputBar) SetValue(val string) {
	ib.value = []rune(val)
}

// PasteText appends text to the current value (used for bracketed paste routing).
func (ib *InputBar) PasteText(text string) {
	ib.value = append(ib.value, []rune(text)...)
	if ib.onChange != nil {
		ib.onChange(string(ib.value))
	}
}

// SetBoundsFromScreen positions the input bar centered on screen.
func (ib *InputBar) SetBoundsFromScreen(width, height int) {
	barWidth := width / 2
	if barWidth < 40 {
		barWidth = 40
	}
	if barWidth > width-4 {
		barWidth = width - 4
	}
	startX := (width - barWidth) / 2
	startY := height / 3
	ib.SetBounds(types.Rect{X: startX, Y: startY, Width: barWidth, Height: 3})
}

// Render draws the input bar as a centered dialog.
func (ib *InputBar) Render(screen tcell.Screen) {
	if !ib.visible {
		return
	}
	bounds := ib.Bounds()
	bgStyle := ib.theme.UIStyle("panel")
	borderStyle := bgStyle.Foreground(tcell.ColorSteelBlue)
	promptStyle := bgStyle.Foreground(tcell.ColorGray)
	inputStyle := bgStyle.Foreground(tcell.ColorWhite)

	// Draw 3-line dialog: top border, input, bottom border
	// Top border
	screen.SetContent(bounds.X, bounds.Y, '╭', nil, borderStyle)
	for x := bounds.X + 1; x < bounds.X+bounds.Width-1; x++ {
		screen.SetContent(x, bounds.Y, '─', nil, borderStyle)
	}
	screen.SetContent(bounds.X+bounds.Width-1, bounds.Y, '╮', nil, borderStyle)

	// Middle: input line
	inputY := bounds.Y + 1
	screen.SetContent(bounds.X, inputY, '│', nil, borderStyle)
	for x := bounds.X + 1; x < bounds.X+bounds.Width-1; x++ {
		screen.SetContent(x, inputY, ' ', nil, bgStyle)
	}
	screen.SetContent(bounds.X+bounds.Width-1, inputY, '│', nil, borderStyle)

	// Draw prompt — reserve at least 15 chars for user input
	const minInputSpace = 15
	maxPromptEnd := bounds.X + bounds.Width - 2 - minInputSpace
	x := bounds.X + 2
	promptRunes := []rune(ib.prompt)
	for i, ch := range promptRunes {
		w := runewidth.RuneWidth(ch)
		// If next rune would overflow, draw truncation indicator and stop
		if x+w > maxPromptEnd {
			_ = i
			if x+1 <= maxPromptEnd {
				screen.SetContent(x, inputY, '…', nil, promptStyle)
				x++
			}
			break
		}
		screen.SetContent(x, inputY, ch, nil, promptStyle)
		x += w
	}
	// Draw value
	for _, ch := range ib.value {
		w := runewidth.RuneWidth(ch)
		if x+w > bounds.X+bounds.Width-2 {
			break
		}
		screen.SetContent(x, inputY, ch, nil, inputStyle)
		x += w
	}
	// Show cursor
	if x < bounds.X+bounds.Width-2 {
		screen.SetContent(x, inputY, ' ', nil, inputStyle.Reverse(true))
	}

	// Bottom border
	bottomY := bounds.Y + 2
	screen.SetContent(bounds.X, bottomY, '╰', nil, borderStyle)
	for x := bounds.X + 1; x < bounds.X+bounds.Width-1; x++ {
		screen.SetContent(x, bottomY, '─', nil, borderStyle)
	}
	screen.SetContent(bounds.X+bounds.Width-1, bottomY, '╯', nil, borderStyle)
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
			if ib.onChange != nil {
				ib.onChange(string(ib.value))
			}
		}
		return true
	case tcell.KeyCtrlV:
		if ib.pasteFunc != nil {
			text := ib.pasteFunc()
			// Only use first line of pasted text
			if nl := strings.IndexByte(text, '\n'); nl >= 0 {
				text = text[:nl]
			}
			text = strings.TrimRight(text, "\r")
			ib.value = append(ib.value, []rune(text)...)
			if ib.onChange != nil {
				ib.onChange(string(ib.value))
			}
		}
		return true
	case tcell.KeyRune:
		ib.value = append(ib.value, keyEv.Rune())
		if ib.onChange != nil {
			ib.onChange(string(ib.value))
		}
		return true
	}

	return false
}
