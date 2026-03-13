package view

import (
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/sahilm/fuzzy"
	"github.com/seoji/ted/internal/syntax"
	"github.com/seoji/ted/internal/types"
)

// PaletteMode determines how the palette interprets input.
type PaletteMode int

const (
	PaletteModeFile    PaletteMode = iota // default: fuzzy file search
	PaletteModeCommand                    // ">" prefix: command search
	PaletteModeGoLine                     // ":" prefix: go to line
)

// PaletteItem represents an item in the command palette.
type PaletteItem struct {
	Label       string
	Description string
	Command     string
	FilePath    string // for file items
}

// CommandPalette is a fuzzy-search overlay for commands and files.
type CommandPalette struct {
	BaseComponent
	theme        *syntax.Theme
	commandItems []PaletteItem
	fileItems    []PaletteItem
	filtered     []PaletteItem
	query        string
	mode         PaletteMode
	selectedIdx  int
	visible      bool
	onSelect     func(item PaletteItem)
	onFileOpen   func(path string)
	onGoToLine   func(line int)
	onDismiss    func()
}

// NewCommandPalette creates a new CommandPalette.
func NewCommandPalette(theme *syntax.Theme) *CommandPalette {
	return &CommandPalette{
		theme: theme,
	}
}

// SetItems sets the available command items.
func (p *CommandPalette) SetItems(items []PaletteItem) {
	p.commandItems = items
	p.filterItems()
}

// SetFileItems sets the available file items for file search mode.
func (p *CommandPalette) SetFileItems(items []PaletteItem) {
	p.fileItems = items
}

// SetOnSelect sets the callback when a command item is selected.
func (p *CommandPalette) SetOnSelect(fn func(item PaletteItem)) {
	p.onSelect = fn
}

// OnSelect returns the current select callback.
func (p *CommandPalette) OnSelect() func(item PaletteItem) {
	return p.onSelect
}

// SetOnFileOpen sets the callback when a file is selected.
func (p *CommandPalette) SetOnFileOpen(fn func(path string)) {
	p.onFileOpen = fn
}

// SetOnGoToLine sets the callback for go-to-line mode.
func (p *CommandPalette) SetOnGoToLine(fn func(line int)) {
	p.onGoToLine = fn
}

// SetOnDismiss sets the callback when the palette is dismissed.
func (p *CommandPalette) SetOnDismiss(fn func()) {
	p.onDismiss = fn
}

