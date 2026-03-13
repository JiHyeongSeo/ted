package types

// Position represents a cursor position in a buffer.
type Position struct {
	Line int // 0-based line index
	Col  int // 0-based column index (byte offset within line)
}

// Selection represents a text selection range.
type Selection struct {
	Start Position
	End   Position
}

// Rect represents a rectangular screen region.
type Rect struct {
	X, Y, Width, Height int
}
