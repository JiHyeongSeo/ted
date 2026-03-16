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

func TestPaletteCommandFiltering(t *testing.T) {
	theme := syntax.DefaultTheme()
	p := NewCommandPalette(theme)
	p.SetItems([]PaletteItem{
		{Label: "File: Save", Command: "file.save"},
		{Label: "File: Open", Command: "file.open"},
		{Label: "Edit: Undo", Command: "edit.undo"},
	})
	p.Show()

	// Type ">" to enter command mode
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, '>', tcell.ModNone))

	// No additional query - all command items
	if len(p.FilteredItems()) != 3 {
		t.Errorf("expected 3 items, got %d", len(p.FilteredItems()))
	}

	// Type query
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModNone))
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone))
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'v', tcell.ModNone))

	if p.Query() != ">sav" {
		t.Errorf("expected query '>sav', got %q", p.Query())
	}

	filtered := p.FilteredItems()
	if len(filtered) < 1 {
		t.Fatal("expected at least 1 filtered item for 'sav'")
	}
	if filtered[0].Command != "file.save" {
		t.Errorf("expected file.save as top match, got %q", filtered[0].Command)
	}
}

func TestPaletteFileFiltering(t *testing.T) {
	theme := syntax.DefaultTheme()
	p := NewCommandPalette(theme)
	p.SetFileItems([]PaletteItem{
		{Label: "main.go", FilePath: "/project/main.go"},
		{Label: "editor.go", FilePath: "/project/editor.go"},
		{Label: "buffer.go", FilePath: "/project/buffer.go"},
	})
	p.Show()

	// Default mode is file search - all file items shown
	if len(p.FilteredItems()) != 3 {
		t.Errorf("expected 3 file items, got %d", len(p.FilteredItems()))
	}

	// Type to filter files
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'm', tcell.ModNone))
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone))

	filtered := p.FilteredItems()
	if len(filtered) < 1 {
		t.Fatal("expected at least 1 filtered file for 'ma'")
	}
	if filtered[0].FilePath != "/project/main.go" {
		t.Errorf("expected main.go as top match, got %q", filtered[0].Label)
	}
}

func TestPaletteNavigation(t *testing.T) {
	theme := syntax.DefaultTheme()
	p := NewCommandPalette(theme)
	p.SetFileItems([]PaletteItem{
		{Label: "A", FilePath: "/a"},
		{Label: "B", FilePath: "/b"},
		{Label: "C", FilePath: "/c"},
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
	// Enter command mode and select
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, '>', tcell.ModNone))
	p.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))

	if selected.Command != "test.cmd" {
		t.Errorf("expected test.cmd, got %q", selected.Command)
	}
	if p.IsVisible() {
		t.Error("palette should hide after selection")
	}
}

func TestPaletteGoToLine(t *testing.T) {
	theme := syntax.DefaultTheme()
	p := NewCommandPalette(theme)

	var gotLine int
	p.SetOnGoToLine(func(line int) { gotLine = line })

	p.Show()
	// Type ":42"
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, ':', tcell.ModNone))
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, '4', tcell.ModNone))
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, '2', tcell.ModNone))
	p.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))

	if gotLine != 42 {
		t.Errorf("expected go to line 42, got %d", gotLine)
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
	p.SetFileItems([]PaletteItem{{Label: "Test", FilePath: "/test"}})
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

func TestFuzzyFilterTracksMatchPositions(t *testing.T) {
	p := NewCommandPalette(nil)
	items := []PaletteItem{
		{Label: "file.save", Description: "Save the current file"},
		{Label: "file.open", Description: "Open a file"},
		{Label: "search.find", Description: "Find text"},
	}
	p.SetItems(items)

	p.query = ">fs"
	p.mode = PaletteModeCommand
	p.filterItems()

	if len(p.filtered) == 0 {
		t.Fatal("expected at least one match for 'fs'")
	}
	if len(p.filtered[0].MatchPositions) == 0 {
		t.Error("expected match positions to be populated")
	}
}

func TestMatchPositionSet(t *testing.T) {
	positions := []int{0, 2, 5}
	set := makePositionSet(positions)

	if !set[0] {
		t.Error("expected position 0 to be in set")
	}
	if set[1] {
		t.Error("expected position 1 to NOT be in set")
	}
	if !set[2] {
		t.Error("expected position 2 to be in set")
	}
}
