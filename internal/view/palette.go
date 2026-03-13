package view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/sahilm/fuzzy"
	"github.com/seoji/ted/internal/syntax"
	"github.com/seoji/ted/internal/types"
)

// PaletteItem represents an item in the command palette.
type PaletteItem struct {
	Label       string
	Description string
	Command     string
}

// CommandPalette is a fuzzy-search overlay for commands.
type CommandPalette struct {
	BaseComponent
	theme       *syntax.Theme
	items       []PaletteItem
	filtered    []PaletteItem
	query       string
	selectedIdx int
	visible     bool
	onSelect    func(item PaletteItem)
	onDismiss   func()
}

// NewCommandPalette creates a new CommandPalette.
func NewCommandPalette(theme *syntax.Theme) *CommandPalette {
	return &CommandPalette{
		theme: theme,
	}
}

// SetItems sets the available palette items.
func (p *CommandPalette) SetItems(items []PaletteItem) {
	p.items = items
	p.filterItems()
}

// SetOnSelect sets the callback when an item is selected.
func (p *CommandPalette) SetOnSelect(fn func(item PaletteItem)) {
	p.onSelect = fn
}

// SetOnDismiss sets the callback when the palette is dismissed.
func (p *CommandPalette) SetOnDismiss(fn func()) {
	p.onDismiss = fn
}

// Show makes the palette visible and resets state.
func (p *CommandPalette) Show() {
	p.visible = true
	p.query = ""
	p.selectedIdx = 0
	p.filterItems()
}

// Hide hides the palette.
func (p *CommandPalette) Hide() {
	p.visible = false
	p.query = ""
}

// IsVisible returns whether the palette is currently shown.
func (p *CommandPalette) IsVisible() bool {
	return p.visible
}

// Query returns the current search query.
func (p *CommandPalette) Query() string {
	return p.query
}

// SelectedIndex returns the currently selected item index.
func (p *CommandPalette) SelectedIndex() int {
	return p.selectedIdx
}

// FilteredItems returns the currently filtered items.
func (p *CommandPalette) FilteredItems() []PaletteItem {
	return p.filtered
}

// Render draws the command palette overlay.
func (p *CommandPalette) Render(screen tcell.Screen) {
	if !p.visible {
		return
	}

	bounds := p.Bounds()
	// Palette occupies center of screen, width ~60%, height up to 12 items + input
	paletteWidth := bounds.Width * 3 / 5
	if paletteWidth < 30 {
		paletteWidth = 30
	}
	if paletteWidth > bounds.Width-4 {
		paletteWidth = bounds.Width - 4
	}
	maxItems := 10
	paletteHeight := len(p.filtered) + 2 // +1 for input, +1 for border
	if paletteHeight > maxItems+2 {
		paletteHeight = maxItems + 2
	}

	startX := bounds.X + (bounds.Width-paletteWidth)/2
	startY := bounds.Y + 2

	bgStyle := p.theme.UIStyle("panel")
	fgStyle := p.theme.UIStyle("default")
	selStyle := p.theme.UIStyle("selection")

	// Draw input row
	for x := startX; x < startX+paletteWidth; x++ {
		screen.SetContent(x, startY, ' ', nil, bgStyle)
	}
	prompt := "> " + p.query
	x := startX + 1
	for _, ch := range prompt {
		if x >= startX+paletteWidth-1 {
			break
		}
		screen.SetContent(x, startY, ch, nil, fgStyle)
		x++
	}

	// Draw filtered items
	for i := 0; i < paletteHeight-2 && i < len(p.filtered); i++ {
		y := startY + 1 + i
		style := bgStyle
		if i == p.selectedIdx {
			style = selStyle
		}

		for x := startX; x < startX+paletteWidth; x++ {
			screen.SetContent(x, y, ' ', nil, style)
		}

		label := "  " + p.filtered[i].Label
		if p.filtered[i].Description != "" {
			label += "  " + p.filtered[i].Description
		}

		x := startX + 1
		for _, ch := range label {
			if x >= startX+paletteWidth-1 {
				break
			}
			screen.SetContent(x, y, ch, nil, style)
			x++
		}
	}
}

// HandleEvent processes key events for the palette.
func (p *CommandPalette) HandleEvent(ev tcell.Event) bool {
	if !p.visible {
		return false
	}

	keyEv, ok := ev.(*tcell.EventKey)
	if !ok {
		return false
	}

	switch keyEv.Key() {
	case tcell.KeyEscape:
		p.Hide()
		if p.onDismiss != nil {
			p.onDismiss()
		}
		return true
	case tcell.KeyEnter:
		if p.selectedIdx >= 0 && p.selectedIdx < len(p.filtered) {
			item := p.filtered[p.selectedIdx]
			p.Hide()
			if p.onSelect != nil {
				p.onSelect(item)
			}
		}
		return true
	case tcell.KeyUp:
		if p.selectedIdx > 0 {
			p.selectedIdx--
		}
		return true
	case tcell.KeyDown:
		if p.selectedIdx < len(p.filtered)-1 {
			p.selectedIdx++
		}
		return true
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(p.query) > 0 {
			p.query = p.query[:len(p.query)-1]
			p.filterItems()
		}
		return true
	case tcell.KeyRune:
		p.query += string(keyEv.Rune())
		p.filterItems()
		return true
	}

	return false
}

func (p *CommandPalette) filterItems() {
	if p.query == "" {
		p.filtered = make([]PaletteItem, len(p.items))
		copy(p.filtered, p.items)
	} else {
		labels := make([]string, len(p.items))
		for i, item := range p.items {
			labels[i] = item.Label
		}
		matches := fuzzy.Find(p.query, labels)
		p.filtered = make([]PaletteItem, len(matches))
		for i, m := range matches {
			p.filtered[i] = p.items[m.Index]
		}
	}
	p.selectedIdx = 0
}

// SetBoundsFromScreen sets palette bounds based on total screen size.
func (p *CommandPalette) SetBoundsFromScreen(width, height int) {
	p.SetBounds(types.Rect{X: 0, Y: 0, Width: width, Height: height})
}
