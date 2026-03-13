package view

import (
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/seoji/ted/internal/syntax"
)

// Tab represents a single tab in the tab bar.
type Tab struct {
	Title    string
	FilePath string
	Dirty    bool
}

// TabBar displays open file tabs at the top of the editor.
type TabBar struct {
	BaseComponent
	theme     *syntax.Theme
	tabs      []Tab
	activeIdx int
}

// NewTabBar creates a new TabBar.
func NewTabBar(theme *syntax.Theme) *TabBar {
	return &TabBar{
		theme: theme,
	}
}

// SetTabs updates the tab list.
func (tb *TabBar) SetTabs(tabs []Tab, activeIdx int) {
	tb.tabs = tabs
	tb.activeIdx = activeIdx
}

// ActiveIndex returns the currently active tab index.
func (tb *TabBar) ActiveIndex() int {
	return tb.activeIdx
}

// TabCount returns the number of open tabs.
func (tb *TabBar) TabCount() int {
	return len(tb.tabs)
}

// Render draws the tab bar.
func (tb *TabBar) Render(screen tcell.Screen) {
	bounds := tb.Bounds()
	inactiveStyle := tb.theme.UIStyle("tabbar.inactive")
	activeStyle := tb.theme.UIStyle("tabbar.active")

	// Clear the tab bar area
	for x := bounds.X; x < bounds.X+bounds.Width; x++ {
		screen.SetContent(x, bounds.Y, ' ', nil, inactiveStyle)
	}

	x := bounds.X
	for i, tab := range tb.tabs {
		if x >= bounds.X+bounds.Width {
			break
		}

		style := inactiveStyle
		if i == tb.activeIdx {
			style = activeStyle
		}

		// Tab format: " filename.ext [x] "
		title := tab.Title
		if title == "" && tab.FilePath != "" {
			title = filepath.Base(tab.FilePath)
		}
		if title == "" {
			title = "[No Name]"
		}

		label := " " + title
		if tab.Dirty {
			label += " ●"
		}
		label += " "

		for _, ch := range label {
			if x >= bounds.X+bounds.Width {
				break
			}
			screen.SetContent(x, bounds.Y, ch, nil, style)
			x++
		}

		// Separator
		if x < bounds.X+bounds.Width {
			screen.SetContent(x, bounds.Y, '│', nil, inactiveStyle)
			x++
		}
	}
}

// HandleEvent processes click events on tabs.
func (tb *TabBar) HandleEvent(ev tcell.Event) bool {
	if mouse, ok := ev.(*tcell.EventMouse); ok {
		if mouse.Buttons()&tcell.Button1 != 0 {
			mx, my := mouse.Position()
			bounds := tb.Bounds()
			if my == bounds.Y && mx >= bounds.X && mx < bounds.X+bounds.Width {
				// Determine which tab was clicked
				x := bounds.X
				for i, tab := range tb.tabs {
					title := tab.Title
					if title == "" && tab.FilePath != "" {
						title = filepath.Base(tab.FilePath)
					}
					if title == "" {
						title = "[No Name]"
					}
					tabWidth := len(" "+title+" ") + 1 // +1 for separator
					if tab.Dirty {
						tabWidth += 2
					}
					if mx >= x && mx < x+tabWidth {
						tb.activeIdx = i
						return true
					}
					x += tabWidth
				}
			}
		}
	}
	return false
}
