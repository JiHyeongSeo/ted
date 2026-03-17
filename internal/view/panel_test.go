package view

import (
	"testing"

	"github.com/JiHyeongSeo/ted/internal/syntax"
)

func TestBottomPanelCreation(t *testing.T) {
	theme := syntax.DefaultTheme()
	p := NewBottomPanel(theme)
	if len(p.tabs) != 2 {
		t.Errorf("expected 2 tabs, got %d", len(p.tabs))
	}
	if p.ActiveTab() != 0 {
		t.Errorf("expected active tab 0, got %d", p.ActiveTab())
	}
}

func TestBottomPanelSetActiveTab(t *testing.T) {
	theme := syntax.DefaultTheme()
	p := NewBottomPanel(theme)
	p.SetActiveTab(1)
	if p.ActiveTab() != 1 {
		t.Errorf("expected active tab 1, got %d", p.ActiveTab())
	}
}

func TestBottomPanelContent(t *testing.T) {
	theme := syntax.DefaultTheme()
	p := NewBottomPanel(theme)

	p.SetContent(0, []string{"error: undefined x", "warning: unused y"})
	if len(p.tabs[0].Content) != 2 {
		t.Errorf("expected 2 lines, got %d", len(p.tabs[0].Content))
	}

	p.AppendContent(0, "info: done")
	if len(p.tabs[0].Content) != 3 {
		t.Errorf("expected 3 lines after append, got %d", len(p.tabs[0].Content))
	}

	p.ClearContent(0)
	if len(p.tabs[0].Content) != 0 {
		t.Errorf("expected 0 lines after clear, got %d", len(p.tabs[0].Content))
	}
}

func TestBottomPanelInvalidTab(t *testing.T) {
	theme := syntax.DefaultTheme()
	p := NewBottomPanel(theme)
	p.SetActiveTab(99) // should be no-op
	if p.ActiveTab() != 0 {
		t.Errorf("expected active tab 0, got %d", p.ActiveTab())
	}
}