// Show makes the palette visible and resets state.
func (p *CommandPalette) Show() {
	p.visible = true
	p.query = ""
	p.mode = PaletteModeFile
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
	paletteWidth := bounds.Width * 3 / 5
	if paletteWidth < 30 {
		paletteWidth = 30
	}
	if paletteWidth > bounds.Width-4 {
		paletteWidth = bounds.Width - 4
	}
	maxItems := 10
	paletteHeight := len(p.filtered) + 2
	if paletteHeight > maxItems+2 {
		paletteHeight = maxItems + 2
	}
	if p.mode == PaletteModeGoLine {
		paletteHeight = 2 // just the input row
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

	// Prompt varies by mode
	var prompt string
	switch p.mode {
	case PaletteModeCommand:
		prompt = "> " + p.query
	case PaletteModeGoLine:
		prompt = ":" + strings.TrimPrefix(p.query, ":")
	default:
		prompt = p.query
	}

	x := startX + 1
	for _, ch := range prompt {
		w := runewidth.RuneWidth(ch)
		if x+w >= startX+paletteWidth-1 {
			break
		}
		screen.SetContent(x, startY, ch, nil, fgStyle)
		x += w
	}

	// Draw hint text when empty
	if p.query == "" {
		hint := "Search files... (> for commands, : for line)"
		hintStyle := bgStyle.Foreground(tcell.ColorDarkGray)
		hx := startX + 1
		for _, ch := range hint {
			if hx >= startX+paletteWidth-1 {
				break
			}
			screen.SetContent(hx, startY, ch, nil, hintStyle)
			hx++
		}
	}

	// Draw filtered items (not in go-to-line mode)
	if p.mode != PaletteModeGoLine {
		visibleCount := paletteHeight - 2
		// Calculate scroll offset to keep selected item visible
		scrollOffset := 0
		if p.selectedIdx >= visibleCount {
			scrollOffset = p.selectedIdx - visibleCount + 1
		}

		for i := 0; i < visibleCount && i+scrollOffset < len(p.filtered); i++ {
			itemIdx := i + scrollOffset
			y := startY + 1 + i
			style := bgStyle
			if itemIdx == p.selectedIdx {
				style = selStyle
			}

			for x := startX; x < startX+paletteWidth; x++ {
				screen.SetContent(x, y, ' ', nil, style)
			}

			label := "  " + p.filtered[itemIdx].Label
			if p.filtered[itemIdx].Description != "" {
				label += "  " + p.filtered[itemIdx].Description
			}

			x := startX + 1
			for _, ch := range label {
				w := runewidth.RuneWidth(ch)
				if x+w >= startX+paletteWidth-1 {
					break
				}
				screen.SetContent(x, y, ch, nil, style)
				x += w
			}
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
		p.handleSelect()
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
			p.detectMode()
			p.filterItems()
		}
		return true
	case tcell.KeyRune:
		p.query += string(keyEv.Rune())
		p.detectMode()
		p.filterItems()
		return true
	}

	return false
}

func (p *CommandPalette) handleSelect() {
	switch p.mode {
	case PaletteModeGoLine:
		numStr := strings.TrimPrefix(p.query, ":")
		if num, err := strconv.Atoi(strings.TrimSpace(numStr)); err == nil && num >= 1 {
			p.Hide()
			if p.onGoToLine != nil {
				p.onGoToLine(num)
			}
		}
	case PaletteModeCommand:
		if p.selectedIdx >= 0 && p.selectedIdx < len(p.filtered) {
			item := p.filtered[p.selectedIdx]
			p.Hide()
			if p.onSelect != nil {
				p.onSelect(item)
			}
		}
	default: // file mode
		if p.selectedIdx >= 0 && p.selectedIdx < len(p.filtered) {
			item := p.filtered[p.selectedIdx]
			p.Hide()
			if item.FilePath != "" && p.onFileOpen != nil {
				p.onFileOpen(item.FilePath)
			} else if p.onSelect != nil {
				p.onSelect(item)
			}
		}
	}
}

func (p *CommandPalette) detectMode() {
	if strings.HasPrefix(p.query, ">") {
		p.mode = PaletteModeCommand
	} else if strings.HasPrefix(p.query, ":") {
		p.mode = PaletteModeGoLine
	} else {
		p.mode = PaletteModeFile
	}
}

func (p *CommandPalette) filterItems() {
	switch p.mode {
	case PaletteModeCommand:
		searchQuery := strings.TrimPrefix(p.query, ">")
		searchQuery = strings.TrimSpace(searchQuery)
		p.fuzzyFilter(p.commandItems, searchQuery)
	case PaletteModeGoLine:
		p.filtered = nil
	default: // file mode
		p.fuzzyFilter(p.fileItems, p.query)
	}
	p.selectedIdx = 0
}

func (p *CommandPalette) fuzzyFilter(items []PaletteItem, query string) {
	if query == "" {
		p.filtered = make([]PaletteItem, len(items))
		copy(p.filtered, items)
		return
	}
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = item.Label
	}
	matches := fuzzy.Find(query, labels)
	p.filtered = make([]PaletteItem, len(matches))
	for i, m := range matches {
		p.filtered[i] = items[m.Index]
	}
}

// SetBoundsFromScreen sets palette bounds based on total screen size.
func (p *CommandPalette) SetBoundsFromScreen(width, height int) {
	p.SetBounds(types.Rect{X: 0, Y: 0, Width: width, Height: height})
}
