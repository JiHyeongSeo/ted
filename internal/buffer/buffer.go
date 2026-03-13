package buffer

import (
	"os"
	"unicode/utf8"
)

// Buffer wraps a PieceTable with line indexing, file I/O, and undo/redo.
type Buffer struct {
	pt          *PieceTable
	undo        *UndoManager
	path        string
	ReadOnly    bool  // true for non-UTF-8 (binary) files
	lineOffsets []int // byte offset of each line start
}

// NewBuffer creates a buffer from a string.
func NewBuffer(content string) *Buffer {
	pt := NewPieceTable(content)
	b := &Buffer{
		pt:   pt,
		undo: NewUndoManager(pt),
	}
	b.rebuildLineIndex()
	return b
}

// OpenFile creates a buffer by reading a file.
func OpenFile(path string) (*Buffer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	b := NewBuffer(string(data))
	b.path = path
	if !utf8.Valid(data) {
		b.ReadOnly = true
	}
	b.undo.MarkSaved()
	return b, nil
}

// Text returns the full buffer content.
func (b *Buffer) Text() string {
	return b.pt.Text()
}

// LineCount returns the number of lines.
func (b *Buffer) LineCount() int {
	return len(b.lineOffsets)
}

// Line returns the content of line n (0-based), without the trailing newline.
func (b *Buffer) Line(n int) string {
	if n < 0 || n >= len(b.lineOffsets) {
		return ""
	}
	start := b.lineOffsets[n]
	var end int
	if n+1 < len(b.lineOffsets) {
		end = b.lineOffsets[n+1] - 1 // exclude \n
	} else {
		end = b.pt.Length()
	}
	if end < start {
		end = start
	}
	return b.pt.TextRange(start, end-start)
}

// LineOffset returns the byte offset of the start of line n.
func (b *Buffer) LineOffset(n int) int {
	if n < 0 || n >= len(b.lineOffsets) {
		return b.pt.Length()
	}
	return b.lineOffsets[n]
}

// PositionToOffset converts a (line, col) to a byte offset.
func (b *Buffer) PositionToOffset(line, col int) int {
	return b.LineOffset(line) + col
}

// Insert inserts text at the given line and column. No-op if ReadOnly.
func (b *Buffer) Insert(line, col int, text string) {
	if b.ReadOnly {
		return
	}
	offset := b.PositionToOffset(line, col)
	edit := b.pt.Insert(offset, text)
	b.undo.Execute(edit)
	b.rebuildLineIndex()
}

// Delete deletes length bytes at the given line and column. No-op if ReadOnly.
func (b *Buffer) Delete(line, col, length int) {
	if b.ReadOnly {
		return
	}
	offset := b.PositionToOffset(line, col)
	edit := b.pt.Delete(offset, length)
	b.undo.Execute(edit)
	b.rebuildLineIndex()
}

// Undo reverses the last edit.
func (b *Buffer) Undo() {
	b.undo.Undo()
	b.rebuildLineIndex()
}

// Redo re-applies the last undone edit.
func (b *Buffer) Redo() {
	b.undo.Redo()
	b.rebuildLineIndex()
}

// IsDirty returns whether the buffer has unsaved changes.
func (b *Buffer) IsDirty() bool {
	return b.undo.IsDirty()
}

// Path returns the file path of the buffer.
func (b *Buffer) Path() string {
	return b.path
}

// SetPath sets the file path for the buffer.
func (b *Buffer) SetPath(path string) {
	b.path = path
}

// Save writes the buffer content to its file path.
func (b *Buffer) Save() error {
	if b.path == "" {
		return os.ErrInvalid
	}
	err := os.WriteFile(b.path, []byte(b.pt.Text()), 0644)
	if err != nil {
		return err
	}
	b.undo.MarkSaved()
	return nil
}

// rebuildLineIndex recalculates line start offsets from the current content.
func (b *Buffer) rebuildLineIndex() {
	text := b.pt.Text()
	b.lineOffsets = []int{0}
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			b.lineOffsets = append(b.lineOffsets, i+1)
		}
	}
}
