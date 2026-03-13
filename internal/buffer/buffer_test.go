package buffer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBufferFromString(t *testing.T) {
	buf := NewBuffer("hello\nworld\n")
	if buf.LineCount() != 3 {
		t.Errorf("expected 3 lines, got %d", buf.LineCount())
	}
	if got := buf.Line(0); got != "hello" {
		t.Errorf("line 0: expected 'hello', got %q", got)
	}
	if got := buf.Line(1); got != "world" {
		t.Errorf("line 1: expected 'world', got %q", got)
	}
	if got := buf.Line(2); got != "" {
		t.Errorf("line 2: expected '', got %q", got)
	}
}

func TestBufferInsert(t *testing.T) {
	buf := NewBuffer("hello\nworld")
	buf.Insert(0, 5, " there")
	if got := buf.Line(0); got != "hello there" {
		t.Errorf("expected 'hello there', got %q", got)
	}
}

func TestBufferInsertNewline(t *testing.T) {
	buf := NewBuffer("hello world")
	buf.Insert(0, 5, "\n")
	if buf.LineCount() != 2 {
		t.Errorf("expected 2 lines, got %d", buf.LineCount())
	}
	if got := buf.Line(0); got != "hello" {
		t.Errorf("line 0: expected 'hello', got %q", got)
	}
	if got := buf.Line(1); got != " world" {
		t.Errorf("line 1: expected ' world', got %q", got)
	}
}

func TestBufferDelete(t *testing.T) {
	buf := NewBuffer("hello world")
	buf.Delete(0, 5, 1) // delete space
	if got := buf.Line(0); got != "helloworld" {
		t.Errorf("expected 'helloworld', got %q", got)
	}
}

func TestBufferUndoRedo(t *testing.T) {
	buf := NewBuffer("hello")
	buf.Insert(0, 5, " world")
	buf.Undo()
	if got := buf.Text(); got != "hello" {
		t.Errorf("after undo: expected 'hello', got %q", got)
	}
	buf.Redo()
	if got := buf.Text(); got != "hello world" {
		t.Errorf("after redo: expected 'hello world', got %q", got)
	}
}

func TestBufferSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	buf := NewBuffer("hello world")
	buf.SetPath(path)
	if err := buf.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "hello world" {
		t.Errorf("saved content: expected 'hello world', got %q", string(data))
	}

	if buf.IsDirty() {
		t.Error("should not be dirty after save")
	}
}

func TestBufferLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("file content\nsecond line"), 0644)

	buf, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	if buf.LineCount() != 2 {
		t.Errorf("expected 2 lines, got %d", buf.LineCount())
	}
	if got := buf.Line(0); got != "file content" {
		t.Errorf("line 0: expected 'file content', got %q", got)
	}
	if buf.Path() != path {
		t.Errorf("expected path %s, got %s", path, buf.Path())
	}
}

func TestBufferNonUTF8ReadOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "binary.dat")
	// Write invalid UTF-8 bytes
	os.WriteFile(path, []byte{0xff, 0xfe, 0x00, 0x01}, 0644)

	buf, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	if !buf.ReadOnly {
		t.Error("expected non-UTF-8 file to be read-only")
	}
}

func TestBufferReadOnlyPreventsEdits(t *testing.T) {
	buf := NewBuffer("hello")
	buf.ReadOnly = true
	buf.Insert(0, 5, " world")
	if got := buf.Text(); got != "hello" {
		t.Errorf("expected read-only buffer unchanged, got %q", got)
	}
}

func TestBufferLineOffset(t *testing.T) {
	buf := NewBuffer("abc\ndef\nghi")
	// "abc\n" = 4 bytes, "def\n" = 4 bytes
	if off := buf.LineOffset(0); off != 0 {
		t.Errorf("line 0 offset: expected 0, got %d", off)
	}
	if off := buf.LineOffset(1); off != 4 {
		t.Errorf("line 1 offset: expected 4, got %d", off)
	}
	if off := buf.LineOffset(2); off != 8 {
		t.Errorf("line 2 offset: expected 8, got %d", off)
	}
}

func TestBufferPositionToOffset(t *testing.T) {
	buf := NewBuffer("abc\ndef\nghi")
	if off := buf.PositionToOffset(1, 2); off != 6 {
		t.Errorf("(1,2) offset: expected 6, got %d", off)
	}
}
