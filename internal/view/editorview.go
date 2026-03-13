package view

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/seoji/ted/internal/buffer"
	"github.com/seoji/ted/internal/syntax"
	"github.com/seoji/ted/internal/types"
)

// EditorView renders a text buffer with line numbers, cursor, and scrolling.
type EditorView struct {
	BaseComponent
	buf          *buffer.Buffer
	theme        *syntax.Theme
	cursor       types.Position   // cursor position in the buffer
	scrollY      int              // first visible line
	scrollX      int              // horizontal scroll offset
	selection    *types.Selection // current selection range
	lineNumWidth int              // width of line number gutter
}

// NewEditorView creates an EditorView for the given buffer.
func NewEditorView(buf *buffer.Buffer, theme *syntax.Theme) *EditorView {
	ev := &EditorView{
		buf:   buf,
		theme: theme,
	}
	ev.updateLineNumWidth()
	return ev
}

// updateLineNumWidth calculates the width needed for line numbers.
func (e *EditorView) updateLineNumWidth() {
	count := e.buf.LineCount()
	if count == 0 {
		e.lineNumWidth = 2
		return
	}
	width := 1
	for count >= 10 {
		width++
		count /= 10
	}
	e.lineNumWidth = width + 1 // +1 for padding
}

// Render draws the editor view to the screen.
func (e *EditorView) Render(screen tcell.Screen) {
	e.updateLineNumWidth()
	bounds := e.Bounds()

	// Calculate visible area
	textAreaX := bounds.X + e.lineNumWidth
	textAreaWidth := bounds.Width - e.lineNumWidth
	if textAreaWidth < 0 {
		textAreaWidth = 0
	}

	// Draw each visible line
	for row := 0; row < bounds.Height; row++ {
		lineNum := e.scrollY + row
		if lineNum >= e.buf.LineCount() {
			// Clear remaining rows
			e.clearRow(screen, bounds.Y+row, bounds.X, bounds.Width)
			continue
		}

		// Draw line number
		lineNumStyle := e.theme.UIStyle("linenumber")
		if lineNum == e.cursor.Line {
			lineNumStyle = e.theme.UIStyle("linenumber.active")
		}
		lineNumStr := fmt.Sprintf("%*d", e.lineNumWidth-1, lineNum+1)
		for i, ch := range lineNumStr {
			screen.SetContent(bounds.X+i, bounds.Y+row, ch, nil, lineNumStyle)
		}
		// Padding space after line number
		screen.SetContent(bounds.X+e.lineNumWidth-1, bounds.Y+row, ' ', nil, lineNumStyle)

		// Draw line content
		lineText := e.buf.Line(lineNum)
		textStyle := e.theme.UIStyle("default")

		for col := 0; col < textAreaWidth; col++ {
			screenX := textAreaX + col
			screenY := bounds.Y + row
			bufferCol := e.scrollX + col

			style := textStyle
			var ch rune = ' '

			// Check if this position is in selection
			if e.selection != nil && e.isInSelection(lineNum, bufferCol) {
				style = e.theme.UIStyle("selection")
			}

			// Get character at this position
			if bufferCol < len(lineText) {
				ch = rune(lineText[bufferCol])
			}

			// Highlight cursor position
			if lineNum == e.cursor.Line && bufferCol == e.cursor.Col {
				style = style.Reverse(true)
			}

			screen.SetContent(screenX, screenY, ch, nil, style)
		}
	}
}

// clearRow clears a row on the screen.
func (e *EditorView) clearRow(screen tcell.Screen, y, x, width int) {
	style := e.theme.UIStyle("default")
	for i := 0; i < width; i++ {
		screen.SetContent(x+i, y, ' ', nil, style)
	}
}

// isInSelection checks if a position is within the current selection.
func (e *EditorView) isInSelection(line, col int) bool {
	if e.selection == nil {
		return false
	}
	start, end := e.selection.Start, e.selection.End
	// Normalize so start is before end
	if start.Line > end.Line || (start.Line == end.Line && start.Col > end.Col) {
		start, end = end, start
	}

	pos := types.Position{Line: line, Col: col}
	if pos.Line < start.Line || pos.Line > end.Line {
		return false
	}
	if pos.Line == start.Line && pos.Col < start.Col {
		return false
	}
	if pos.Line == end.Line && pos.Col >= end.Col {
		return false
	}
	return true
}

