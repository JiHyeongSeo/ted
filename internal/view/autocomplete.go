package view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/JiHyeongSeo/ted/internal/syntax"
)

// CompletionItem represents a single autocomplete suggestion.
type CompletionItem struct {
	Label      string
	Detail     string
	InsertText string
}

// AutocompletePopup renders a completion list near the cursor.
type AutocompletePopup struct {
	BaseComponent
	theme       *syntax.Theme
	items       []CompletionItem
	selectedIdx int
	visible     bool
	anchorX     int // screen position to anchor the popup
	anchorY     int
	onSelect    func(item CompletionItem)
	onDismiss   func()
}

// NewAutocompletePopup creates a new AutocompletePopup.
func NewAutocompletePopup(theme *syntax.Theme) *AutocompletePopup {
	return &AutocompletePopup{theme: theme}
}

// Show displays the popup at the given screen position with items.
func (a *AutocompletePopup) Show(items []CompletionItem, anchorX, anchorY int) {
	if len(items) == 0 {
		return
	}
	a.items = items
	a.selectedIdx = 0
	a.anchorX = anchorX
	a.anchorY = anchorY
	a.visible = true
}

// Hide hides the popup.
func (a *AutocompletePopup) Hide() {
	a.visible = false
	a.items = nil
}

// IsVisible returns whether the popup is shown.
func (a *AutocompletePopup) IsVisible() bool {
	return a.visible
}

// SetOnSelect sets the selection callback.
func (a *AutocompletePopup) SetOnSelect(fn func(item CompletionItem)) {
	a.onSelect = fn
}

// SetOnDismiss sets the dismiss callback.
func (a *AutocompletePopup) SetOnDismiss(fn func()) {
	a.onDismiss = fn
}

// SelectedItem returns the currently selected item, or nil.
func (a *AutocompletePopup) SelectedItem() *CompletionItem {
	if a.selectedIdx >= 0 && a.selectedIdx < len(a.items) {
		return &a.items[a.selectedIdx]
	}
	return nil
}

// Render draws the autocomplete popup.
func (a *AutocompletePopup) Render(screen tcell.Screen) {
	if !a.visible || len(a.items) == 0 {
		return
	}

	screenW, screenH := screen.Size()

	// Calculate popup dimensions
	maxVisible := 8
	if len(a.items) < maxVisible {
		maxVisible = len(a.items)
	}

	// Find widest item for popup width
	popupWidth := 20
	for _, item := range a.items {
		w := runewidth.StringWidth(item.Label)
		if item.Detail != "" {
			w += 2 + runewidth.StringWidth(item.Detail)
		}
		if w+4 > popupWidth {
			popupWidth = w + 4
		}
	}
	if popupWidth > 60 {
		popupWidth = 60
	}

	// Position: below cursor, or above if not enough space
	popupX := a.anchorX
	popupY := a.anchorY + 1
	if popupX+popupWidth > screenW {
		popupX = screenW - popupWidth
	}
	if popupX < 0 {
		popupX = 0
	}
	if popupY+maxVisible > screenH {
		popupY = a.anchorY - maxVisible
		if popupY < 0 {
			popupY = 0
		}
	}

	bgStyle := a.theme.UIStyle("panel")
	selStyle := a.theme.UIStyle("selection")

	// Ensure selected item is in visible range
	scrollOffset := 0
	if a.selectedIdx >= maxVisible {
		scrollOffset = a.selectedIdx - maxVisible + 1
	}

	for i := 0; i < maxVisible; i++ {
		itemIdx := i + scrollOffset
		if itemIdx >= len(a.items) {
			break
		}
		item := a.items[itemIdx]
		y := popupY + i
		style := bgStyle
		if itemIdx == a.selectedIdx {
			style = selStyle
		}

		// Clear row
		for x := popupX; x < popupX+popupWidth; x++ {
			screen.SetContent(x, y, ' ', nil, style)
		}

		// Draw label
		x := popupX + 1
		for _, ch := range item.Label {
			w := runewidth.RuneWidth(ch)
			if x+w > popupX+popupWidth-1 {
				break
			}
			screen.SetContent(x, y, ch, nil, style)
			x += w
		}

		// Draw detail (dimmed)
		if item.Detail != "" {
			detailStyle := style.Foreground(tcell.ColorDarkGray)
			x += 2
			for _, ch := range item.Detail {
				w := runewidth.RuneWidth(ch)
				if x+w > popupX+popupWidth-1 {
					break
				}
				screen.SetContent(x, y, ch, nil, detailStyle)
				x += w
			}
		}
	}
}

// HandleEvent processes key events for the autocomplete popup.
func (a *AutocompletePopup) HandleEvent(ev tcell.Event) bool {
	if !a.visible {
		return false
	}

	keyEv, ok := ev.(*tcell.EventKey)
	if !ok {
		return false
	}

	switch keyEv.Key() {
	case tcell.KeyEscape:
		a.Hide()
		if a.onDismiss != nil {
			a.onDismiss()
		}
		return true
	case tcell.KeyEnter, tcell.KeyTab:
		if a.selectedIdx >= 0 && a.selectedIdx < len(a.items) {
			item := a.items[a.selectedIdx]
			a.Hide()
			if a.onSelect != nil {
				a.onSelect(item)
			}
		}
		return true
	case tcell.KeyUp:
		if a.selectedIdx > 0 {
			a.selectedIdx--
		}
		return true
	case tcell.KeyDown:
		if a.selectedIdx < len(a.items)-1 {
			a.selectedIdx++
		}
		return true
	}

	return false
}
