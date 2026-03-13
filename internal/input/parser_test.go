package input

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestParseCtrlS(t *testing.T) {
	ke := ParseKeyString("ctrl+s")
	if ke.Mod&tcell.ModCtrl == 0 {
		t.Error("expected Ctrl modifier")
	}
	if ke.Rune != 's' {
		t.Errorf("expected rune 's', got %q", ke.Rune)
	}
}

func TestParseF12(t *testing.T) {
	ke := ParseKeyString("f12")
	if ke.Key != tcell.KeyF12 {
		t.Errorf("expected F12 key, got %v", ke.Key)
	}
}

func TestParseShiftF12(t *testing.T) {
	ke := ParseKeyString("shift+f12")
	if ke.Key != tcell.KeyF12 {
		t.Errorf("expected F12 key, got %v", ke.Key)
	}
	if ke.Mod&tcell.ModShift == 0 {
		t.Error("expected Shift modifier")
	}
}

func TestParseCtrlShiftF(t *testing.T) {
	ke := ParseKeyString("ctrl+shift+f")
	if ke.Mod&tcell.ModCtrl == 0 {
		t.Error("expected Ctrl modifier")
	}
	if ke.Mod&tcell.ModShift == 0 {
		t.Error("expected Shift modifier")
	}
	if ke.Rune != 'f' {
		t.Errorf("expected rune 'f', got %q", ke.Rune)
	}
}

func TestParseEnter(t *testing.T) {
	ke := ParseKeyString("enter")
	if ke.Key != tcell.KeyEnter {
		t.Errorf("expected Enter key, got %v", ke.Key)
	}
}

func TestParseEscape(t *testing.T) {
	ke := ParseKeyString("esc")
	if ke.Key != tcell.KeyEscape {
		t.Errorf("expected Escape key, got %v", ke.Key)
	}
}

func TestParseCaseInsensitive(t *testing.T) {
	ke := ParseKeyString("Ctrl+S")
	if ke.Mod&tcell.ModCtrl == 0 {
		t.Error("expected Ctrl modifier")
	}
	if ke.Rune != 's' {
		t.Errorf("expected rune 's', got %q", ke.Rune)
	}
}

func TestKeyEventString(t *testing.T) {
	ke := ParseKeyString("ctrl+s")
	s := ke.String()
	if s != "ctrl+s" {
		t.Errorf("expected 'ctrl+s', got %q", s)
	}
}
