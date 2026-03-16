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

func TestIncrementalInsert(t *testing.T) {
	b := NewBuffer("hello\nworld\nfoo")
	if b.LineCount() != 3 {
		t.Fatalf("expected 3 lines, got %d", b.LineCount())
	}

	// Insert newline in middle of first line
	b.Insert(0, 3, "\nnew")
	if b.LineCount() != 4 {
		t.Fatalf("expected 4 lines after insert, got %d", b.LineCount())
	}
	if b.Line(0) != "hel" {
		t.Errorf("line 0: got %q", b.Line(0))
	}
	if b.Line(1) != "newlo" {
		t.Errorf("line 1: got %q", b.Line(1))
	}
	if b.Line(2) != "world" {
		t.Errorf("line 2: got %q", b.Line(2))
	}
}

func TestIncrementalDelete(t *testing.T) {
	b := NewBuffer("hello\nworld\nfoo")
	// Delete "lo\nwor" (spanning newline)
	b.Delete(0, 3, 6)
	if b.LineCount() != 2 {
		t.Fatalf("expected 2 lines after delete, got %d", b.LineCount())
	}
	if b.Line(0) != "helld" {
		t.Errorf("line 0: got %q", b.Line(0))
	}
	if b.Line(1) != "foo" {
		t.Errorf("line 1: got %q", b.Line(1))
	}
}

func TestIncrementalMatchesFull(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		edits    func(b *Buffer)
		expected string
	}{
		{
			name:    "Insert without newlines",
			initial: "hello\nworld",
			edits: func(b *Buffer) {
				b.Insert(0, 5, " there")
			},
			expected: "hello there\nworld",
		},
		{
			name:    "Insert with multiple newlines",
			initial: "hello\nworld",
			edits: func(b *Buffer) {
				b.Insert(0, 5, "\nfoo\nbar")
			},
			expected: "hello\nfoo\nbar\nworld",
		},
		{
			name:    "Delete without newlines",
			initial: "hello world\ntest",
			edits: func(b *Buffer) {
				b.Delete(0, 5, 6) // delete " world"
			},
			expected: "hello\ntest",
		},
		{
			name:    "Delete with newlines",
			initial: "hello\nworld\ntest",
			edits: func(b *Buffer) {
				b.Delete(0, 5, 7) // delete "\nworld\n"
			},
			expected: "hellotest",
		},
		{
			name:    "Multiple edits",
			initial: "a\nb\nc",
			edits: func(b *Buffer) {
				b.Insert(0, 1, "1")
				b.Insert(1, 1, "2")
				b.Delete(2, 0, 1)
			},
			expected: "a1\nb2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with incremental updates
			b1 := NewBuffer(tt.initial)
			tt.edits(b1)
			result1 := b1.Text()
			lineCount1 := b1.LineCount()

			// Test with full rebuild (create fresh buffer, apply edits, rebuild)
			b2 := NewBuffer(tt.initial)
			tt.edits(b2)
			b2.rebuildLineIndex()
			result2 := b2.Text()
			lineCount2 := b2.LineCount()

			if result1 != tt.expected {
				t.Errorf("incremental: expected %q, got %q", tt.expected, result1)
			}
			if result2 != tt.expected {
				t.Errorf("full rebuild: expected %q, got %q", tt.expected, result2)
			}
			if result1 != result2 {
				t.Errorf("mismatch: incremental %q != full rebuild %q", result1, result2)
			}
			if lineCount1 != lineCount2 {
				t.Errorf("line count mismatch: incremental %d != full rebuild %d", lineCount1, lineCount2)
			}

			// Verify all lines match
			for i := 0; i < lineCount1; i++ {
				line1 := b1.Line(i)
				line2 := b2.Line(i)
				if line1 != line2 {
					t.Errorf("line %d mismatch: incremental %q != full rebuild %q", i, line1, line2)
				}
			}
		})
	}
}

func TestOpenLargeFile(t *testing.T) {
	f, err := os.CreateTemp("", "ted-large-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	line := "This is a test line for large file handling in ted editor.\n"
	for i := 0; i < 200000; i++ {
		f.WriteString(line)
	}
	f.Close()

	buf, err := OpenFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer buf.Close()

	// 200000 lines with newlines = 200001 line offsets (last empty line after final \n)
	expectedLines := 200001
	if buf.LineCount() != expectedLines {
		t.Errorf("expected %d lines, got %d", expectedLines, buf.LineCount())
	}

	first := buf.Line(0)
	expected := "This is a test line for large file handling in ted editor."
	if first != expected {
		t.Errorf("unexpected first line: %q", first)
	}
	last := buf.Line(199999)
	if last != expected {
		t.Errorf("unexpected last line: %q", last)
	}
}

// Benchmarks to demonstrate performance improvement

func BenchmarkIncrementalInsert(b *testing.B) {
	// Create a large buffer (simulating a 100KB file with ~2000 lines)
	content := ""
	for i := 0; i < 2000; i++ {
		content += "This is a line of text in the file\n"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := NewBuffer(content)
		// Insert text at the beginning (worst case for incremental)
		buf.Insert(0, 0, "x")
	}
}

func BenchmarkFullRebuildInsert(b *testing.B) {
	// Create a large buffer (simulating a 100KB file with ~2000 lines)
	content := ""
	for i := 0; i < 2000; i++ {
		content += "This is a line of text in the file\n"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := NewBuffer(content)
		offset := buf.PositionToOffset(0, 0)
		buf.pt.Insert(offset, "x")
		buf.rebuildLineIndex() // Force full rebuild
	}
}
