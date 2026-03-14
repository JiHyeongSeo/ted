package input

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestCtrlCResolve(t *testing.T) {
	km := NewKeymap()
	km.Bind("ctrl+c", "edit.copy", "")
	km.Bind("ctrl+v", "edit.paste", "")
	km.Bind("ctrl+x", "edit.cut", "")
	km.Bind("ctrl+s", "file.save", "")
	km.Bind("ctrl+z", "edit.undo", "")

	t.Logf("tcell.KeyCtrlA = %d (0x%x)", tcell.KeyCtrlA, int(tcell.KeyCtrlA))
	t.Logf("tcell.KeyCtrlC = %d (0x%x)", tcell.KeyCtrlC, int(tcell.KeyCtrlC))
	t.Logf("tcell.KeyCtrlV = %d (0x%x)", tcell.KeyCtrlV, int(tcell.KeyCtrlV))
	t.Logf("tcell.KeyCtrlX = %d (0x%x)", tcell.KeyCtrlX, int(tcell.KeyCtrlX))
	t.Logf("tcell.KeyRune = %d (0x%x)", tcell.KeyRune, int(tcell.KeyRune))

	// Use tcell.NewEventKey to create realistic events
	tests := []struct {
		name    string
		key     tcell.Key
		rune_   rune
		mod     tcell.ModMask
		wantCmd string
	}{
		{"Ctrl+C (key=KeyCtrlC, rune=0, mod=ModCtrl)", tcell.KeyCtrlC, 0, tcell.ModCtrl, "edit.copy"},
		{"Ctrl+C (key=KeyCtrlC, rune='c', mod=ModCtrl)", tcell.KeyCtrlC, 'c', tcell.ModCtrl, "edit.copy"},
		{"Ctrl+C (key=KeyCtrlC, rune=0, mod=0)", tcell.KeyCtrlC, 0, 0, "edit.copy"},
		{"Ctrl+V (key=KeyCtrlV, rune=0, mod=ModCtrl)", tcell.KeyCtrlV, 0, tcell.ModCtrl, "edit.paste"},
		{"Ctrl+X (key=KeyCtrlX, rune=0, mod=ModCtrl)", tcell.KeyCtrlX, 0, tcell.ModCtrl, "edit.cut"},
		{"Ctrl+S (key=KeyCtrlS, rune=0, mod=ModCtrl)", tcell.KeyCtrlS, 0, tcell.ModCtrl, "file.save"},
		{"Ctrl+Z (key=KeyCtrlZ, rune=0, mod=ModCtrl)", tcell.KeyCtrlZ, 0, tcell.ModCtrl, "edit.undo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			km.ClearChord()
			ev := tcell.NewEventKey(tt.key, tt.rune_, tt.mod)
			t.Logf("EventKey: Key=%d(0x%x) Rune=%d('%c') Mod=%d Name=%s",
				ev.Key(), int(ev.Key()), ev.Rune(), ev.Rune(), ev.Modifiers(), ev.Name())

			// Trace eventToKeyEvent
			ke := eventToKeyEvent(ev)
			t.Logf("eventToKeyEvent: Key=%d(0x%x) Rune=%d('%c') Mod=%d",
				ke.Key, int(ke.Key), ke.Rune, ke.Rune, ke.Mod)

			cmd, result := km.Resolve(ev, "editor")
			t.Logf("Resolve: cmd=%q result=%d", cmd, result)

			if cmd != tt.wantCmd {
				t.Errorf("got cmd=%q, want %q", cmd, tt.wantCmd)
			}
			if result != ResolveMatched {
				t.Errorf("got result=%d, want ResolveMatched(%d)", result, ResolveMatched)
			}
		})
	}
}
