package view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
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
	theme       *syntax.Theme
	tabs        []PanelTab
	activeTab   int
	scrollY     int
	selectedRow int // highlighted row in content (-1 = none)
	onLineClick func(tabIdx int, lineIdx int) // callback when a content line is clicked
}

// NewBottomPanel creates a new BottomPanel.
func NewBottomPanel(theme *syntax.Theme) *BottomPanel {
	return &BottomPanel{
		theme:       theme,
		selectedRow: -1,
		tabs: []PanelTab{
			{Name: "Problems"},
			{Name: "Output"},
			{Name: "Terminal"},
		},
	}
}

// SetOnLineClick sets the callback when a content line is clicked.
func (p *BottomPanel) SetOnLineClick(fn func(tabIdx int, lineIdx int)) {
	p.onLineClick = fn
}

// SetActiveTab sets the active panel tab.
func (p *BottomPanel) SetActiveTab(idx int) {
	if idx >= 0 && idx < len(p.tabs) {
		p.activeTab = idx
		p.scrollY = 0
		p.selectedRow = -1
	}
}

// ActiveTab returns the active tab index.
func (p *BottomPanel) ActiveTab() int {
	return p.activeTab
}

// SelectedRow returns the currently selected content row index.
func (p *BottomPanel) SelectedRow() int {
	return p.selectedRow
}

// SetContent sets the content for a panel tab.
func (p *BottomPanel) SetContent(tabIdx int, lines []string) {
	if tabIdx >= 0 && tabIdx < len(p.tabs) {
		p.tabs[tabIdx].Content = lines
		if tabIdx == p.activeTab {
			p.selectedRow = -1
		}
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

// ContentLineCount returns the number of content lines in the active tab.
func (p *BottomPanel) ContentLineCount() int {
	if p.activeTab >= 0 && p.activeTab < len(p.tabs) {
		return len(p.tabs[p.activeTab].Content)
	}
	return 0
}

// Render draws the bottom panel.
func (p *BottomPanel) Render(screen tcell.Screen) {
	bounds := p.Bounds()
	panelStyle := p.theme.UIStyle("panel")
	activeTabStyle := p.theme.UIStyle("tabbar.active")
	inactiveTabStyle := p.theme.UIStyle("tabbar.inactive")
	selectedStyle := panelStyle.Background(tcell.ColorDarkBlue)

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
			lineIdx := i + p.scrollY
			line := content[lineIdx]
			y := bounds.Y + 1 + i
			x := bounds.X

			style := panelStyle
			if lineIdx == p.selectedRow {
				style = selectedStyle
				// Fill entire row with selected style
				for cx := bounds.X; cx < bounds.X+bounds.Width; cx++ {
					screen.SetContent(cx, y, ' ', nil, style)
				}
			}

			for _, ch := range line {
				w := runewidth.RuneWidth(ch)
				if x+w > bounds.X+bounds.Width {
					break
				}
				screen.SetContent(x, y, ch, nil, style)
				x += w
			}
		}
	}
}

// HandleEvent processes events for the panel.
func (p *BottomPanel) HandleEvent(ev tcell.Event) bool {
	switch tev := ev.(type) {
	case *tcell.EventKey:
		return p.handleKey(tev)
	case *tcell.EventMouse:
		return p.handleMouse(tev)
	}
	return false
}

func (p *BottomPanel) handleKey(ev *tcell.EventKey) bool {
	contentHeight := p.Bounds().Height - 1
	contentLen := p.ContentLineCount()

	switch ev.Key() {
	case tcell.KeyUp:
		if p.selectedRow > 0 {
			p.selectedRow--
			// Scroll up if needed
			if p.selectedRow < p.scrollY {
				p.scrollY = p.selectedRow
			}
		}
		return true
	case tcell.KeyDown:
		if p.selectedRow < contentLen-1 {
			p.selectedRow++
			// Scroll down if needed
			if p.selectedRow >= p.scrollY+contentHeight {
				p.scrollY = p.selectedRow - contentHeight + 1
			}
		}
		return true
	case tcell.KeyEnter:
		if p.selectedRow >= 0 && p.onLineClick != nil {
			p.onLineClick(p.activeTab, p.selectedRow)
		}
		return true
	}
	return false
}

func (p *BottomPanel) handleMouse(ev *tcell.EventMouse) bool {
	mx, my := ev.Position()
	bounds := p.Bounds()

	// Check if mouse is in panel area
	if mx < bounds.X || mx >= bounds.X+bounds.Width || my < bounds.Y || my >= bounds.Y+bounds.Height {
		return false
	}

	// Mouse wheel scrolling
	contentHeight := bounds.Height - 1
	maxScroll := p.ContentLineCount() - contentHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if ev.Buttons()&tcell.WheelUp != 0 {
		if p.scrollY > 0 {
			p.scrollY -= 3
			if p.scrollY < 0 {
				p.scrollY = 0
			}
		}
		return true
	}
	if ev.Buttons()&tcell.WheelDown != 0 {
		if p.scrollY < maxScroll {
			p.scrollY += 3
			if p.scrollY > maxScroll {
				p.scrollY = maxScroll
			}
		}
		return true
	}

	if ev.Buttons()&tcell.Button1 == 0 {
		return false
	}

	// Click on tab bar (first row)
	if my == bounds.Y {
		x := bounds.X
		for i, tab := range p.tabs {
			tabWidth := len(" " + tab.Name + " ")
			if mx >= x && mx < x+tabWidth {
				p.SetActiveTab(i)
				return true
			}
			x += tabWidth
		}
		return true
	}

	// Click on content row
	row := my - bounds.Y - 1 // -1 for tab bar row
	lineIdx := row + p.scrollY
	if lineIdx >= 0 && lineIdx < p.ContentLineCount() {
		p.selectedRow = lineIdx
		if p.onLineClick != nil {
			p.onLineClick(p.activeTab, lineIdx)
		}
		return true
	}

	return false
}
