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

// DefaultTheme returns a built-in Dark+ theme (VSCode-inspired, pure black background).
func DefaultTheme() *Theme {
	return &Theme{
		Name: "Dark+",
		Colors: map[string]string{
			"keyword":     "#569cd6", // blue
			"string":      "#ce9178", // orange
			"function":    "#dcdcaa", // yellow
			"comment":     "#6a9955", // green
			"type":        "#4ec9b0", // teal
			"number":      "#b5cea8", // light green
			"operator":    "#d4d4d4", // light grey
			"variable":    "#9cdcfe", // light blue
			"constant":    "#4fc1ff", // bright blue
			"property":    "#9cdcfe", // light blue
			"punctuation": "#d4d4d4", // light grey
		},
		UI: map[string]string{
			"background":          "#000000",
			"foreground":          "#d4d4d4",
			"cursor":              "#aeafad",
			"selection":           "#264f78",
			"lineNumber":          "#5a5a5a",
			"lineNumberActive":    "#c6c6c6",
			"statusBar":           "#007acc",
			"statusBarForeground": "#ffffff",
			"tabActive":           "#000000",
			"tabInactive":         "#0a0a0a",
			"sidebar":             "#0a0a0a",
			"panelBackground":     "#0a0a0a",
			"errorForeground":     "#f44747",
			"warningForeground":   "#cca700",
			"matchHighlight":      "#e2c56d",
			"gitAdded":            "#254525",
			"gitModified":         "#153050",
			"gitDeleted":          "#501515",
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
