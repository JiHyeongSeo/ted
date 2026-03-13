package view

import (
	"testing"

	"github.com/seoji/ted/internal/types"
)

func TestLayoutDefault(t *testing.T) {
	layout := NewLayout()
	regions := layout.Compute(100, 30)

	if len(regions) != 3 {
		t.Errorf("Expected 3 regions, got %d", len(regions))
	}

	tabbar := regions["tabbar"]
	if tabbar.X != 0 || tabbar.Y != 0 || tabbar.Width != 100 || tabbar.Height != 1 {
		t.Errorf("TabBar bounds incorrect: %+v", tabbar)
	}

	statusbar := regions["statusbar"]
	if statusbar.X != 0 || statusbar.Y != 29 || statusbar.Width != 100 || statusbar.Height != 1 {
		t.Errorf("StatusBar bounds incorrect: %+v", statusbar)
	}

	editor := regions["editor"]
	expectedEditor := types.Rect{X: 0, Y: 1, Width: 100, Height: 28}
	if editor != expectedEditor {
		t.Errorf("Editor bounds incorrect: got %+v, expected %+v", editor, expectedEditor)
	}

	if _, exists := regions["sidebar"]; exists {
		t.Error("Sidebar should not exist when not visible")
	}
	if _, exists := regions["panel"]; exists {
		t.Error("Panel should not exist when not visible")
	}
}

func TestLayoutWithSidebar(t *testing.T) {
	layout := NewLayout()
	layout.SetSidebarVisible(true)
	layout.SetSidebarWidth(30)
	regions := layout.Compute(100, 30)

	// sidebar + separator + tabbar + statusbar + editor = 5
	if len(regions) != 5 {
		t.Errorf("Expected 5 regions, got %d", len(regions))
	}

	sidebar := regions["sidebar"]
	expectedSidebar := types.Rect{X: 0, Y: 1, Width: 30, Height: 28}
	if sidebar != expectedSidebar {
		t.Errorf("Sidebar bounds incorrect: got %+v, expected %+v", sidebar, expectedSidebar)
	}

	// Editor starts after sidebar + 1 separator
	editor := regions["editor"]
	expectedEditor := types.Rect{X: 31, Y: 1, Width: 69, Height: 28}
	if editor != expectedEditor {
		t.Errorf("Editor bounds incorrect: got %+v, expected %+v", editor, expectedEditor)
	}
}

func TestLayoutWithPanel(t *testing.T) {
	layout := NewLayout()
	layout.SetPanelVisible(true)
	layout.SetPanelHeight(12)
	regions := layout.Compute(100, 30)

	if len(regions) != 4 {
		t.Errorf("Expected 4 regions, got %d", len(regions))
	}

	editor := regions["editor"]
	expectedEditor := types.Rect{X: 0, Y: 1, Width: 100, Height: 16}
	if editor != expectedEditor {
		t.Errorf("Editor bounds incorrect: got %+v, expected %+v", editor, expectedEditor)
	}

	panel := regions["panel"]
	expectedPanel := types.Rect{X: 0, Y: 17, Width: 100, Height: 12}
	if panel != expectedPanel {
		t.Errorf("Panel bounds incorrect: got %+v, expected %+v", panel, expectedPanel)
	}
}