// HandleEvent processes input events.
func (e *EditorView) HandleEvent(ev tcell.Event) bool {
	if !e.IsFocused() {
		return false
	}

	switch event := ev.(type) {
	case *tcell.EventKey:
		return e.handleKeyEvent(event)
	}
	return false
}

// handleKeyEvent processes keyboard input.
func (e *EditorView) handleKeyEvent(ev *tcell.EventKey) bool {
	key := ev.Key()

	// Handle special keys
	switch key {
	case tcell.KeyUp:
		e.MoveCursorUp()
		return true
	case tcell.KeyDown:
		e.MoveCursorDown()
		return true
	case tcell.KeyLeft:
		e.MoveCursorLeft()
		return true
	case tcell.KeyRight:
		e.MoveCursorRight()
		return true
	case tcell.KeyHome:
		e.MoveCursorToLineStart()
		return true
	case tcell.KeyEnd:
		e.MoveCursorToLineEnd()
		return true
	case tcell.KeyCtrlA:
		e.MoveCursorToLineStart()
		return true
	case tcell.KeyCtrlE:
		e.MoveCursorToLineEnd()
		return true
	case tcell.KeyEnter:
		e.InsertNewline()
		return true
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		e.DeleteBack()
		return true
	case tcell.KeyDelete:
		e.DeleteForward()
		return true
	case tcell.KeyRune:
		// Handle regular character input
		e.InsertChar(ev.Rune())
		return true
	}

	return false
}

// MoveCursorUp moves the cursor up one line.
func (e *EditorView) MoveCursorUp() {
	if e.cursor.Line > 0 {
		e.cursor.Line--
		e.clampCursorCol()
		e.ensureCursorVisible()
	}
}

// MoveCursorDown moves the cursor down one line.
func (e *EditorView) MoveCursorDown() {
	if e.cursor.Line < e.buf.LineCount()-1 {
		e.cursor.Line++
		e.clampCursorCol()
		e.ensureCursorVisible()
	}
}

// MoveCursorLeft moves the cursor left one character.
func (e *EditorView) MoveCursorLeft() {
	if e.cursor.Col > 0 {
		e.cursor.Col--
	} else if e.cursor.Line > 0 {
		// Move to end of previous line
		e.cursor.Line--
		e.cursor.Col = len(e.buf.Line(e.cursor.Line))
	}
	e.ensureCursorVisible()
}

// MoveCursorRight moves the cursor right one character.
func (e *EditorView) MoveCursorRight() {
	lineLen := len(e.buf.Line(e.cursor.Line))
	if e.cursor.Col < lineLen {
		e.cursor.Col++
	} else if e.cursor.Line < e.buf.LineCount()-1 {
		// Move to start of next line
		e.cursor.Line++
		e.cursor.Col = 0
	}
	e.ensureCursorVisible()
}

// MoveCursorToLineStart moves the cursor to the start of the current line.
func (e *EditorView) MoveCursorToLineStart() {
	e.cursor.Col = 0
	e.ensureCursorVisible()
}

// MoveCursorToLineEnd moves the cursor to the end of the current line.
func (e *EditorView) MoveCursorToLineEnd() {
	e.cursor.Col = len(e.buf.Line(e.cursor.Line))
	e.ensureCursorVisible()
}

// MoveCursorToBufferStart moves the cursor to the start of the buffer.
func (e *EditorView) MoveCursorToBufferStart() {
	e.cursor.Line = 0
	e.cursor.Col = 0
	e.ensureCursorVisible()
}

// MoveCursorToBufferEnd moves the cursor to the end of the buffer.
func (e *EditorView) MoveCursorToBufferEnd() {
	if e.buf.LineCount() > 0 {
		e.cursor.Line = e.buf.LineCount() - 1
		e.cursor.Col = len(e.buf.Line(e.cursor.Line))
	} else {
		e.cursor.Line = 0
		e.cursor.Col = 0
	}
	e.ensureCursorVisible()
}

// clampCursorCol ensures the cursor column is within the current line's bounds.
func (e *EditorView) clampCursorCol() {
	if e.cursor.Line >= e.buf.LineCount() {
		e.cursor.Line = e.buf.LineCount() - 1
		if e.cursor.Line < 0 {
			e.cursor.Line = 0
		}
	}
	lineLen := len(e.buf.Line(e.cursor.Line))
	if e.cursor.Col > lineLen {
		e.cursor.Col = lineLen
	}
}

