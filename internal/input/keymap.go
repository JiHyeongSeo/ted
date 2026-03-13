package input

import (
	"encoding/json"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

// Binding maps a key combination to a command.
type Binding struct {
	Keys    []KeyEvent // single key or chord (e.g., ctrl+k ctrl+i)
	Command string
	Context string // optional context restriction (e.g., "editor", "palette")
}

// Keymap manages key bindings and resolves key events to commands.
type Keymap struct {
	bindings     []Binding
	chordTimeout time.Duration
	pendingChord []KeyEvent
	chordTimer   *time.Timer
}

// NewKeymap creates a new Keymap.
func NewKeymap() *Keymap {
	return &Keymap{
		chordTimeout: 500 * time.Millisecond,
	}
}

// Bind adds a key binding.
func (km *Keymap) Bind(keyStr string, command string, context string) {
	parts := strings.Fields(keyStr) // split chord parts by space
	var keys []KeyEvent
	for _, part := range parts {
		keys = append(keys, ParseKeyString(part))
	}
	km.bindings = append(km.bindings, Binding{
		Keys:    keys,
		Command: command,
		Context: context,
	})
}

// keybindingEntry is the JSON format for keybindings.
type keybindingEntry struct {
	Key     string `json:"key"`
	Command string `json:"command"`
	Context string `json:"context,omitempty"`
}

// LoadFromFile loads keybindings from a JSON file.
func (km *Keymap) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return km.LoadFromJSON(data)
}

// LoadFromJSON loads keybindings from JSON bytes.
func (km *Keymap) LoadFromJSON(data []byte) error {
	var wrapper struct {
		Keybindings []keybindingEntry `json:"keybindings"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return err
	}
	for _, entry := range wrapper.Keybindings {
		km.Bind(entry.Key, entry.Command, entry.Context)
	}
	return nil
}

// Resolve attempts to match a key event to a command.
// Returns the command name and whether a match was found.
// For chords, returns ("", false) while waiting for more keys.
type ResolveResult int

const (
	ResolveNone    ResolveResult = iota // no match
	ResolveMatched                       // full match found
	ResolvePending                       // waiting for chord completion
)

// Resolve processes a key event and returns the result.
func (km *Keymap) Resolve(ev *tcell.EventKey, context string) (command string, result ResolveResult) {
	currentKeys := append(km.pendingChord, eventToKeyEvent(ev))

	// Check for exact matches
	for _, b := range km.bindings {
		if b.Context != "" && b.Context != context {
			continue
		}
		if len(b.Keys) == len(currentKeys) && keysMatch(b.Keys, currentKeys) {
			km.pendingChord = nil
			return b.Command, ResolveMatched
		}
	}

	// Check for partial chord matches
	for _, b := range km.bindings {
		if b.Context != "" && b.Context != context {
			continue
		}
		if len(b.Keys) > len(currentKeys) && keysMatch(b.Keys[:len(currentKeys)], currentKeys) {
			km.pendingChord = currentKeys
			return "", ResolvePending
		}
	}

	// No match
	km.pendingChord = nil
	return "", ResolveNone
}

// ClearChord resets any pending chord state.
func (km *Keymap) ClearChord() {
	km.pendingChord = nil
}

// HasPendingChord returns true if waiting for a chord to complete.
func (km *Keymap) HasPendingChord() bool {
	return len(km.pendingChord) > 0
}

// BindingCount returns the number of registered bindings.
func (km *Keymap) BindingCount() int {
	return len(km.bindings)
}

// Bindings returns all registered bindings.
func (km *Keymap) Bindings() []Binding {
	return km.bindings
}

func eventToKeyEvent(ev *tcell.EventKey) KeyEvent {
	key := ev.Key()
	r := ev.Rune()
	mod := ev.Modifiers()

	// tcell normalizes Ctrl+letter to key codes (e.g., Ctrl+S -> Key=83).
	// Normalize back to KeyRune + rune + ModCtrl to match our parsed format.
	if mod&tcell.ModCtrl != 0 && key != tcell.KeyRune && r != 0 {
		// Normalize rune to lowercase for consistent matching
		r = unicode.ToLower(r)
		return KeyEvent{Key: tcell.KeyRune, Rune: r, Mod: mod}
	}

	// For Shift+letter combos, tcell sends uppercase rune.
	// Normalize to lowercase and keep Shift modifier.
	if mod&tcell.ModShift != 0 && key == tcell.KeyRune && unicode.IsUpper(r) {
		r = unicode.ToLower(r)
		return KeyEvent{Key: tcell.KeyRune, Rune: r, Mod: mod}
	}

	return KeyEvent{Key: key, Rune: r, Mod: mod}
}

func keysMatch(a, b []KeyEvent) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Key != b[i].Key || a[i].Rune != b[i].Rune || a[i].Mod != b[i].Mod {
			return false
		}
	}
	return true
}
