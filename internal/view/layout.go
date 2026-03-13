package view

import "github.com/seoji/ted/internal/types"

// LayoutRegion defines a named area with sizing rules.
type LayoutRegion struct {
	Name    string
	Fixed   int  // fixed size in the layout direction (0 = flex)
	Visible bool
}

// Layout calculates component bounds from total screen dimensions.
type Layout struct {
	sidebarWidth    int
	sidebarVisible  bool
	panelHeight     int
	panelVisible    bool
	tabBarHeight    int
	statusBarHeight int
}

// NewLayout creates a Layout with default sizes.
func NewLayout() *Layout {
	return &Layout{
		sidebarWidth:    30,
		sidebarVisible:  false,
		panelHeight:     12,
		panelVisible:    false,
		tabBarHeight:    1,
		statusBarHeight: 1,
	}
}

func (l *Layout) SetSidebarVisible(v bool) { l.sidebarVisible = v }
func (l *Layout) SetPanelVisible(v bool)   { l.panelVisible = v }
func (l *Layout) SetSidebarWidth(w int)    { l.sidebarWidth = w }
func (l *Layout) SetPanelHeight(h int)     { l.panelHeight = h }
func (l *Layout) SidebarVisible() bool     { return l.sidebarVisible }
func (l *Layout) PanelVisible() bool       { return l.panelVisible }

// Compute calculates bounds for all regions given total screen dimensions.
// Returns a map of region name -> Rect.
func (l *Layout) Compute(width, height int) map[string]types.Rect {
	regions := make(map[string]types.Rect)

	y := 0

	// TabBar at top
	regions["tabbar"] = types.Rect{X: 0, Y: y, Width: width, Height: l.tabBarHeight}
	y += l.tabBarHeight

	// StatusBar at bottom
	statusY := height - l.statusBarHeight
	regions["statusbar"] = types.Rect{X: 0, Y: statusY, Width: width, Height: l.statusBarHeight}

	// Remaining vertical space
	middleHeight := statusY - y

	// Panel from bottom of middle area
	panelHeight := 0
	if l.panelVisible && l.panelHeight > 0 {
		panelHeight = l.panelHeight
		if panelHeight > middleHeight-1 { // leave at least 1 row for editor
			panelHeight = middleHeight - 1
		}
	}

	// Sidebar from left
	sidebarWidth := 0
	if l.sidebarVisible && l.sidebarWidth > 0 {
		sidebarWidth = l.sidebarWidth
		if sidebarWidth > width-10 { // leave at least 10 cols for editor
			sidebarWidth = width - 10
		}
	}

	editorHeight := middleHeight - panelHeight
	editorWidth := width - sidebarWidth

	if sidebarWidth > 0 {
		regions["sidebar"] = types.Rect{X: 0, Y: y, Width: sidebarWidth, Height: middleHeight}
	}

	regions["editor"] = types.Rect{X: sidebarWidth, Y: y, Width: editorWidth, Height: editorHeight}

	if panelHeight > 0 {
		regions["panel"] = types.Rect{X: sidebarWidth, Y: y + editorHeight, Width: editorWidth, Height: panelHeight}
	}

	return regions
}
