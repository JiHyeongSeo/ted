package input

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func makeKeyEvent(key tcell.Key, r rune, mod tcell.ModMask) *tcell.EventKey {
	return tcell.NewEventKey(key, r, mod)
}

func TestKeymapBind(t *testing.T) {
	km := NewKeymap()
	km.Bind("ctrl+s", "file.save", "")
	if km.BindingCount() != 1 {
		t.Errorf("expected 1 binding, got %d", km.BindingCount())
	}
}

func TestKeymapResolve(t *testing.T) {
	km := NewKeymap()
	km.Bind("ctrl+s", "file.save", "")

	ev := makeKeyEvent(tcell.KeyRune, 's', tcell.ModCtrl)
	cmd, result := km.Resolve(ev, "")
	if result != ResolveMatched {
		t.Errorf("expected ResolveMatched, got %v", result)
	}
	if cmd != "file.save" {
		t.Errorf("expected 'file.save', got %q", cmd)
	}
}

func TestKeymapResolveNoMatch(t *testing.T) {
	km := NewKeymap()
	km.Bind("ctrl+s", "file.save", "")

	ev := makeKeyEvent(tcell.KeyRune, 'x', tcell.ModCtrl)
	_, result := km.Resolve(ev, "")
	if result != ResolveNone {
		t.Errorf("expected ResolveNone, got %v", result)
	}
}

func TestKeymapChord(t *testing.T) {
	km := NewKeymap()
	km.Bind("ctrl+k ctrl+i", "lsp.hover", "")

	// First key of chord
	ev1 := makeKeyEvent(tcell.KeyRune, 'k', tcell.ModCtrl)
	_, result := km.Resolve(ev1, "")
	if result != ResolvePending {
		t.Errorf("expected ResolvePending, got %v", result)
	}
	if !km.HasPendingChord() {
		t.Error("expected pending chord")
	}

	// Second key of chord
	ev2 := makeKeyEvent(tcell.KeyRune, 'i', tcell.ModCtrl)
	cmd, result := km.Resolve(ev2, "")
	if result != ResolveMatched {
		t.Errorf("expected ResolveMatched, got %v", result)
	}
	if cmd != "lsp.hover" {
		t.Errorf("expected 'lsp.hover', got %q", cmd)
	}
}

func TestKeymapChordMismatch(t *testing.T) {
	km := NewKeymap()
	km.Bind("ctrl+k ctrl+i", "lsp.hover", "")

	ev1 := makeKeyEvent(tcell.KeyRune, 'k', tcell.ModCtrl)
	km.Resolve(ev1, "")

	// Wrong second key
	ev2 := makeKeyEvent(tcell.KeyRune, 'x', tcell.ModCtrl)
	_, result := km.Resolve(ev2, "")
	if result != ResolveNone {
		t.Errorf("expected ResolveNone, got %v", result)
	}
	if km.HasPendingChord() {
		t.Error("chord should be cleared")
	}
}

func TestKeymapContext(t *testing.T) {
	km := NewKeymap()
	km.Bind("enter", "palette.select", "palette")
	km.Bind("enter", "editor.newline", "editor")

	ev := makeKeyEvent(tcell.KeyEnter, 0, tcell.ModNone)

	cmd, result := km.Resolve(ev, "palette")
	if result != ResolveMatched || cmd != "palette.select" {
		t.Errorf("expected palette.select, got %q (result=%v)", cmd, result)
	}

	cmd, result = km.Resolve(ev, "editor")
	if result != ResolveMatched || cmd != "editor.newline" {
		t.Errorf("expected editor.newline, got %q (result=%v)", cmd, result)
	}
}

func TestKeymapLoadFromJSON(t *testing.T) {
	km := NewKeymap()
	data := []byte(`{
		"keybindings": [
			{"key": "ctrl+s", "command": "file.save"},
			{"key": "ctrl+z", "command": "edit.undo"},
			{"key": "ctrl+k ctrl+i", "command": "lsp.hover"}
		]
	}`)

	err := km.LoadFromJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if km.BindingCount() != 3 {
		t.Errorf("expected 3 bindings, got %d", km.BindingCount())
	}
}

func TestKeymapClearChord(t *testing.T) {
	km := NewKeymap()
	km.Bind("ctrl+k ctrl+i", "lsp.hover", "")

	ev := makeKeyEvent(tcell.KeyRune, 'k', tcell.ModCtrl)
	km.Resolve(ev, "")
	if !km.HasPendingChord() {
		t.Error("expected pending chord")
	}
	km.ClearChord()
	if km.HasPendingChord() {
		t.Error("chord should be cleared")
	}
}

func TestKeymapF12(t *testing.T) {
	km := NewKeymap()
	km.Bind("f12", "lsp.goToDefinition", "")

	ev := makeKeyEvent(tcell.KeyF12, 0, tcell.ModNone)
	cmd, result := km.Resolve(ev, "")
	if result != ResolveMatched || cmd != "lsp.goToDefinition" {
		t.Errorf("expected lsp.goToDefinition, got %q (result=%v)", cmd, result)
	}
}
