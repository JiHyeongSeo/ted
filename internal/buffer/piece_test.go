package buffer

import (
	"strings"
	"testing"
)

func TestNewPieceTable(t *testing.T) {
	pt := NewPieceTable("hello world")
	if got := pt.Text(); got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
	if pt.Length() != 11 {
		t.Errorf("expected length 11, got %d", pt.Length())
	}
}

func TestPieceTableEmpty(t *testing.T) {
	pt := NewPieceTable("")
	if got := pt.Text(); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
	if pt.Length() != 0 {
		t.Errorf("expected length 0, got %d", pt.Length())
	}
}

func TestInsertAtStart(t *testing.T) {
	pt := NewPieceTable("world")
	pt.Insert(0, "hello ")
	if got := pt.Text(); got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

func TestInsertAtEnd(t *testing.T) {
	pt := NewPieceTable("hello")
	pt.Insert(5, " world")
	if got := pt.Text(); got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

func TestInsertInMiddle(t *testing.T) {
	pt := NewPieceTable("helloworld")
	pt.Insert(5, " ")
	if got := pt.Text(); got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

func TestDeleteFromStart(t *testing.T) {
	pt := NewPieceTable("hello world")
	pt.Delete(0, 6)
	if got := pt.Text(); got != "world" {
		t.Errorf("expected 'world', got %q", got)
	}
}

func TestDeleteFromEnd(t *testing.T) {
	pt := NewPieceTable("hello world")
	pt.Delete(5, 6)
	if got := pt.Text(); got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestDeleteFromMiddle(t *testing.T) {
	pt := NewPieceTable("hello world")
	pt.Delete(5, 1)
	if got := pt.Text(); got != "helloworld" {
		t.Errorf("expected 'helloworld', got %q", got)
	}
}

func TestDeleteAll(t *testing.T) {
	pt := NewPieceTable("hello")
	pt.Delete(0, 5)
	if got := pt.Text(); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestMultipleInserts(t *testing.T) {
	pt := NewPieceTable("")
	pt.Insert(0, "c")
	pt.Insert(0, "a")
	pt.Insert(1, "b")
	if got := pt.Text(); got != "abc" {
		t.Errorf("expected 'abc', got %q", got)
	}
}

func TestInsertAndDelete(t *testing.T) {
	pt := NewPieceTable("ac")
	pt.Insert(1, "b")
	if got := pt.Text(); got != "abc" {
		t.Errorf("expected 'abc', got %q", got)
	}
	pt.Delete(1, 1)
	if got := pt.Text(); got != "ac" {
		t.Errorf("expected 'ac', got %q", got)
	}
}

func TestDeleteAcrossPieces(t *testing.T) {
	pt := NewPieceTable("hello")
	pt.Insert(5, " world")
	// Now pieces: ["hello", " world"]
	// Delete "lo wo" (offset 3, length 5) which spans both pieces
	pt.Delete(3, 5)
	if got := pt.Text(); got != "helrld" {
		t.Errorf("expected 'helrld', got %q", got)
	}
}

func TestLargeContent(t *testing.T) {
	content := strings.Repeat("abcdefghij\n", 10000) // 110KB
	pt := NewPieceTable(content)
	if pt.Length() != len(content) {
		t.Errorf("expected length %d, got %d", len(content), pt.Length())
	}
	// Insert in the middle
	mid := pt.Length() / 2
	pt.Insert(mid, "INSERTED")
	expected := content[:mid] + "INSERTED" + content[mid:]
	if got := pt.Text(); got != expected {
		t.Error("large content insert mismatch")
	}
}

func TestTextRange(t *testing.T) {
	pt := NewPieceTable("hello world")
	got := pt.TextRange(6, 5)
	if got != "world" {
		t.Errorf("expected 'world', got %q", got)
	}
}

func TestTextRangeAcrossPieces(t *testing.T) {
	pt := NewPieceTable("hello")
	pt.Insert(5, " world")
	got := pt.TextRange(3, 5)
	if got != "lo wo" {
		t.Errorf("expected 'lo wo', got %q", got)
	}
}
