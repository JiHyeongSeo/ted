package view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/seoji/ted/internal/syntax"
)

// PanelTab represents a tab in the bottom panel.
type PanelTab struct {
	Name    string
	Content []string // lines of content
}

// BottomPanel is a tabbed container at the bottom of the editor.
type BottomPanel struct {
	BaseComponent
	theme     *syntax.Theme
	tabs      []PanelTab
	activeTab int
	scrollY   int
}

// NewBottomPanel creates a new BottomPanel.
func NewBottomPanel(theme *syntax.Theme) *BottomPanel {
	return &BottomPanel{
		theme: theme,
		tabs: []PanelTab{
			{Name: "Problems"},
			{Name: "Output"},
			{Name: "Terminal"},
		},
	}
}

// SetActiveTab sets the active panel tab.
func (p *BottomPanel) SetActiveTab(idx int) {
	if idx >= 0 && idx < len(p.tabs) {
		p.activeTab = idx
		p.scrollY = 0
	}
}

// ActiveTab returns the active tab index.
func (p *BottomPanel) ActiveTab() int {
	return p.activeTab
}

// SetContent sets the content for a panel tab.
func (p *BottomPanel) SetContent(tabIdx int, lines []string) {
	if tabIdx >= 0 && tabIdx < len(p.tabs) {
		p.tabs[tabIdx].Content = lines
	}
}

// AppendContent appends a line to a panel tab.
func (p *BottomPanel) AppendContent(tabIdx int, line string) {
	if tabIdx >= 0 && tabIdx < len(p.tabs) {
		p.tabs[tabIdx].Content = append(p.tabs[tabIdx].Content, line)
	}
}

// ClearContent clears content for a panel tab.
func (p *BottomPanel) ClearContent(tabIdx int) {
	if tabIdx >= 0 && tabIdx < len(p.tabs) {
		p.tabs[tabIdx].Content = nil
	}
}

// Render draws the bottom panel.
func (p *BottomPanel) Render(screen tcell.Screen) {
	bounds := p.Bounds()
	panelStyle := p.theme.UIStyle("panel")
	activeTabStyle := p.theme.UIStyle("tabbar.active")
	inactiveTabStyle := p.theme.UIStyle("tabbar.inactive")

	// Clear area
	for y := bounds.Y; y < bounds.Y+bounds.Height; y++ {
		for x := bounds.X; x < bounds.X+bounds.Width; x++ {
			screen.SetContent(x, y, ' ', nil, panelStyle)
		}
	}

	// Draw tab bar at top of panel
	x := bounds.X
	for i, tab := range p.tabs {
		style := inactiveTabStyle
		if i == p.activeTab {
			style = activeTabStyle
		}

		label := " " + tab.Name + " "
		for _, ch := range label {
			if x >= bounds.X+bounds.Width {
				break
			}
			screen.SetContent(x, bounds.Y, ch, nil, style)
			x++
		}
	}

	// Draw content area
	if p.activeTab >= 0 && p.activeTab < len(p.tabs) {
		content := p.tabs[p.activeTab].Content
		contentHeight := bounds.Height - 1 // subtract tab row

		for i := 0; i < contentHeight && i+p.scrollY < len(content); i++ {
			line := content[i+p.scrollY]
			y := bounds.Y + 1 + i
			x := bounds.X

			for _, ch := range line {
				if x >= bounds.X+bounds.Width {
					break
				}
				screen.SetContent(x, y, ch, nil, panelStyle)
				x++
			}
		}
	}
}

// HandleEvent processes events for the panel.
func (p *BottomPanel) HandleEvent(ev tcell.Event) bool {
	if !p.IsFocused() {
		return false
	}

	keyEv, ok := ev.(*tcell.EventKey)
	if !ok {
		return false
	}

	switch keyEv.Key() {
	case tcell.KeyUp:
		if p.scrollY > 0 {
			p.scrollY--
		}
		return true
	case tcell.KeyDown:
		if p.activeTab >= 0 && p.activeTab < len(p.tabs) {
			maxScroll := len(p.tabs[p.activeTab].Content) - (p.Bounds().Height - 1)
			if p.scrollY < maxScroll {
				p.scrollY++
			}
		}
		return true
	}

	return false
}
