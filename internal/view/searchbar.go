package view

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/seoji/ted/internal/search"
	"github.com/seoji/ted/internal/syntax"
)

// SearchBar provides inline find/replace functionality.
type SearchBar struct {
	BaseComponent
	theme           *syntax.Theme
	query           []rune
	replacement     []rune
	visible         bool
	replaceMode     bool
	cursorInReplace bool
	matches         []search.Match
	currentMatch    int
	onSearch        func(query string)
	onReplace       func(query, replacement string)
	onReplaceAll    func(query, replacement string)
	onDismiss       func()
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
	sb.query = nil
	sb.replacement = nil
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
	return string(sb.query)
}

// Replacement returns the current replacement text.
func (sb *SearchBar) Replacement() string {
	return string(sb.replacement)
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

// drawRunes draws runes to the screen with proper wide character handling.
// Returns the next x position.
func drawRunes(screen tcell.Screen, x, y, maxX int, runes []rune, style tcell.Style) int {
	for _, ch := range runes {
		w := runewidth.RuneWidth(ch)
		if x+w > maxX {
			break
		}
		screen.SetContent(x, y, ch, nil, style)
		x += w
	}
	return x
}

// Render draws the search bar.
func (sb *SearchBar) Render(screen tcell.Screen) {
	if !sb.visible {
		return
	}

	bounds := sb.Bounds()
	style := sb.theme.UIStyle("panel")
	maxX := bounds.X + bounds.Width

	// Clear area
	for y := bounds.Y; y < bounds.Y+bounds.Height; y++ {
		for x := bounds.X; x < maxX; x++ {
			screen.SetContent(x, y, ' ', nil, style)
		}
	}

	// Search row
	searchLabel := []rune(" Find: ")
	matchInfo := ""
	if len(sb.matches) > 0 {
		matchInfo = fmt.Sprintf(" (%d/%d)", sb.currentMatch+1, len(sb.matches))
	} else if len(sb.query) > 0 {
		matchInfo = " (0 results)"
	}

	x := drawRunes(screen, bounds.X, bounds.Y, maxX, searchLabel, style)
	x = drawRunes(screen, x, bounds.Y, maxX, sb.query, style)
	drawRunes(screen, x, bounds.Y, maxX, []rune(matchInfo), style)

	// Replace row (if in replace mode)
	if sb.replaceMode && bounds.Height > 1 {
		replaceLabel := []rune(" Replace: ")
		x = drawRunes(screen, bounds.X, bounds.Y+1, maxX, replaceLabel, style)
		drawRunes(screen, x, bounds.Y+1, maxX, sb.replacement, style)
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
			sb.onReplace(string(sb.query), string(sb.replacement))
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
					sb.onSearch(string(sb.query))
				}
			}
		}
		return true
	case tcell.KeyRune:
		if sb.cursorInReplace {
			sb.replacement = append(sb.replacement, keyEv.Rune())
		} else {
			sb.query = append(sb.query, keyEv.Rune())
			if sb.onSearch != nil {
				sb.onSearch(string(sb.query))
			}
		}
		return true
	}

	return false
}
