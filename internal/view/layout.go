package view

import "github.com/JiHyeongSeo/ted/internal/types"

// Layout calculates component bounds from total screen dimensions.
type Layout struct {
	sidebarWidth    int
	sidebarVisible  bool
	panelHeight     int
	panelVisible    bool
	tabBarHeight    int
	statusBarHeight int
	splitMode       bool
	splitRatio      float64 // 0.0-1.0, how much space left pane gets
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
		splitMode:       false,
		splitRatio:      0.5,
	}
}

func (l *Layout) SetSidebarVisible(v bool) { l.sidebarVisible = v }
func (l *Layout) SetPanelVisible(v bool)   { l.panelVisible = v }
func (l *Layout) SetSidebarWidth(w int)    { l.sidebarWidth = w }
func (l *Layout) SetPanelHeight(h int)     { l.panelHeight = h }
func (l *Layout) SidebarVisible() bool     { return l.sidebarVisible }
func (l *Layout) PanelVisible() bool       { return l.panelVisible }

func (l *Layout) SetSplitMode(v bool) { l.splitMode = v }
func (l *Layout) SplitMode() bool     { return l.splitMode }
func (l *Layout) SetSplitRatio(r float64) {
	if r < 0.2 {
		r = 0.2
	}
	if r > 0.8 {
		r = 0.8
	}
	l.splitRatio = r
}

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

	// Add 1-col separator between sidebar and editor
	separatorWidth := 0
	if sidebarWidth > 0 {
		separatorWidth = 1
	}
	editorWidth := width - sidebarWidth - separatorWidth

	if sidebarWidth > 0 {
		regions["sidebar"] = types.Rect{X: 0, Y: y, Width: sidebarWidth, Height: middleHeight}
		regions["separator"] = types.Rect{X: sidebarWidth, Y: y, Width: 1, Height: middleHeight}
	}

	if l.splitMode {
		splitSepWidth := 1
		leftWidth := int(float64(editorWidth-splitSepWidth) * l.splitRatio)
		rightWidth := editorWidth - splitSepWidth - leftWidth
		editorX := sidebarWidth + separatorWidth

		regions["editor.left"] = types.Rect{
			X: editorX, Y: y, Width: leftWidth, Height: editorHeight,
		}
		regions["editor.separator"] = types.Rect{
			X: editorX + leftWidth, Y: y, Width: 1, Height: editorHeight,
		}
		regions["editor.right"] = types.Rect{
			X: editorX + leftWidth + splitSepWidth, Y: y, Width: rightWidth, Height: editorHeight,
		}
	} else {
		regions["editor"] = types.Rect{
			X: sidebarWidth + separatorWidth, Y: y, Width: editorWidth, Height: editorHeight,
		}
	}

	if panelHeight > 0 {
		regions["panel"] = types.Rect{X: sidebarWidth + separatorWidth, Y: y + editorHeight, Width: editorWidth, Height: panelHeight}
	}

	return regions
}
