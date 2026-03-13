package editor

import (
	"testing"

	"github.com/seoji/ted/internal/buffer"
)

func TestTabManagerOpen(t *testing.T) {
	tm := NewTabManager()
	buf := buffer.NewBuffer("hello")
	idx := tm.Open(buf, "text")
	if idx != 0 {
		t.Errorf("expected index 0, got %d", idx)
	}
	if tm.Count() != 1 {
		t.Errorf("expected 1 tab, got %d", tm.Count())
	}
	if tm.Active().Buffer != buf {
		t.Error("active buffer mismatch")
	}
}

func TestTabManagerMultiple(t *testing.T) {
	tm := NewTabManager()
	buf1 := buffer.NewBuffer("one")
	buf2 := buffer.NewBuffer("two")
	tm.Open(buf1, "text")
	tm.Open(buf2, "text")
	if tm.Count() != 2 {
		t.Errorf("expected 2 tabs, got %d", tm.Count())
	}
	// Last opened is active
	if tm.ActiveIndex() != 1 {
		t.Errorf("expected active index 1, got %d", tm.ActiveIndex())
	}
}

func TestTabManagerClose(t *testing.T) {
	tm := NewTabManager()
	buf1 := buffer.NewBuffer("one")
	buf2 := buffer.NewBuffer("two")
	tm.Open(buf1, "text")
	tm.Open(buf2, "text")

	newIdx := tm.Close(1)
	if tm.Count() != 1 {
		t.Errorf("expected 1 tab, got %d", tm.Count())
	}
	if newIdx != 0 {
		t.Errorf("expected active index 0, got %d", newIdx)
	}
}

func TestTabManagerCloseAll(t *testing.T) {
	tm := NewTabManager()
	buf := buffer.NewBuffer("one")
	tm.Open(buf, "text")

	newIdx := tm.Close(0)
	if newIdx != -1 {
		t.Errorf("expected -1 after closing all, got %d", newIdx)
	}
	if tm.Active() != nil {
		t.Error("expected nil active tab")
	}
}

func TestTabManagerNextPrevious(t *testing.T) {
	tm := NewTabManager()
	tm.Open(buffer.NewBuffer("a"), "text")
	tm.Open(buffer.NewBuffer("b"), "text")
	tm.Open(buffer.NewBuffer("c"), "text")

	tm.SetActive(0)
	tm.Next()
	if tm.ActiveIndex() != 1 {
		t.Errorf("expected index 1, got %d", tm.ActiveIndex())
	}
	tm.Previous()
	if tm.ActiveIndex() != 0 {
		t.Errorf("expected index 0, got %d", tm.ActiveIndex())
	}
	// Wrap around
	tm.Previous()
	if tm.ActiveIndex() != 2 {
		t.Errorf("expected wraparound to 2, got %d", tm.ActiveIndex())
	}
}

func TestTabManagerFindByPath(t *testing.T) {
	tm := NewTabManager()
	buf := buffer.NewBuffer("content")
	buf.SetPath("/test/file.go")
	tm.Open(buf, "go")

	idx := tm.FindByPath("/test/file.go")
	if idx != 0 {
		t.Errorf("expected 0, got %d", idx)
	}

	idx = tm.FindByPath("/nonexistent")
	if idx != -1 {
		t.Errorf("expected -1, got %d", idx)
	}
}
