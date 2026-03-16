package buffer

// source identifies which buffer a piece references.
type source int

const (
	srcOriginal source = iota
	srcAdd
)

// piece represents a contiguous span of text in either the original or add buffer.
type piece struct {
	source source
	start  int // byte offset into the source buffer
	length int // number of bytes
}

// PieceTable implements the piece table data structure for efficient text editing.
type PieceTable struct {
	original ContentSource // immutable original content
	add      []byte        // append-only add buffer
	pieces   []piece       // ordered list of pieces
}

// NewPieceTable creates a new PieceTable initialized with the given ContentSource.
func NewPieceTable(content ContentSource) *PieceTable {
	pt := &PieceTable{
		original: content,
		add:      make([]byte, 0, 1024),
	}
	if content.Len() > 0 {
		pt.pieces = []piece{{source: srcOriginal, start: 0, length: content.Len()}}
	}
	return pt
}

// NewPieceTableFromString creates a new PieceTable from a string (convenience constructor).
func NewPieceTableFromString(s string) *PieceTable {
	return NewPieceTable(NewStringContent(s))
}

// Length returns the total byte length of the content.
func (pt *PieceTable) Length() int {
	total := 0
	for _, p := range pt.pieces {
		total += p.length
	}
	return total
}

// Text returns the full content as a string.
func (pt *PieceTable) Text() string {
	var b []byte
	for _, p := range pt.pieces {
		b = append(b, pt.pieceBytes(p)...)
	}
	return string(b)
}

// TextRange returns length bytes starting at offset.
func (pt *PieceTable) TextRange(offset, length int) string {
	var b []byte
	remaining := length
	pos := 0
	for _, p := range pt.pieces {
		if remaining <= 0 {
			break
		}
		pEnd := pos + p.length
		if pEnd <= offset {
			pos = pEnd
			continue
		}
		// Calculate overlap
		readStart := 0
		if offset > pos {
			readStart = offset - pos
		}
		readLen := p.length - readStart
		if readLen > remaining {
			readLen = remaining
		}
		data := pt.pieceBytes(p)
		b = append(b, data[readStart:readStart+readLen]...)
		remaining -= readLen
		pos = pEnd
	}
	return string(b)
}

// Insert inserts text at the given byte offset.
// Returns the edit operation for undo support.
func (pt *PieceTable) Insert(offset int, text string) Edit {
	addStart := len(pt.add)
	pt.add = append(pt.add, text...)
	newPiece := piece{source: srcAdd, start: addStart, length: len(text)}

	edit := Edit{
		Type:   EditInsert,
		Offset: offset,
		Text:   text,
	}

	if len(pt.pieces) == 0 {
		pt.pieces = []piece{newPiece}
		return edit
	}

	idx, within := pt.findPiece(offset)
	if within == 0 {
		// Insert before piece at idx
		pt.pieces = insertPiece(pt.pieces, idx, newPiece)
	} else if within == pt.pieces[idx].length {
		// Insert after piece at idx
		pt.pieces = insertPiece(pt.pieces, idx+1, newPiece)
	} else {
		// Split piece at idx
		old := pt.pieces[idx]
		left := piece{source: old.source, start: old.start, length: within}
		right := piece{source: old.source, start: old.start + within, length: old.length - within}
		replacement := []piece{left, newPiece, right}
		pt.pieces = replacePieces(pt.pieces, idx, 1, replacement)
	}
	return edit
}

// Delete removes length bytes starting at offset.
// Returns the edit operation for undo support.
func (pt *PieceTable) Delete(offset, length int) Edit {
	totalLen := pt.Length()
	if offset < 0 {
		offset = 0
	}
	if offset > totalLen {
		offset = totalLen
	}
	if length < 0 {
		length = 0
	}
	if offset+length > totalLen {
		length = totalLen - offset
	}

	deletedText := pt.TextRange(offset, length)
	edit := Edit{
		Type:   EditDelete,
		Offset: offset,
		Text:   deletedText,
	}

	if length == 0 {
		return edit
	}

	startIdx, startWithin := pt.findPiece(offset)
	endIdx, endWithin := pt.findPiece(offset + length)

	var newPieces []piece

	// Left remainder of first affected piece
	if startWithin > 0 {
		p := pt.pieces[startIdx]
		newPieces = append(newPieces, piece{source: p.source, start: p.start, length: startWithin})
	}

	// Right remainder of last affected piece
	if endIdx < len(pt.pieces) && endWithin > 0 {
		p := pt.pieces[endIdx]
		if endWithin < p.length {
			newPieces = append(newPieces, piece{source: p.source, start: p.start + endWithin, length: p.length - endWithin})
		}
	}

	// Replace affected pieces
	replaceEnd := endIdx
	if endIdx < len(pt.pieces) && endWithin > 0 {
		replaceEnd = endIdx + 1
	}
	pt.pieces = replacePieces(pt.pieces, startIdx, replaceEnd-startIdx, newPieces)

	return edit
}

// findPiece returns the piece index and offset within that piece for a given byte offset.
func (pt *PieceTable) findPiece(offset int) (idx int, within int) {
	pos := 0
	for i, p := range pt.pieces {
		if offset <= pos+p.length {
			return i, offset - pos
		}
		pos += p.length
	}
	return len(pt.pieces), 0
}

func (pt *PieceTable) pieceBytes(p piece) []byte {
	if p.source == srcOriginal {
		return pt.original.Slice(p.start, p.start+p.length)
	}
	return pt.add[p.start : p.start+p.length]
}

// Close releases resources held by the PieceTable's original content source.
func (pt *PieceTable) Close() error {
	if pt.original != nil {
		return pt.original.Close()
	}
	return nil
}

func insertPiece(pieces []piece, idx int, p piece) []piece {
	pieces = append(pieces, piece{})
	copy(pieces[idx+1:], pieces[idx:])
	pieces[idx] = p
	return pieces
}

func replacePieces(pieces []piece, idx, count int, replacement []piece) []piece {
	tail := append([]piece{}, pieces[idx+count:]...)
	pieces = append(pieces[:idx], replacement...)
	pieces = append(pieces, tail...)
	return pieces
}

// EditType represents the type of an edit operation.
type EditType int

const (
	EditInsert EditType = iota
	EditDelete
)

// Edit represents a single edit operation, used for undo/redo.
type Edit struct {
	Type   EditType
	Offset int
	Text   string
}
