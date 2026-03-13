package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := LoadDefaults()
	if err != nil {
		t.Fatalf("LoadDefaults failed: %v", err)
	}
	if cfg.Editor.TabSize != 4 {
		t.Errorf("expected tabSize=4, got %d", cfg.Editor.TabSize)
	}
	if !cfg.Editor.InsertSpaces {
		t.Error("expected insertSpaces=true")
	}
	if cfg.Editor.CursorStyle != "block" {
		t.Errorf("expected cursorStyle=block, got %s", cfg.Editor.CursorStyle)
	}
	if cfg.Sidebar.Width != 30 {
		t.Errorf("expected sidebar.width=30, got %d", cfg.Sidebar.Width)
	}
	if cfg.Syntax.Theme != "catppuccin-mocha" {
		t.Errorf("expected theme catppuccin-mocha, got %s", cfg.Syntax.Theme)
	}
}

func TestMergeUserConfig(t *testing.T) {
	dir := t.TempDir()
	userFile := filepath.Join(dir, "settings.json")
	err := os.WriteFile(userFile, []byte(`{"editor":{"tabSize":2}}`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, _ := LoadDefaults()
	err = MergeFromFile(cfg, userFile)
	if err != nil {
		t.Fatalf("MergeFromFile failed: %v", err)
	}
	if cfg.Editor.TabSize != 2 {
		t.Errorf("expected tabSize=2 after merge, got %d", cfg.Editor.TabSize)
	}
	// Non-overridden fields remain default
	if !cfg.Editor.InsertSpaces {
		t.Error("expected insertSpaces to remain true")
	}
}

func TestMergeProjectConfig(t *testing.T) {
	dir := t.TempDir()
	projFile := filepath.Join(dir, "settings.json")
	err := os.WriteFile(projFile, []byte(`{"editor":{"tabSize":8,"wordWrap":true}}`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, _ := LoadDefaults()
	err = MergeFromFile(cfg, projFile)
	if err != nil {
		t.Fatalf("MergeFromFile failed: %v", err)
	}
	if cfg.Editor.TabSize != 8 {
		t.Errorf("expected tabSize=8, got %d", cfg.Editor.TabSize)
	}
	if !cfg.Editor.WordWrap {
		t.Error("expected wordWrap=true after project override")
	}
}

func TestLoadReturnsDefaultOnMissingFile(t *testing.T) {
	cfg, _ := LoadDefaults()
	err := MergeFromFile(cfg, "/nonexistent/path/settings.json")
	if err != nil {
		t.Error("expected no error for missing file")
	}
	// Should remain defaults
	if cfg.Editor.TabSize != 4 {
		t.Errorf("expected default tabSize=4, got %d", cfg.Editor.TabSize)
	}
}

func TestDefaultUserConfigDir(t *testing.T) {
	// With XDG_CONFIG_HOME set
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	dir := DefaultUserConfigDir()
	if dir != "/tmp/xdg/ted" {
		t.Errorf("expected /tmp/xdg/ted, got %s", dir)
	}

	// Without XDG_CONFIG_HOME, falls back to ~/.config/ted
	t.Setenv("XDG_CONFIG_HOME", "")
	dir = DefaultUserConfigDir()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "ted")
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	userDir := filepath.Join(dir, "user")
	projDir := filepath.Join(dir, "project", ".ted")
	os.MkdirAll(userDir, 0755)
	os.MkdirAll(projDir, 0755)

	os.WriteFile(filepath.Join(userDir, "settings.json"), []byte(`{"editor":{"tabSize":2}}`), 0644)
	os.WriteFile(filepath.Join(projDir, "settings.json"), []byte(`{"editor":{"tabSize":8}}`), 0644)

	cfg, err := Load(userDir, filepath.Join(dir, "project"))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	// Project overrides user
	if cfg.Editor.TabSize != 8 {
		t.Errorf("expected tabSize=8 (project wins), got %d", cfg.Editor.TabSize)
	}
}
