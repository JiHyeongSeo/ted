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

// DefaultTheme returns a built-in Tango Dark theme.
func DefaultTheme() *Theme {
	return &Theme{
		Name: "Tango Dark",
		Colors: map[string]string{
			"keyword":     "#ad7fa8", // Tango plum
			"string":      "#73d216", // Tango chameleon
			"function":    "#729fcf", // Tango sky blue
			"comment":     "#888a85", // Tango aluminium 4
			"type":        "#fce94f", // Tango butter
			"number":      "#fcaf3e", // Tango orange
			"operator":    "#34e2e2", // Tango cyan
			"variable":    "#d3d7cf", // Tango aluminium 2
			"constant":    "#fcaf3e", // Tango orange
			"property":    "#729fcf", // Tango sky blue
			"punctuation": "#babdb6", // Tango aluminium 3
		},
		UI: map[string]string{
			"background":          "#2e3436", // Tango aluminium 7
			"foreground":          "#d3d7cf", // Tango aluminium 2
			"cursor":              "#eeeeec", // Tango aluminium 1
			"selection":           "#4e585d", // slightly lighter bg
			"lineNumber":          "#555753", // Tango aluminium 5
			"lineNumberActive":    "#babdb6", // Tango aluminium 3
			"statusBar":           "#1c1f20", // darker than bg
			"statusBarForeground": "#d3d7cf",
			"tabActive":           "#2e3436", // same as bg
			"tabInactive":         "#1c1f20", // darker
			"sidebar":             "#1c1f20",
			"panelBackground":     "#1c1f20",
			"errorForeground":     "#ef2929", // Tango scarlet red
			"warningForeground":   "#fce94f", // Tango butter
			"matchHighlight":      "#fce94f",
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
