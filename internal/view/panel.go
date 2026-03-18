package view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/JiHyeongSeo/ted/internal/syntax"
)

// PanelSpan is a styled text segment for rich panel lines.
type PanelSpan struct {
	Text  string
	Style tcell.Style
}

// RichLine is a single panel line with colored spans and an optional navigation tag.
// Tag == -1 means the line is not navigable (header/separator).
type RichLine struct {
	Spans []PanelSpan
	Tag   int // result index; -1 = not navigable
}

// PanelTab represents a tab in the bottom panel.
type PanelTab struct {
	Name      string
	Content   []string   // plain text lines (fallback)
	RichLines []RichLine // colored lines; used when non-nil
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

// SetContent sets plain-text content for a panel tab.
func (p *BottomPanel) SetContent(tabIdx int, lines []string) {
	if tabIdx >= 0 && tabIdx < len(p.tabs) {
		p.tabs[tabIdx].Content = lines
		p.tabs[tabIdx].RichLines = nil
		if tabIdx == p.activeTab {
			p.selectedRow = -1
		}
	}
}

// SetRichContent sets colored content for a panel tab.
func (p *BottomPanel) SetRichContent(tabIdx int, lines []RichLine) {
	if tabIdx >= 0 && tabIdx < len(p.tabs) {
		p.tabs[tabIdx].RichLines = lines
		p.tabs[tabIdx].Content = nil
		if tabIdx == p.activeTab {
			p.selectedRow = -1
		}
	}
}

// LineTag returns the navigation tag of a content line (-1 = not navigable).
func (p *BottomPanel) LineTag(lineIdx int) int {
	if p.activeTab >= 0 && p.activeTab < len(p.tabs) {
		tab := &p.tabs[p.activeTab]
		if tab.RichLines != nil && lineIdx >= 0 && lineIdx < len(tab.RichLines) {
			return tab.RichLines[lineIdx].Tag
		}
	}
	return -1
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
		tab := &p.tabs[p.activeTab]
		if tab.RichLines != nil {
			return len(tab.RichLines)
		}
		return len(tab.Content)
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
		tab := &p.tabs[p.activeTab]
		contentHeight := bounds.Height - 1 // subtract tab row

		if tab.RichLines != nil {
			for i := 0; i < contentHeight && i+p.scrollY < len(tab.RichLines); i++ {
				lineIdx := i + p.scrollY
				rl := tab.RichLines[lineIdx]
				y := bounds.Y + 1 + i
				x := bounds.X
				isSelected := lineIdx == p.selectedRow && rl.Tag >= 0

				if isSelected {
					for cx := bounds.X; cx < bounds.X+bounds.Width; cx++ {
						screen.SetContent(cx, y, ' ', nil, selectedStyle)
					}
				}

				for _, span := range rl.Spans {
					st := span.Style
					if isSelected {
						// Keep foreground, use selected background
						fg, _, _ := span.Style.Decompose()
						st = selectedStyle.Foreground(fg)
					}
					for _, ch := range span.Text {
						w := runewidth.RuneWidth(ch)
						if x+w > bounds.X+bounds.Width {
							break
						}
						screen.SetContent(x, y, ch, nil, st)
						x += w
					}
				}
			}
		} else {
			for i := 0; i < contentHeight && i+p.scrollY < len(tab.Content); i++ {
				lineIdx := i + p.scrollY
				line := tab.Content[lineIdx]
				y := bounds.Y + 1 + i
				x := bounds.X

				style := panelStyle
				if lineIdx == p.selectedRow {
					style = selectedStyle
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
	if contentHeight < 1 {
		contentHeight = 1
	}
	contentLen := p.ContentLineCount()

	clampScroll := func() {
		if p.scrollY < 0 {
			p.scrollY = 0
		}
		maxScroll := contentLen - contentHeight
		if maxScroll < 0 {
			maxScroll = 0
		}
		if p.scrollY > maxScroll {
			p.scrollY = maxScroll
		}
	}
	ensureVisible := func() {
		if p.selectedRow < p.scrollY {
			p.scrollY = p.selectedRow
		} else if p.selectedRow >= p.scrollY+contentHeight {
			p.scrollY = p.selectedRow - contentHeight + 1
		}
		clampScroll()
	}

	switch ev.Key() {
	case tcell.KeyUp:
		if p.selectedRow > 0 {
			p.selectedRow--
			ensureVisible()
		}
		return true
	case tcell.KeyDown:
		if p.selectedRow < contentLen-1 {
			p.selectedRow++
			ensureVisible()
		}
		return true
	case tcell.KeyPgUp:
		p.selectedRow -= contentHeight
		if p.selectedRow < 0 {
			p.selectedRow = 0
		}
		ensureVisible()
		return true
	case tcell.KeyPgDn:
		p.selectedRow += contentHeight
		if p.selectedRow >= contentLen {
			p.selectedRow = contentLen - 1
		}
		if p.selectedRow < 0 {
			p.selectedRow = 0
		}
		ensureVisible()
		return true
	case tcell.KeyHome:
		p.selectedRow = 0
		p.scrollY = 0
		return true
	case tcell.KeyEnd:
		if contentLen > 0 {
			p.selectedRow = contentLen - 1
		}
		clampScroll()
		ensureVisible()
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
