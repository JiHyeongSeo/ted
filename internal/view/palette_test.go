package view

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/seoji/ted/internal/syntax"
)

func TestPaletteShowHide(t *testing.T) {
	theme := syntax.DefaultTheme()
	p := NewCommandPalette(theme)
	if p.IsVisible() {
		t.Error("should not be visible initially")
	}
	p.Show()
	if !p.IsVisible() {
		t.Error("should be visible after Show")
	}
	p.Hide()
	if p.IsVisible() {
		t.Error("should not be visible after Hide")
	}
}

func TestPaletteFiltering(t *testing.T) {
	theme := syntax.DefaultTheme()
	p := NewCommandPalette(theme)
	p.SetItems([]PaletteItem{
		{Label: "File: Save", Command: "file.save"},
		{Label: "File: Open", Command: "file.open"},
		{Label: "Edit: Undo", Command: "edit.undo"},
	})
	p.Show()

	// No query - all items
	if len(p.FilteredItems()) != 3 {
		t.Errorf("expected 3 items, got %d", len(p.FilteredItems()))
	}

	// Type query
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModNone))
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone))
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'v', tcell.ModNone))

	if p.Query() != "sav" {
		t.Errorf("expected query 'sav', got %q", p.Query())
	}

	filtered := p.FilteredItems()
	if len(filtered) < 1 {
		t.Fatal("expected at least 1 filtered item for 'sav'")
	}
	if filtered[0].Command != "file.save" {
		t.Errorf("expected file.save as top match, got %q", filtered[0].Command)
	}
}

func TestPaletteNavigation(t *testing.T) {
	theme := syntax.DefaultTheme()
	p := NewCommandPalette(theme)
	p.SetItems([]PaletteItem{
		{Label: "A", Command: "a"},
		{Label: "B", Command: "b"},
		{Label: "C", Command: "c"},
	})
	p.Show()

	if p.SelectedIndex() != 0 {
		t.Errorf("expected initial selection 0, got %d", p.SelectedIndex())
	}

	p.HandleEvent(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
	if p.SelectedIndex() != 1 {
		t.Errorf("expected selection 1, got %d", p.SelectedIndex())
	}

	p.HandleEvent(tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone))
	if p.SelectedIndex() != 0 {
		t.Errorf("expected selection 0, got %d", p.SelectedIndex())
	}

	// Don't go below 0
	p.HandleEvent(tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone))
	if p.SelectedIndex() != 0 {
		t.Errorf("expected selection clamped at 0, got %d", p.SelectedIndex())
	}
}

func TestPaletteSelect(t *testing.T) {
	theme := syntax.DefaultTheme()
	p := NewCommandPalette(theme)
	p.SetItems([]PaletteItem{
		{Label: "Test", Command: "test.cmd"},
	})

	var selected PaletteItem
	p.SetOnSelect(func(item PaletteItem) {
		selected = item
	})

	p.Show()
	p.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))

	if selected.Command != "test.cmd" {
		t.Errorf("expected test.cmd, got %q", selected.Command)
	}
	if p.IsVisible() {
		t.Error("palette should hide after selection")
	}
}

func TestPaletteDismiss(t *testing.T) {
	theme := syntax.DefaultTheme()
	p := NewCommandPalette(theme)

	dismissed := false
	p.SetOnDismiss(func() { dismissed = true })

	p.Show()
	p.HandleEvent(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))

	if !dismissed {
		t.Error("dismiss callback should have been called")
	}
	if p.IsVisible() {
		t.Error("palette should hide on escape")
	}
}

func TestPaletteBackspace(t *testing.T) {
	theme := syntax.DefaultTheme()
	p := NewCommandPalette(theme)
	p.SetItems([]PaletteItem{{Label: "Test", Command: "test"}})
	p.Show()

	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone))
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'b', tcell.ModNone))
	if p.Query() != "ab" {
		t.Errorf("expected 'ab', got %q", p.Query())
	}

	p.HandleEvent(tcell.NewEventKey(tcell.KeyBackspace2, 0, tcell.ModNone))
	if p.Query() != "a" {
		t.Errorf("expected 'a', got %q", p.Query())
	}
}
