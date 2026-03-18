package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is the top-level configuration structure.
type Config struct {
	Editor  EditorConfig         `json:"editor"`
	Sidebar SidebarConfig        `json:"sidebar"`
	Panel   PanelConfig          `json:"panel"`
	Search  SearchConfig         `json:"search"`
	Syntax  SyntaxConfig         `json:"syntax"`
	LSP     map[string]LSPConfig `json:"lsp"`
}

type EditorConfig struct {
	TabSize      int    `json:"tabSize"`
	InsertSpaces bool   `json:"insertSpaces"`
	WordWrap     bool   `json:"wordWrap"`
	LineNumbers  bool   `json:"lineNumbers"`
	CursorStyle  string `json:"cursorStyle"`
}

type SidebarConfig struct {
	Width   int  `json:"width"`
	Visible bool `json:"visible"`
}

type PanelConfig struct {
	Height  int  `json:"height"`
	Visible bool `json:"visible"`
}

type SearchConfig struct {
	UseRipgrep      bool     `json:"useRipgrep"`
	ExcludePatterns []string `json:"excludePatterns"`
}

type SyntaxConfig struct {
	Theme string `json:"theme"`
}

type LSPConfig struct {
	Command     string   `json:"command"`
	Args        []string `json:"args"`
	RootMarkers []string `json:"rootMarkers"`
}

// LoadDefaults returns a Config with hardcoded default values.
func LoadDefaults() (*Config, error) {
	return &Config{
		Editor: EditorConfig{
			TabSize:      4,
			InsertSpaces: true,
			WordWrap:     false,
			LineNumbers:  true,
			CursorStyle:  "block",
		},
		Sidebar: SidebarConfig{
			Width:   30,
			Visible: false,
		},
		Panel: PanelConfig{
			Height:  30,
			Visible: false,
		},
		Search: SearchConfig{
			UseRipgrep:      true,
			ExcludePatterns: []string{"node_modules", ".git", "__pycache__"},
		},
		Syntax: SyntaxConfig{
			Theme: "catppuccin-mocha",
		},
		LSP: map[string]LSPConfig{
			"go": {
				Command:     "gopls",
				Args:        []string{"serve"},
				RootMarkers: []string{"go.mod"},
			},
			"python": {
				Command:     "pylsp",
				Args:        []string{},
				RootMarkers: []string{"pyproject.toml", "setup.py"},
			},
		},
	}, nil
}

// MergeFromFile reads a JSON file and merges values into cfg.
// json.Unmarshal only overwrites fields present in the JSON,
// so unspecified fields retain their current values.
func MergeFromFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, cfg)
}

// DefaultUserConfigDir returns the ted config directory, respecting XDG_CONFIG_HOME.
func DefaultUserConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "ted")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ted")
}

// Load builds a Config by layering: defaults -> user config -> project config.
func Load(userConfigDir string, projectRoot string) (*Config, error) {
	cfg, err := LoadDefaults()
	if err != nil {
		return nil, err
	}
	if userConfigDir != "" {
		if err := MergeFromFile(cfg, filepath.Join(userConfigDir, "settings.json")); err != nil {
			return nil, err
		}
	}
	if projectRoot != "" {
		if err := MergeFromFile(cfg, filepath.Join(projectRoot, ".ted", "settings.json")); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}