func TestLayoutWithBoth(t *testing.T) {
	layout := NewLayout()
	layout.SetSidebarVisible(true)
	layout.SetSidebarWidth(25)
	layout.SetPanelVisible(true)
	layout.SetPanelHeight(10)
	regions := layout.Compute(100, 30)

	// sidebar + separator + tabbar + statusbar + editor + panel = 6
	if len(regions) != 6 {
		t.Errorf("Expected 6 regions, got %d", len(regions))
	}

	sidebar := regions["sidebar"]
	expectedSidebar := types.Rect{X: 0, Y: 1, Width: 25, Height: 28}
	if sidebar != expectedSidebar {
		t.Errorf("Sidebar bounds incorrect: got %+v, expected %+v", sidebar, expectedSidebar)
	}

	// Editor: X = 25 (sidebar) + 1 (separator) = 26, Width = 100 - 25 - 1 = 74
	editor := regions["editor"]
	expectedEditor := types.Rect{X: 26, Y: 1, Width: 74, Height: 18}
	if editor != expectedEditor {
		t.Errorf("Editor bounds incorrect: got %+v, expected %+v", editor, expectedEditor)
	}

	panel := regions["panel"]
	expectedPanel := types.Rect{X: 26, Y: 19, Width: 74, Height: 10}
	if panel != expectedPanel {
		t.Errorf("Panel bounds incorrect: got %+v, expected %+v", panel, expectedPanel)
	}
}

func TestLayoutSmallScreen(t *testing.T) {
	layout := NewLayout()
	layout.SetSidebarVisible(true)
	layout.SetSidebarWidth(30)
	layout.SetPanelVisible(true)
	layout.SetPanelHeight(12)

	regions := layout.Compute(40, 10)

	sidebar := regions["sidebar"]
	if sidebar.Width != 30 {
		t.Errorf("Sidebar width should be 30, got %d", sidebar.Width)
	}

	// Editor: 40 - 30 - 1(sep) = 9
	editor := regions["editor"]
	if editor.Width != 9 {
		t.Errorf("Editor width should be 9, got %d", editor.Width)
	}

	panel := regions["panel"]
	if panel.Height != 7 {
		t.Errorf("Panel height should be clamped to 7, got %d", panel.Height)
	}

	if editor.Height != 1 {
		t.Errorf("Editor height should be at least 1, got %d", editor.Height)
	}
}

func TestLayoutMinimumSizes(t *testing.T) {
	layout := NewLayout()
	layout.SetSidebarVisible(true)
	layout.SetSidebarWidth(100) // Request more than screen width
	regions := layout.Compute(50, 20)

	// Clamped: sidebar max = 50-10 = 40
	sidebar := regions["sidebar"]
	if sidebar.Width != 40 {
		t.Errorf("Sidebar should be clamped to 40 (50-10), got %d", sidebar.Width)
	}

	// Editor: 50 - 40 - 1(sep) = 9
	editor := regions["editor"]
	if editor.Width != 9 {
		t.Errorf("Editor width should be 9, got %d", editor.Width)
	}
}

func TestLayoutVisibilityToggle(t *testing.T) {
	layout := NewLayout()

	if layout.SidebarVisible() {
		t.Error("Sidebar should be initially invisible")
	}
	if layout.PanelVisible() {
		t.Error("Panel should be initially invisible")
	}

	layout.SetSidebarVisible(true)
	layout.SetPanelVisible(true)

	if !layout.SidebarVisible() {
		t.Error("Sidebar should be visible after SetSidebarVisible(true)")
	}
	if !layout.PanelVisible() {
		t.Error("Panel should be visible after SetPanelVisible(true)")
	}

	layout.SetSidebarVisible(false)
	layout.SetPanelVisible(false)

	if layout.SidebarVisible() {
		t.Error("Sidebar should be invisible after SetSidebarVisible(false)")
	}
	if layout.PanelVisible() {
		t.Error("Panel should be invisible after SetPanelVisible(false)")
	}
}

func TestLayoutSizeSetters(t *testing.T) {
	layout := NewLayout()

	layout.SetSidebarWidth(40)
	layout.SetSidebarVisible(true)
	regions := layout.Compute(100, 30)
	if regions["sidebar"].Width != 40 {
		t.Errorf("Expected sidebar width 40, got %d", regions["sidebar"].Width)
	}

	layout2 := NewLayout()
	layout2.SetPanelHeight(15)
	layout2.SetPanelVisible(true)
	regions2 := layout2.Compute(100, 30)
	if regions2["panel"].Height != 15 {
		t.Errorf("Expected panel height 15, got %d", regions2["panel"].Height)
	}
}
