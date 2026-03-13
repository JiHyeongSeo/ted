package view

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/seoji/ted/internal/search"
	"github.com/seoji/ted/internal/syntax"
)

// SearchBar provides inline find/replace functionality.
type SearchBar struct {
	BaseComponent
	theme       *syntax.Theme
	query       string
	replacement string
	visible     bool
	replaceMode bool
	cursorInReplace bool
	matches     []search.Match
	currentMatch int
	onSearch    func(query string)
	onReplace   func(query, replacement string)
	onReplaceAll func(query, replacement string)
	onDismiss   func()
}

// NewSearchBar creates a new SearchBar.
func NewSearchBar(theme *syntax.Theme) *SearchBar {
	return &SearchBar{
		theme: theme,
	}
}

// Show displays the search bar.
func (sb *SearchBar) Show(replaceMode bool) {
	sb.visible = true
	sb.replaceMode = replaceMode
	sb.query = ""
	sb.replacement = ""
	sb.cursorInReplace = false
	sb.currentMatch = 0
	sb.matches = nil
}

// Hide hides the search bar.
func (sb *SearchBar) Hide() {
	sb.visible = false
}

// IsVisible returns whether the search bar is shown.
func (sb *SearchBar) IsVisible() bool {
	return sb.visible
}

// ReplaceMode returns whether the search bar is in replace mode.
func (sb *SearchBar) ReplaceMode() bool {
	return sb.replaceMode
}

// Query returns the current search query.
func (sb *SearchBar) Query() string {
	return sb.query
}

// Replacement returns the current replacement text.
func (sb *SearchBar) Replacement() string {
	return sb.replacement
}

// SetMatches updates the match results.
func (sb *SearchBar) SetMatches(matches []search.Match) {
	sb.matches = matches
	if sb.currentMatch >= len(matches) {
		sb.currentMatch = 0
	}
}

// CurrentMatch returns the index of the currently highlighted match.
func (sb *SearchBar) CurrentMatch() int {
	return sb.currentMatch
}

// MatchCount returns the number of matches.
func (sb *SearchBar) MatchCount() int {
	return len(sb.matches)
}

// SetOnSearch sets the search callback.
func (sb *SearchBar) SetOnSearch(fn func(query string)) {
	sb.onSearch = fn
}

// SetOnReplace sets the replace callback.
func (sb *SearchBar) SetOnReplace(fn func(query, replacement string)) {
	sb.onReplace = fn
}

// SetOnReplaceAll sets the replace-all callback.
func (sb *SearchBar) SetOnReplaceAll(fn func(query, replacement string)) {
	sb.onReplaceAll = fn
}

// SetOnDismiss sets the dismiss callback.
func (sb *SearchBar) SetOnDismiss(fn func()) {
	sb.onDismiss = fn
}

// Render draws the search bar.
func (sb *SearchBar) Render(screen tcell.Screen) {
	if !sb.visible {
		return
	}

	bounds := sb.Bounds()
	style := sb.theme.UIStyle("panel")

	// Clear area
	for y := bounds.Y; y < bounds.Y+bounds.Height; y++ {
		for x := bounds.X; x < bounds.X+bounds.Width; x++ {
			screen.SetContent(x, y, ' ', nil, style)
		}
	}

	// Search row
	searchLabel := " Find: "
	matchInfo := ""
	if len(sb.matches) > 0 {
		matchInfo = fmt.Sprintf(" (%d/%d)", sb.currentMatch+1, len(sb.matches))
	} else if sb.query != "" {
		matchInfo = " (0 results)"
	}

	x := bounds.X
	for _, ch := range searchLabel {
		if x < bounds.X+bounds.Width {
			screen.SetContent(x, bounds.Y, ch, nil, style)
			x++
		}
	}
	for _, ch := range sb.query {
		if x < bounds.X+bounds.Width {
			screen.SetContent(x, bounds.Y, ch, nil, style)
			x++
		}
	}
	for _, ch := range matchInfo {
		if x < bounds.X+bounds.Width {
			screen.SetContent(x, bounds.Y, ch, nil, style)
			x++
		}
	}

	// Replace row (if in replace mode)
	if sb.replaceMode && bounds.Height > 1 {
		replaceLabel := " Replace: "
		x = bounds.X
		for _, ch := range replaceLabel {
			if x < bounds.X+bounds.Width {
				screen.SetContent(x, bounds.Y+1, ch, nil, style)
				x++
			}
		}
		for _, ch := range sb.replacement {
			if x < bounds.X+bounds.Width {
				screen.SetContent(x, bounds.Y+1, ch, nil, style)
				x++
			}
		}
	}
}

// HandleEvent processes key events for the search bar.
func (sb *SearchBar) HandleEvent(ev tcell.Event) bool {
	if !sb.visible {
		return false
	}

	keyEv, ok := ev.(*tcell.EventKey)
	if !ok {
		return false
	}

	switch keyEv.Key() {
	case tcell.KeyEscape:
		sb.Hide()
		if sb.onDismiss != nil {
			sb.onDismiss()
		}
		return true
	case tcell.KeyEnter:
		if sb.cursorInReplace && sb.onReplace != nil {
			sb.onReplace(sb.query, sb.replacement)
		} else {
			// Move to next match
			if len(sb.matches) > 0 {
				sb.currentMatch = (sb.currentMatch + 1) % len(sb.matches)
			}
		}
		return true
	case tcell.KeyTab:
		if sb.replaceMode {
			sb.cursorInReplace = !sb.cursorInReplace
		}
		return true
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if sb.cursorInReplace {
			if len(sb.replacement) > 0 {
				sb.replacement = sb.replacement[:len(sb.replacement)-1]
			}
		} else {
			if len(sb.query) > 0 {
				sb.query = sb.query[:len(sb.query)-1]
				if sb.onSearch != nil {
					sb.onSearch(sb.query)
				}
			}
		}
		return true
	case tcell.KeyRune:
		if sb.cursorInReplace {
			sb.replacement += string(keyEv.Rune())
		} else {
			sb.query += string(keyEv.Rune())
			if sb.onSearch != nil {
				sb.onSearch(sb.query)
			}
		}
		return true
	}

	return false
}
