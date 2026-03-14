package editor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/seoji/ted/internal/config"
	"github.com/seoji/ted/internal/syntax"
)

func TestNewEditor(t *testing.T) {
	cfg, _ := config.LoadDefaults()
	theme := syntax.DefaultTheme()
	e := New(cfg, theme)
	if e == nil {
		t.Fatal("expected non-nil editor")
	}
	if e.tabs.Count() != 0 {
		t.Errorf("expected 0 tabs, got %d", e.tabs.Count())
	}
}

func TestEditorOpenEmpty(t *testing.T) {
	cfg, _ := config.LoadDefaults()
	theme := syntax.DefaultTheme()
	e := New(cfg, theme)
	e.OpenEmpty()
	if e.tabs.Count() != 1 {
		t.Errorf("expected 1 tab, got %d", e.tabs.Count())
	}
}

func TestEditorOpenFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	os.WriteFile(path, []byte("package main"), 0644)

	cfg, _ := config.LoadDefaults()
	theme := syntax.DefaultTheme()
	e := New(cfg, theme)

	err := e.OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	if e.tabs.Count() != 1 {
		t.Errorf("expected 1 tab, got %d", e.tabs.Count())
	}
	if e.tabs.Active().Language != "go" {
		t.Errorf("expected language 'go', got %q", e.tabs.Active().Language)
	}
}

func TestEditorOpenFileTwice(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	os.WriteFile(path, []byte("package main"), 0644)

	cfg, _ := config.LoadDefaults()
	theme := syntax.DefaultTheme()
	e := New(cfg, theme)

	e.OpenFile(path)
	e.OpenFile(path) // should not create new tab
	if e.tabs.Count() != 1 {
		t.Errorf("expected 1 tab (dedup), got %d", e.tabs.Count())
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path string
		lang string
	}{
		{"main.go", "go"},
		{"app.py", "python"},
		{"index.js", "javascript"},
		{"styles.css", "css"},
		{"README.md", "markdown"},
		{"config.json", "json"},
		{"unknown.xyz", "text"},
	}

	for _, tt := range tests {
		got := detectLanguage(tt.path)
		if got != tt.lang {
			t.Errorf("detectLanguage(%q) = %q, want %q", tt.path, got, tt.lang)
		}
	}
}

func TestEditorExecuteCommand(t *testing.T) {
	cfg, _ := config.LoadDefaults()
	theme := syntax.DefaultTheme()
	e := New(cfg, theme)
	e.OpenEmpty()

	// Sidebar is always visible now
	if !e.layout.SidebarVisible() {
		t.Error("sidebar should be visible initially")
	}
}