// ensureCursorVisible adjusts scroll position to keep cursor visible.
func (e *EditorView) ensureCursorVisible() {
	bounds := e.Bounds()

	// Vertical scrolling
	if e.cursor.Line < e.scrollY {
		e.scrollY = e.cursor.Line
	} else if e.cursor.Line >= e.scrollY+bounds.Height {
		e.scrollY = e.cursor.Line - bounds.Height + 1
	}

	// Horizontal scrolling
	textAreaWidth := bounds.Width - e.lineNumWidth
	if textAreaWidth < 0 {
		textAreaWidth = 0
	}
	if e.cursor.Col < e.scrollX {
		e.scrollX = e.cursor.Col
	} else if e.cursor.Col >= e.scrollX+textAreaWidth {
		e.scrollX = e.cursor.Col - textAreaWidth + 1
	}
}

// InsertChar inserts a character at the cursor position.
func (e *EditorView) InsertChar(ch rune) {
	if e.buf.ReadOnly {
		return
	}
	e.buf.Insert(e.cursor.Line, e.cursor.Col, string(ch))
	e.cursor.Col++
	e.ensureCursorVisible()
}

// InsertNewline inserts a newline at the cursor position.
func (e *EditorView) InsertNewline() {
	if e.buf.ReadOnly {
		return
	}
	e.buf.Insert(e.cursor.Line, e.cursor.Col, "\n")
	e.cursor.Line++
	e.cursor.Col = 0
	e.ensureCursorVisible()
}

// DeleteBack deletes the character before the cursor (backspace).
func (e *EditorView) DeleteBack() {
	if e.buf.ReadOnly {
		return
	}
	if e.cursor.Col > 0 {
		e.buf.Delete(e.cursor.Line, e.cursor.Col-1, 1)
		e.cursor.Col--
	} else if e.cursor.Line > 0 {
		// Delete newline at end of previous line
		prevLineLen := len(e.buf.Line(e.cursor.Line - 1))
		e.buf.Delete(e.cursor.Line-1, prevLineLen, 1)
		e.cursor.Line--
		e.cursor.Col = prevLineLen
	}
	e.ensureCursorVisible()
}

// DeleteForward deletes the character at the cursor position (delete key).
func (e *EditorView) DeleteForward() {
	if e.buf.ReadOnly {
		return
	}
	lineLen := len(e.buf.Line(e.cursor.Line))
	if e.cursor.Col < lineLen {
		e.buf.Delete(e.cursor.Line, e.cursor.Col, 1)
	} else if e.cursor.Line < e.buf.LineCount()-1 {
		// Delete newline at end of current line
		e.buf.Delete(e.cursor.Line, e.cursor.Col, 1)
	}
	e.ensureCursorVisible()
}

// StartSelection begins a selection at the current cursor position.
func (e *EditorView) StartSelection() {
	e.selection = &types.Selection{
		Start: e.cursor,
		End:   e.cursor,
	}
}

// ExtendSelection extends the selection to the current cursor position.
func (e *EditorView) ExtendSelection() {
	if e.selection != nil {
		e.selection.End = e.cursor
	}
}

// ClearSelection removes the current selection.
func (e *EditorView) ClearSelection() {
	e.selection = nil
}

// CursorPosition returns the current cursor position.
func (e *EditorView) CursorPosition() types.Position {
	return e.cursor
}

// ScrollPosition returns the current scroll position.
func (e *EditorView) ScrollPosition() (scrollY, scrollX int) {
	return e.scrollY, e.scrollX
}

// Buffer returns the underlying buffer.
func (e *EditorView) Buffer() *buffer.Buffer {
	return e.buf
}

// SetBuffer sets a new buffer for the editor view.
func (e *EditorView) SetBuffer(buf *buffer.Buffer) {
	e.buf = buf
	e.cursor = types.Position{Line: 0, Col: 0}
	e.scrollY = 0
	e.scrollX = 0
	e.selection = nil
	e.updateLineNumWidth()
}

// Selection returns the current selection, or nil if none.
func (e *EditorView) Selection() *types.Selection {
	return e.selection
}
