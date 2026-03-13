package syntax

import (
	"encoding/json"
	"os"
	"strconv"

	"github.com/gdamore/tcell/v2"
)

// Theme defines colors for syntax highlighting and UI elements.
type Theme struct {
	Name   string            `json:"name"`
	Colors map[string]string `json:"colors"`
	UI     map[string]string `json:"ui"`
}

// LoadTheme loads a theme from a JSON file.
func LoadTheme(path string) (*Theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseTheme(data)
}

// ParseTheme parses a theme from JSON bytes.
func ParseTheme(data []byte) (*Theme, error) {
	t := &Theme{}
	if err := json.Unmarshal(data, t); err != nil {
		return nil, err
	}
	return t, nil
}

// DefaultTheme returns a built-in dark theme.
func DefaultTheme() *Theme {
	return &Theme{
		Name: "Default Dark",
		Colors: map[string]string{
			"keyword":     "#cba6f7",
			"string":      "#a6e3a1",
			"function":    "#89b4fa",
			"comment":     "#585b70",
			"type":        "#f9e2af",
			"number":      "#fab387",
			"operator":    "#89dceb",
			"variable":    "#cdd6f4",
			"constant":    "#fab387",
			"property":    "#89b4fa",
			"punctuation": "#bac2de",
		},
		UI: map[string]string{
			"background":          "#1e1e2e",
			"foreground":          "#cdd6f4",
			"cursor":              "#f5e0dc",
			"selection":           "#45475a",
			"lineNumber":          "#585b70",
			"lineNumberActive":    "#cdd6f4",
			"statusBar":           "#181825",
			"statusBarForeground": "#cdd6f4",
			"tabActive":           "#1e1e2e",
			"tabInactive":         "#181825",
			"sidebar":             "#181825",
			"panelBackground":     "#181825",
			"errorForeground":     "#f38ba8",
			"warningForeground":   "#f9e2af",
			"matchHighlight":      "#f9e2af",
		},
	}
}

// TokenStyle returns the tcell.Style for a given syntax token type.
func (t *Theme) TokenStyle(tokenType string) tcell.Style {
	fg := t.ResolveColor(t.Colors[tokenType])
	bg := t.ResolveColor(t.UI["background"])
	return tcell.StyleDefault.Foreground(fg).Background(bg)
}

// UIStyle returns the tcell.Style for a given UI element.
func (t *Theme) UIStyle(element string) tcell.Style {
	var fg, bg tcell.Color

	switch element {
	case "statusbar":
		bg = t.ResolveColor(t.UI["statusBar"])
		fg = t.ResolveColor(t.UI["statusBarForeground"])
	case "tabbar.active":
		bg = t.ResolveColor(t.UI["tabActive"])
		fg = t.ResolveColor(t.UI["foreground"])
	case "tabbar.inactive":
		bg = t.ResolveColor(t.UI["tabInactive"])
		fg = t.ResolveColor(t.UI["foreground"])
	case "sidebar":
		bg = t.ResolveColor(t.UI["sidebar"])
		fg = t.ResolveColor(t.UI["foreground"])
	case "panel":
		bg = t.ResolveColor(t.UI["panelBackground"])
		fg = t.ResolveColor(t.UI["foreground"])
	case "linenumber":
		bg = t.ResolveColor(t.UI["background"])
		fg = t.ResolveColor(t.UI["lineNumber"])
	case "linenumber.active":
		bg = t.ResolveColor(t.UI["background"])
		fg = t.ResolveColor(t.UI["lineNumberActive"])
	case "selection":
		bg = t.ResolveColor(t.UI["selection"])
		fg = t.ResolveColor(t.UI["foreground"])
	case "error":
		bg = t.ResolveColor(t.UI["background"])
		fg = t.ResolveColor(t.UI["errorForeground"])
	case "warning":
		bg = t.ResolveColor(t.UI["background"])
		fg = t.ResolveColor(t.UI["warningForeground"])
	default:
		bg = t.ResolveColor(t.UI["background"])
		fg = t.ResolveColor(t.UI["foreground"])
	}

	return tcell.StyleDefault.Foreground(fg).Background(bg)
}

// ResolveColor converts a hex color string (#rrggbb) to tcell.Color.
func (t *Theme) ResolveColor(hex string) tcell.Color {
	if hex == "" {
		return tcell.ColorDefault
	}
	if hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return tcell.ColorDefault
	}

	r, err := strconv.ParseInt(hex[0:2], 16, 32)
	if err != nil {
		return tcell.ColorDefault
	}
	g, err := strconv.ParseInt(hex[2:4], 16, 32)
	if err != nil {
		return tcell.ColorDefault
	}
	b, err := strconv.ParseInt(hex[4:6], 16, 32)
	if err != nil {
		return tcell.ColorDefault
	}

	return tcell.NewRGBColor(int32(r), int32(g), int32(b))
}
