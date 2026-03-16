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
	pt := NewPieceTableFromString(content)
	b := &Buffer{
		pt:   pt,
		undo: NewUndoManager(pt),
	}
	b.rebuildLineIndex()
	return b
}

// LargeFileThreshold is the size above which files are opened via mmap.
const LargeFileThreshold = 10 * 1024 * 1024 // 10MB

// OpenFile creates a buffer by reading a file.
func OpenFile(path string) (*Buffer, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if info.Size() >= LargeFileThreshold {
		return openLargeFile(path)
	}

	// Small file: existing behavior
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

func openLargeFile(path string) (*Buffer, error) {
	mc, err := NewMmapContent(path)
	if err != nil {
		return nil, err
	}

	pt := NewPieceTable(mc)
	b := &Buffer{
		pt:   pt,
		undo: NewUndoManager(pt),
		path: path,
	}
	b.rebuildLineIndex()
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
	b.updateLineIndex(edit)
}

// Delete deletes length bytes at the given line and column. No-op if ReadOnly.
func (b *Buffer) Delete(line, col, length int) {
	if b.ReadOnly {
		return
	}
	offset := b.PositionToOffset(line, col)
	edit := b.pt.Delete(offset, length)
	b.undo.Execute(edit)
	b.updateLineIndex(edit)
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

// Close releases resources held by the buffer (e.g., mmap).
func (b *Buffer) Close() error {
	return b.pt.Close()
}

// IsLarge returns true if the buffer content is at or above the large file threshold.
func (b *Buffer) IsLarge() bool {
	return b.pt.Length() >= LargeFileThreshold
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

// updateLineIndex incrementally updates line offsets after an edit.
func (b *Buffer) updateLineIndex(edit Edit) {
	if edit.Text == "" {
		return
	}

	switch edit.Type {
	case EditInsert:
		b.updateLineIndexInsert(edit.Offset, edit.Text)
	case EditDelete:
		b.updateLineIndexDelete(edit.Offset, edit.Text)
	}
}

// updateLineIndexInsert updates line offsets after an insert operation.
func (b *Buffer) updateLineIndexInsert(offset int, text string) {
	// Find which line the offset falls in
	lineIdx := b.lineAtOffset(offset)

	// Count newlines in inserted text and their positions
	var newOffsets []int
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			newOffsets = append(newOffsets, offset+i+1)
		}
	}

	if len(newOffsets) == 0 {
		// No new lines added, just shift subsequent line offsets
		textLen := len(text)
		for i := lineIdx + 1; i < len(b.lineOffsets); i++ {
			b.lineOffsets[i] += textLen
		}
		return
	}

	// Insert new line offsets and shift existing ones
	textLen := len(text)
	insertAfter := lineIdx // insert new entries after this index

	// Shift all subsequent offsets
	for i := insertAfter + 1; i < len(b.lineOffsets); i++ {
		b.lineOffsets[i] += textLen
	}

	// Insert new line start offsets
	newLineOffsets := make([]int, 0, len(b.lineOffsets)+len(newOffsets))
	newLineOffsets = append(newLineOffsets, b.lineOffsets[:insertAfter+1]...)
	newLineOffsets = append(newLineOffsets, newOffsets...)
	newLineOffsets = append(newLineOffsets, b.lineOffsets[insertAfter+1:]...)
	b.lineOffsets = newLineOffsets
}

// updateLineIndexDelete updates line offsets after a delete operation.
func (b *Buffer) updateLineIndexDelete(offset int, deletedText string) {
	// Count newlines in deleted text
	newlineCount := 0
	for i := 0; i < len(deletedText); i++ {
		if deletedText[i] == '\n' {
			newlineCount++
		}
	}

	if newlineCount == 0 {
		// No lines removed, just shift subsequent line offsets
		textLen := len(deletedText)
		lineIdx := b.lineAtOffset(offset)
		for i := lineIdx + 1; i < len(b.lineOffsets); i++ {
			b.lineOffsets[i] -= textLen
		}
		return
	}

	// Find the line range that was affected
	lineIdx := b.lineAtOffset(offset)

	// Remove the merged lines and shift offsets
	removeStart := lineIdx + 1
	removeEnd := removeStart + newlineCount
	if removeEnd > len(b.lineOffsets) {
		removeEnd = len(b.lineOffsets)
	}

	textLen := len(deletedText)
	// Shift remaining offsets
	newLineOffsets := make([]int, 0, len(b.lineOffsets)-newlineCount)
	newLineOffsets = append(newLineOffsets, b.lineOffsets[:removeStart]...)
	for i := removeEnd; i < len(b.lineOffsets); i++ {
		newLineOffsets = append(newLineOffsets, b.lineOffsets[i]-textLen)
	}
	b.lineOffsets = newLineOffsets
}

// lineAtOffset returns the line index for a given byte offset.
func (b *Buffer) lineAtOffset(offset int) int {
	// Binary search for the line containing offset
	lo, hi := 0, len(b.lineOffsets)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if b.lineOffsets[mid] <= offset {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
}

// rebuildLineIndex recalculates line start offsets from the current content.
// It iterates over pieces directly to avoid materializing the entire content
// into a single string, which is important for large mmap-backed files.
func (b *Buffer) rebuildLineIndex() {
	b.lineOffsets = []int{0}
	pos := 0
	for _, p := range b.pt.pieces {
		data := b.pt.pieceBytes(p)
		for i, ch := range data {
			if ch == '\n' {
				b.lineOffsets = append(b.lineOffsets, pos+i+1)
			}
		}
		pos += p.length
	}
}
