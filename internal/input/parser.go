package input

import (
	"strings"

	"github.com/gdamore/tcell/v2"
)

// KeyEvent represents a parsed key combination.
type KeyEvent struct {
	Key  tcell.Key
	Rune rune
	Mod  tcell.ModMask
}

// ParseKeyString parses a key string like "ctrl+s" into a KeyEvent.
func ParseKeyString(s string) KeyEvent {
	s = strings.TrimSpace(strings.ToLower(s))
	parts := strings.Split(s, "+")

	var mod tcell.ModMask
	keyPart := ""

	for _, p := range parts {
		p = strings.TrimSpace(p)
		switch p {
		case "ctrl":
			mod |= tcell.ModCtrl
		case "shift":
			mod |= tcell.ModShift
		case "alt":
			mod |= tcell.ModAlt
		default:
			keyPart = p
		}
	}

	// Named keys
	if key, ok := namedKeys[keyPart]; ok {
		return KeyEvent{Key: key, Mod: mod}
	}

	// Single character
	if len(keyPart) == 1 {
		r := rune(keyPart[0])
		return KeyEvent{Key: tcell.KeyRune, Rune: r, Mod: mod}
	}

	return KeyEvent{Key: tcell.KeyRune, Mod: mod}
}

// Matches returns true if the KeyEvent matches a tcell.EventKey.
func (ke KeyEvent) Matches(ev *tcell.EventKey) bool {
	if ke.Key == tcell.KeyRune {
		return ev.Key() == tcell.KeyRune &&
			ev.Rune() == ke.Rune &&
			ev.Modifiers() == ke.Mod
	}
	return ev.Key() == ke.Key && ev.Modifiers() == ke.Mod
}

// String returns the key string representation.
func (ke KeyEvent) String() string {
	var parts []string
	if ke.Mod&tcell.ModCtrl != 0 {
		parts = append(parts, "ctrl")
	}
	if ke.Mod&tcell.ModShift != 0 {
		parts = append(parts, "shift")
	}
	if ke.Mod&tcell.ModAlt != 0 {
		parts = append(parts, "alt")
	}

	if ke.Key == tcell.KeyRune {
		parts = append(parts, string(ke.Rune))
	} else {
		for name, key := range namedKeys {
			if key == ke.Key {
				parts = append(parts, name)
				break
			}
		}
	}
	return strings.Join(parts, "+")
}

var namedKeys = map[string]tcell.Key{
	"enter":     tcell.KeyEnter,
	"esc":       tcell.KeyEscape,
	"escape":    tcell.KeyEscape,
	"tab":       tcell.KeyTab,
	"backspace": tcell.KeyBackspace2,
	"delete":    tcell.KeyDelete,
	"insert":    tcell.KeyInsert,
	"home":      tcell.KeyHome,
	"end":       tcell.KeyEnd,
	"pgup":      tcell.KeyPgUp,
	"pageup":    tcell.KeyPgUp,
	"pgdn":      tcell.KeyPgDn,
	"pagedown":  tcell.KeyPgDn,
	"up":        tcell.KeyUp,
	"down":      tcell.KeyDown,
	"left":      tcell.KeyLeft,
	"right":     tcell.KeyRight,
	"space":     tcell.KeyRune, // handled specially
	"f1":        tcell.KeyF1,
	"f2":        tcell.KeyF2,
	"f3":        tcell.KeyF3,
	"f4":        tcell.KeyF4,
	"f5":        tcell.KeyF5,
	"f6":        tcell.KeyF6,
	"f7":        tcell.KeyF7,
	"f8":        tcell.KeyF8,
	"f9":        tcell.KeyF9,
	"f10":       tcell.KeyF10,
	"f11":       tcell.KeyF11,
	"f12":       tcell.KeyF12,
}
