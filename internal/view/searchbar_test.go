package view

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/seoji/ted/internal/search"
	"github.com/seoji/ted/internal/syntax"
)

func TestSearchBarShowHide(t *testing.T) {
	theme := syntax.DefaultTheme()
	sb := NewSearchBar(theme)
	if sb.IsVisible() {
		t.Error("should not be visible initially")
	}
	sb.Show(false)
	if !sb.IsVisible() {
		t.Error("should be visible after Show")
	}
	sb.Hide()
	if sb.IsVisible() {
		t.Error("should not be visible after Hide")
	}
}

func TestSearchBarTyping(t *testing.T) {
	theme := syntax.DefaultTheme()
	sb := NewSearchBar(theme)
	sb.Show(false)

	var searchQuery string
	sb.SetOnSearch(func(q string) { searchQuery = q })

	sb.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'h', tcell.ModNone))
	sb.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'i', tcell.ModNone))

	if sb.Query() != "hi" {
		t.Errorf("expected query 'hi', got %q", sb.Query())
	}
	if searchQuery != "hi" {
		t.Errorf("expected search callback 'hi', got %q", searchQuery)
	}
}

func TestSearchBarReplace(t *testing.T) {
	theme := syntax.DefaultTheme()
	sb := NewSearchBar(theme)
	sb.Show(true)

	// Type in search
	sb.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone))
	// Tab to replace
	sb.HandleEvent(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone))
	// Type in replace
	sb.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'b', tcell.ModNone))

	if sb.Query() != "a" {
		t.Errorf("expected query 'a', got %q", sb.Query())
	}
	if sb.Replacement() != "b" {
		t.Errorf("expected replacement 'b', got %q", sb.Replacement())
	}
}

func TestSearchBarDismiss(t *testing.T) {
	theme := syntax.DefaultTheme()
	sb := NewSearchBar(theme)

	dismissed := false
	sb.SetOnDismiss(func() { dismissed = true })

	sb.Show(false)
	sb.HandleEvent(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))

	if !dismissed {
		t.Error("dismiss callback should have been called")
	}
}

func TestSearchBarMatches(t *testing.T) {
	theme := syntax.DefaultTheme()
	sb := NewSearchBar(theme)
	sb.Show(false)

	matches := []search.Match{
		{Line: 0, Col: 0, Length: 2},
		{Line: 3, Col: 5, Length: 2},
	}
	sb.SetMatches(matches)

	if sb.MatchCount() != 2 {
		t.Errorf("expected 2 matches, got %d", sb.MatchCount())
	}
	if sb.CurrentMatch() != 0 {
		t.Errorf("expected current match 0, got %d", sb.CurrentMatch())
	}

	// Enter advances to next
	sb.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if sb.CurrentMatch() != 1 {
		t.Errorf("expected current match 1, got %d", sb.CurrentMatch())
	}

	// Wrap around
	sb.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if sb.CurrentMatch() != 0 {
		t.Errorf("expected current match 0 (wrap), got %d", sb.CurrentMatch())
	}
}

func TestSearchBarBackspace(t *testing.T) {
	theme := syntax.DefaultTheme()
	sb := NewSearchBar(theme)
	sb.Show(false)

	sb.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone))
	sb.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'b', tcell.ModNone))
	sb.HandleEvent(tcell.NewEventKey(tcell.KeyBackspace2, 0, tcell.ModNone))

	if sb.Query() != "a" {
		t.Errorf("expected 'a' after backspace, got %q", sb.Query())
	}
}
