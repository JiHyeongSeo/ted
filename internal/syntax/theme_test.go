package syntax

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()
	if theme.Name != "Monokai" {
		t.Errorf("expected name 'Monokai', got %q", theme.Name)
	}
	if _, ok := theme.Colors["keyword"]; !ok {
		t.Error("expected keyword color")
	}
	if _, ok := theme.UI["background"]; !ok {
		t.Error("expected background UI color")
	}
}

func TestParseTheme(t *testing.T) {
	data := []byte(`{
		"name": "Test Theme",
		"colors": {"keyword": "#ff0000"},
		"ui": {"background": "#000000", "foreground": "#ffffff"}
	}`)

	theme, err := ParseTheme(data)
	if err != nil {
		t.Fatal(err)
	}
	if theme.Name != "Test Theme" {
		t.Errorf("expected 'Test Theme', got %q", theme.Name)
	}
	if theme.Colors["keyword"] != "#ff0000" {
		t.Errorf("expected keyword #ff0000, got %s", theme.Colors["keyword"])
	}
}

func TestLoadTheme(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "theme.json")
	os.WriteFile(path, []byte(`{"name":"File Theme","colors":{},"ui":{}}`), 0644)

	theme, err := LoadTheme(path)
	if err != nil {
		t.Fatal(err)
	}
	if theme.Name != "File Theme" {
		t.Errorf("expected 'File Theme', got %q", theme.Name)
	}
}

func TestLoadThemeMissing(t *testing.T) {
	_, err := LoadTheme("/nonexistent/theme.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestResolveColor(t *testing.T) {
	theme := DefaultTheme()

	// Valid hex
	c := theme.ResolveColor("#ff0000")
	if c == tcell.ColorDefault {
		t.Error("expected non-default color for #ff0000")
	}

	// Empty string
	c = theme.ResolveColor("")
	if c != tcell.ColorDefault {
		t.Error("expected default color for empty string")
	}

	// Invalid hex
	c = theme.ResolveColor("#xyz")
	if c != tcell.ColorDefault {
		t.Error("expected default color for invalid hex")
	}
}

func TestTokenStyle(t *testing.T) {
	theme := DefaultTheme()
	style := theme.TokenStyle("keyword")
	// Just verify it returns a valid style (not default)
	if style == tcell.StyleDefault {
		t.Error("expected non-default style for keyword")
	}
}

func TestUIStyle(t *testing.T) {
	theme := DefaultTheme()

	styles := []string{"statusbar", "tabbar.active", "tabbar.inactive",
		"sidebar", "panel", "linenumber", "linenumber.active",
		"selection", "error", "warning", "default"}

	for _, s := range styles {
		style := theme.UIStyle(s)
		if style == (tcell.Style{}) {
			t.Errorf("expected valid style for %q", s)
		}
	}
}
