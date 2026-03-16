package view

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/JiHyeongSeo/ted/internal/syntax"
	"github.com/JiHyeongSeo/ted/internal/types"
)


// ListPicker is a centered overlay for selecting from a list of items.
type ListPicker struct {
	BaseComponent
	theme       *syntax.Theme
	title       string
	items       []string
	filtered    []int // indices into items
	filter      []rune
	selectedIdx int
	scrollY     int
	visible     bool
	onSelect    func(item string)
	onCancel    func()
}

// NewListPicker creates a new ListPicker.
func NewListPicker(theme *syntax.Theme) *ListPicker {
	return &ListPicker{theme: theme}
}

// Show displays the picker with the given title and items.
func (lp *ListPicker) Show(title string, items []string) {
	lp.title = title
	lp.items = items
	lp.filter = nil
	lp.selectedIdx = 0
	lp.scrollY = 0
	lp.visible = true
	lp.updateFilter()
}

// Hide hides the picker.
func (lp *ListPicker) Hide() {
	lp.visible = false
}

// IsVisible returns whether the picker is shown.
func (lp *ListPicker) IsVisible() bool {
	return lp.visible
}

// SetOnSelect sets the callback when an item is selected.
func (lp *ListPicker) SetOnSelect(fn func(item string)) {
	lp.onSelect = fn
}

// SetOnCancel sets the callback when the picker is cancelled.
func (lp *ListPicker) SetOnCancel(fn func()) {
	lp.onCancel = fn
}

// SetBoundsFromScreen positions the picker centered on screen.
func (lp *ListPicker) SetBoundsFromScreen(width, height int) {
	w := width / 2
	if w < 40 {
		w = 40
	}
	if w > width-4 {
		w = width - 4
	}
	h := height / 2
	if h < 10 {
		h = 10
	}
	if h > height-4 {
		h = height - 4
	}
	x := (width - w) / 2
	y := (height - h) / 2
	lp.SetBounds(types.Rect{X: x, Y: y, Width: w, Height: h})
}

func (lp *ListPicker) updateFilter() {
	query := strings.ToLower(string(lp.filter))
	lp.filtered = nil
	for i, item := range lp.items {
		if query == "" || strings.Contains(strings.ToLower(item), query) {
			lp.filtered = append(lp.filtered, i)
		}
	}
	if lp.selectedIdx >= len(lp.filtered) {
		lp.selectedIdx = len(lp.filtered) - 1
	}
	if lp.selectedIdx < 0 {
		lp.selectedIdx = 0
	}
	lp.scrollY = 0
}

