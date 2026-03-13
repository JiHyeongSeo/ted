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

// DefaultTheme returns a built-in Monokai theme.
func DefaultTheme() *Theme {
	return &Theme{
		Name: "Monokai",
		Colors: map[string]string{
			"keyword":     "#f92672", // pink
			"string":      "#e6db74", // yellow
			"function":    "#a6e22e", // green
			"comment":     "#75715e", // grey
			"type":        "#66d9ef", // cyan
			"number":      "#ae81ff", // purple
			"operator":    "#f92672", // pink
			"variable":    "#f8f8f2", // white
			"constant":    "#ae81ff", // purple
			"property":    "#66d9ef", // cyan
			"punctuation": "#f8f8f2", // white
		},
		UI: map[string]string{
			"background":          "#272822",
			"foreground":          "#f8f8f2",
			"cursor":              "#f8f8f0",
			"selection":           "#49483e",
			"lineNumber":          "#90908a",
			"lineNumberActive":    "#f8f8f2",
			"statusBar":           "#1e1f1c",
			"statusBarForeground": "#f8f8f2",
			"tabActive":           "#272822",
			"tabInactive":         "#1e1f1c",
			"sidebar":             "#1e1f1c",
			"panelBackground":     "#1e1f1c",
			"errorForeground":     "#f92672",
			"warningForeground":   "#e6db74",
			"matchHighlight":      "#e6db74",
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