// Render draws the list picker.
func (lp *ListPicker) Render(screen tcell.Screen) {
	if !lp.visible {
		return
	}
	bounds := lp.Bounds()
	bgStyle := lp.theme.UIStyle("panel")
	borderStyle := bgStyle.Foreground(tcell.ColorSteelBlue)
	titleStyle := bgStyle.Foreground(tcell.ColorWhite).Bold(true)
	filterStyle := bgStyle.Foreground(tcell.ColorWhite)
	itemStyle := bgStyle.Foreground(tcell.ColorGray)
	selectedStyle := bgStyle.Background(tcell.ColorSteelBlue).Foreground(tcell.ColorWhite)
	dimStyle := bgStyle.Foreground(tcell.ColorDarkGray)

	// Top border with title
	screen.SetContent(bounds.X, bounds.Y, '╭', nil, borderStyle)
	for x := bounds.X + 1; x < bounds.X+bounds.Width-1; x++ {
		screen.SetContent(x, bounds.Y, '─', nil, borderStyle)
	}
	screen.SetContent(bounds.X+bounds.Width-1, bounds.Y, '╮', nil, borderStyle)

	// Title on top border
	titleText := " " + lp.title + " "
	tx := bounds.X + 2
	for _, ch := range titleText {
		if tx >= bounds.X+bounds.Width-2 {
			break
		}
		screen.SetContent(tx, bounds.Y, ch, nil, titleStyle)
		tx++
	}

	// Filter input line (row 1)
	filterY := bounds.Y + 1
	screen.SetContent(bounds.X, filterY, '│', nil, borderStyle)
	for x := bounds.X + 1; x < bounds.X+bounds.Width-1; x++ {
		screen.SetContent(x, filterY, ' ', nil, bgStyle)
	}
	screen.SetContent(bounds.X+bounds.Width-1, filterY, '│', nil, borderStyle)

	// Draw filter icon and text
	screen.SetContent(bounds.X+2, filterY, '/', nil, dimStyle)
	fx := bounds.X + 4
	for _, ch := range lp.filter {
		if fx >= bounds.X+bounds.Width-2 {
			break
		}
		screen.SetContent(fx, filterY, ch, nil, filterStyle)
		fx++
	}
	if fx < bounds.X+bounds.Width-2 {
		screen.SetContent(fx, filterY, ' ', nil, filterStyle.Reverse(true))
	}

	// Separator
	sepY := bounds.Y + 2
	screen.SetContent(bounds.X, sepY, '├', nil, borderStyle)
	for x := bounds.X + 1; x < bounds.X+bounds.Width-1; x++ {
		screen.SetContent(x, sepY, '─', nil, borderStyle)
	}
	screen.SetContent(bounds.X+bounds.Width-1, sepY, '┤', nil, borderStyle)

	// Items
	listHeight := bounds.Height - 4 // top border + filter + separator + bottom border
	for i := 0; i < listHeight; i++ {
		y := bounds.Y + 3 + i
		idx := lp.scrollY + i

		screen.SetContent(bounds.X, y, '│', nil, borderStyle)
		screen.SetContent(bounds.X+bounds.Width-1, y, '│', nil, borderStyle)

		if idx >= len(lp.filtered) {
			for x := bounds.X + 1; x < bounds.X+bounds.Width-1; x++ {
				screen.SetContent(x, y, ' ', nil, bgStyle)
			}
			continue
		}

		style := itemStyle
		if idx == lp.selectedIdx {
			style = selectedStyle
		}

		// Clear row
		for x := bounds.X + 1; x < bounds.X+bounds.Width-1; x++ {
			screen.SetContent(x, y, ' ', nil, style)
		}

		// Draw item text
		item := lp.items[lp.filtered[idx]]
		ix := bounds.X + 3
		for _, ch := range item {
			w := runewidth.RuneWidth(ch)
			if ix+w > bounds.X+bounds.Width-2 {
				break
			}
			screen.SetContent(ix, y, ch, nil, style)
			ix += w
		}
	}

	// Bottom border
	bottomY := bounds.Y + bounds.Height - 1
	screen.SetContent(bounds.X, bottomY, '╰', nil, borderStyle)
	for x := bounds.X + 1; x < bounds.X+bounds.Width-1; x++ {
		screen.SetContent(x, bottomY, '─', nil, borderStyle)
	}
	screen.SetContent(bounds.X+bounds.Width-1, bottomY, '╯', nil, borderStyle)

}

// HandleEvent processes key events for the picker.
func (lp *ListPicker) HandleEvent(ev tcell.Event) bool {
	if !lp.visible {
		return false
	}

	keyEv, ok := ev.(*tcell.EventKey)
	if !ok {
		return false
	}

	listHeight := lp.Bounds().Height - 4

	switch keyEv.Key() {
	case tcell.KeyEscape:
		lp.Hide()
		if lp.onCancel != nil {
			lp.onCancel()
		}
		return true
	case tcell.KeyEnter:
		if lp.selectedIdx >= 0 && lp.selectedIdx < len(lp.filtered) {
			item := lp.items[lp.filtered[lp.selectedIdx]]
			lp.Hide()
			if lp.onSelect != nil {
				lp.onSelect(item)
			}
		}
		return true
	case tcell.KeyUp:
		if lp.selectedIdx > 0 {
			lp.selectedIdx--
			if lp.selectedIdx < lp.scrollY {
				lp.scrollY = lp.selectedIdx
			}
		}
		return true
	case tcell.KeyDown:
		if lp.selectedIdx < len(lp.filtered)-1 {
			lp.selectedIdx++
			if lp.selectedIdx >= lp.scrollY+listHeight {
				lp.scrollY = lp.selectedIdx - listHeight + 1
			}
		}
		return true
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(lp.filter) > 0 {
			lp.filter = lp.filter[:len(lp.filter)-1]
			lp.updateFilter()
		}
		return true
	case tcell.KeyRune:
		lp.filter = append(lp.filter, keyEv.Rune())
		lp.updateFilter()
		return true
	}

	return false
}
