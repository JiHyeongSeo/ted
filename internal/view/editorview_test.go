package view

import (
	"testing"

	"github.com/seoji/ted/internal/buffer"
	"github.com/seoji/ted/internal/syntax"
	"github.com/seoji/ted/internal/types"
)

// TestNewEditorView tests EditorView creation.
func TestNewEditorView(t *testing.T) {
	buf := buffer.NewBuffer("Hello\nWorld\n")
	theme := syntax.DefaultTheme()
	ev := NewEditorView(buf, theme)

	if ev == nil {
		t.Fatal("NewEditorView returned nil")
	}
	if ev.buf != buf {
		t.Error("Buffer not set correctly")
	}
	if ev.theme != theme {
		t.Error("Theme not set correctly")
	}
	if ev.cursor.Line != 0 || ev.cursor.Col != 0 {
		t.Errorf("Initial cursor position should be (0,0), got (%d,%d)", ev.cursor.Line, ev.cursor.Col)
	}
	if ev.scrollY != 0 || ev.scrollX != 0 {
		t.Errorf("Initial scroll position should be (0,0), got (%d,%d)", ev.scrollY, ev.scrollX)
	}
}

// TestCursorMovement tests basic cursor movement.
func TestCursorMovement(t *testing.T) {
	buf := buffer.NewBuffer("Line 1\nLine 2\nLine 3\n")
	theme := syntax.DefaultTheme()
	ev := NewEditorView(buf, theme)
	ev.SetBounds(types.Rect{X: 0, Y: 0, Width: 80, Height: 24})

	// Move down
	ev.MoveCursorDown()
	if ev.cursor.Line != 1 || ev.cursor.Col != 0 {
		t.Errorf("After MoveCursorDown, expected (1,0), got (%d,%d)", ev.cursor.Line, ev.cursor.Col)
	}

	// Move right
	ev.MoveCursorRight()
	if ev.cursor.Line != 1 || ev.cursor.Col != 1 {
		t.Errorf("After MoveCursorRight, expected (1,1), got (%d,%d)", ev.cursor.Line, ev.cursor.Col)
	}

	// Move up
	ev.MoveCursorUp()
	if ev.cursor.Line != 0 || ev.cursor.Col != 1 {
		t.Errorf("After MoveCursorUp, expected (0,1), got (%d,%d)", ev.cursor.Line, ev.cursor.Col)
	}

	// Move left
	ev.MoveCursorLeft()
	if ev.cursor.Line != 0 || ev.cursor.Col != 0 {
		t.Errorf("After MoveCursorLeft, expected (0,0), got (%d,%d)", ev.cursor.Line, ev.cursor.Col)
	}
}

// TestCursorClamping tests cursor clamping at buffer boundaries.
func TestCursorClamping(t *testing.T) {
	buf := buffer.NewBuffer("Line 1\nLine 2\n")
	theme := syntax.DefaultTheme()
	ev := NewEditorView(buf, theme)
	ev.SetBounds(types.Rect{X: 0, Y: 0, Width: 80, Height: 24})

	// Try to move up from first line - should stay at (0,0)
	ev.MoveCursorUp()
	if ev.cursor.Line != 0 {
		t.Errorf("Cursor moved up from first line, got line %d", ev.cursor.Line)
	}

	// Try to move left from column 0 - should stay at (0,0)
	ev.MoveCursorLeft()
	if ev.cursor.Line != 0 || ev.cursor.Col != 0 {
		t.Errorf("Cursor moved left from (0,0), got (%d,%d)", ev.cursor.Line, ev.cursor.Col)
	}

	// Move to last line
	ev.MoveCursorToBufferEnd()
	lastLine := buf.LineCount() - 1
	if ev.cursor.Line != lastLine {
		t.Errorf("MoveCursorToBufferEnd: expected line %d, got %d", lastLine, ev.cursor.Line)
	}

	// Try to move down from last line - should stay there
	initialLine := ev.cursor.Line
	ev.MoveCursorDown()
	if ev.cursor.Line != initialLine {
		t.Errorf("Cursor moved down from last line, got line %d", ev.cursor.Line)
	}
}

// TestCursorAtLineBoundaries tests cursor behavior at line boundaries.
func TestCursorAtLineBoundaries(t *testing.T) {
	buf := buffer.NewBuffer("abc\ndefgh\n")
	theme := syntax.DefaultTheme()
	ev := NewEditorView(buf, theme)
	ev.SetBounds(types.Rect{X: 0, Y: 0, Width: 80, Height: 24})

	// Move to end of first line
	ev.MoveCursorToLineEnd()
	if ev.cursor.Col != 3 { // "abc" has length 3
		t.Errorf("MoveCursorToLineEnd on line 0: expected col 3, got %d", ev.cursor.Col)
	}

	// Move right - should go to start of next line
	ev.MoveCursorRight()
	if ev.cursor.Line != 1 || ev.cursor.Col != 0 {
		t.Errorf("MoveCursorRight at line end: expected (1,0), got (%d,%d)", ev.cursor.Line, ev.cursor.Col)
	}

	// Move left - should go to end of previous line
	ev.MoveCursorLeft()
	if ev.cursor.Line != 0 || ev.cursor.Col != 3 {
		t.Errorf("MoveCursorLeft at line start: expected (0,3), got (%d,%d)", ev.cursor.Line, ev.cursor.Col)
	}

	// Move to start of line
	ev.MoveCursorToLineStart()
	if ev.cursor.Col != 0 {
		t.Errorf("MoveCursorToLineStart: expected col 0, got %d", ev.cursor.Col)
	}
}

// TestCursorAtLineBoundariesNoWrap tests that cursor at start doesn't wrap when already on first line.
func TestCursorAtLineBoundariesNoWrap(t *testing.T) {
	buf := buffer.NewBuffer("abc\ndefgh\n")
	theme := syntax.DefaultTheme()
	ev := NewEditorView(buf, theme)
	ev.SetBounds(types.Rect{X: 0, Y: 0, Width: 80, Height: 24})

	// At (0, 0), move left should stay at (0, 0)
	ev.MoveCursorLeft()
	if ev.cursor.Line != 0 || ev.cursor.Col != 0 {
		t.Errorf("MoveCursorLeft at (0,0): expected (0,0), got (%d,%d)", ev.cursor.Line, ev.cursor.Col)
	}

	// Move to end of last line
	ev.MoveCursorToBufferEnd()
	lastLine := buf.LineCount() - 1
	lastCol := len(buf.Line(lastLine))

	// Move right should stay at end
	ev.MoveCursorRight()
	if ev.cursor.Line != lastLine || ev.cursor.Col != lastCol {
		t.Errorf("MoveCursorRight at buffer end: expected (%d,%d), got (%d,%d)",
			lastLine, lastCol, ev.cursor.Line, ev.cursor.Col)
	}
}

// TestScrolling tests automatic scrolling when cursor moves out of view.
func TestScrolling(t *testing.T) {
	// Create buffer with many lines
	content := ""
	for i := 0; i < 100; i++ {
		content += "Line\n"
	}
	buf := buffer.NewBuffer(content)
	theme := syntax.DefaultTheme()
	ev := NewEditorView(buf, theme)
	ev.SetBounds(types.Rect{X: 0, Y: 0, Width: 80, Height: 10}) // Only 10 lines visible

	// Move cursor to line 20
	for i := 0; i < 20; i++ {
		ev.MoveCursorDown()
	}

	// ScrollY should have adjusted to keep cursor visible
	if ev.scrollY > 20 {
		t.Errorf("ScrollY too high: %d (cursor at line 20)", ev.scrollY)
	}
	if ev.scrollY+10 <= 20 {
		t.Errorf("ScrollY too low: %d (cursor at line 20, height 10)", ev.scrollY)
	}

	// Verify cursor is in visible range
	if ev.cursor.Line < ev.scrollY || ev.cursor.Line >= ev.scrollY+10 {
		t.Errorf("Cursor line %d not in visible range [%d, %d)",
			ev.cursor.Line, ev.scrollY, ev.scrollY+10)
	}
}

// TestHorizontalScrolling tests horizontal scrolling for long lines.
func TestHorizontalScrolling(t *testing.T) {
	buf := buffer.NewBuffer("This is a very long line that should cause horizontal scrolling\n")
	theme := syntax.DefaultTheme()
	ev := NewEditorView(buf, theme)
	ev.SetBounds(types.Rect{X: 0, Y: 0, Width: 15, Height: 10}) // Narrow width

	// Move cursor to column 30
	for i := 0; i < 30; i++ {
		ev.MoveCursorRight()
	}

	// ScrollX should have adjusted
	textAreaWidth := 15 - ev.lineNumWidth
	if ev.scrollX > 30 {
		t.Errorf("ScrollX too high: %d (cursor at col 30)", ev.scrollX)
	}
	if ev.scrollX+textAreaWidth <= 30 {
		t.Errorf("ScrollX too low: %d (cursor at col 30, width %d)", ev.scrollX, textAreaWidth)
	}
}

// TestInsertChar tests character insertion.
func TestInsertChar(t *testing.T) {
	buf := buffer.NewBuffer("Hello\n")
	theme := syntax.DefaultTheme()
	ev := NewEditorView(buf, theme)
	ev.SetBounds(types.Rect{X: 0, Y: 0, Width: 80, Height: 24})

	// Insert 'X' at start
	ev.InsertChar('X')
	if buf.Line(0) != "XHello" {
		t.Errorf("After InsertChar('X'), expected 'XHello', got '%s'", buf.Line(0))
	}
	if ev.cursor.Col != 1 {
		t.Errorf("After InsertChar, cursor col should be 1, got %d", ev.cursor.Col)
	}

	// Insert another character
	ev.InsertChar('Y')
	if buf.Line(0) != "XYHello" {
		t.Errorf("After InsertChar('Y'), expected 'XYHello', got '%s'", buf.Line(0))
	}
	if ev.cursor.Col != 2 {
		t.Errorf("After second InsertChar, cursor col should be 2, got %d", ev.cursor.Col)
	}
}

// TestDeleteBack tests backspace deletion.
func TestDeleteBack(t *testing.T) {
	buf := buffer.NewBuffer("Hello\n")
	theme := syntax.DefaultTheme()
	ev := NewEditorView(buf, theme)
	ev.SetBounds(types.Rect{X: 0, Y: 0, Width: 80, Height: 24})

	// Move to position 2
	ev.cursor.Col = 2

	// Delete back
	ev.DeleteBack()
	if buf.Line(0) != "Hllo" {
		t.Errorf("After DeleteBack, expected 'Hllo', got '%s'", buf.Line(0))
	}
	if ev.cursor.Col != 1 {
		t.Errorf("After DeleteBack, cursor col should be 1, got %d", ev.cursor.Col)
	}
}

// TestDeleteBackAtLineStart tests backspace at line start (should join lines).
func TestDeleteBackAtLineStart(t *testing.T) {
	buf := buffer.NewBuffer("Hello\nWorld\n")
	theme := syntax.DefaultTheme()
	ev := NewEditorView(buf, theme)
	ev.SetBounds(types.Rect{X: 0, Y: 0, Width: 80, Height: 24})

	// Move to start of second line
	ev.cursor.Line = 1
	ev.cursor.Col = 0

	// Delete back - should join lines
	ev.DeleteBack()
	if buf.LineCount() != 2 { // "HelloWorld\n" counts as 2 lines (last empty)
		t.Errorf("After DeleteBack at line start, expected 2 lines, got %d", buf.LineCount())
	}
	if buf.Line(0) != "HelloWorld" {
		t.Errorf("After DeleteBack at line start, expected 'HelloWorld', got '%s'", buf.Line(0))
	}
	if ev.cursor.Line != 0 || ev.cursor.Col != 5 {
		t.Errorf("After DeleteBack at line start, expected cursor (0,5), got (%d,%d)",
			ev.cursor.Line, ev.cursor.Col)
	}
}

// TestDeleteForward tests delete key.
func TestDeleteForward(t *testing.T) {
	buf := buffer.NewBuffer("Hello\n")
	theme := syntax.DefaultTheme()
	ev := NewEditorView(buf, theme)
	ev.SetBounds(types.Rect{X: 0, Y: 0, Width: 80, Height: 24})

	// Position at 'e'
	ev.cursor.Col = 1

	// Delete forward
	ev.DeleteForward()
	if buf.Line(0) != "Hllo" {
		t.Errorf("After DeleteForward, expected 'Hllo', got '%s'", buf.Line(0))
	}
	if ev.cursor.Col != 1 {
		t.Errorf("After DeleteForward, cursor col should stay at 1, got %d", ev.cursor.Col)
	}
}

// TestDeleteForwardAtLineEnd tests delete at line end (should join lines).
func TestDeleteForwardAtLineEnd(t *testing.T) {
	buf := buffer.NewBuffer("Hello\nWorld\n")
	theme := syntax.DefaultTheme()
	ev := NewEditorView(buf, theme)
	ev.SetBounds(types.Rect{X: 0, Y: 0, Width: 80, Height: 24})

	// Move to end of first line
	ev.cursor.Col = 5

	// Delete forward - should join lines
	ev.DeleteForward()
	if buf.LineCount() != 2 {
		t.Errorf("After DeleteForward at line end, expected 2 lines, got %d", buf.LineCount())
	}
	if buf.Line(0) != "HelloWorld" {
		t.Errorf("After DeleteForward at line end, expected 'HelloWorld', got '%s'", buf.Line(0))
	}
	if ev.cursor.Line != 0 || ev.cursor.Col != 5 {
		t.Errorf("After DeleteForward at line end, expected cursor (0,5), got (%d,%d)",
			ev.cursor.Line, ev.cursor.Col)
	}
}

// TestInsertNewline tests newline insertion.
func TestInsertNewline(t *testing.T) {
	buf := buffer.NewBuffer("Hello")
	theme := syntax.DefaultTheme()
	ev := NewEditorView(buf, theme)
	ev.SetBounds(types.Rect{X: 0, Y: 0, Width: 80, Height: 24})

	// Position at 'l'
	ev.cursor.Col = 3

	// Insert newline
	ev.InsertNewline()
	if buf.LineCount() != 2 {
		t.Errorf("After InsertNewline, expected 2 lines, got %d", buf.LineCount())
	}
	if buf.Line(0) != "Hel" {
		t.Errorf("After InsertNewline, line 0 should be 'Hel', got '%s'", buf.Line(0))
	}
	if buf.Line(1) != "lo" {
		t.Errorf("After InsertNewline, line 1 should be 'lo', got '%s'", buf.Line(1))
	}
	if ev.cursor.Line != 1 || ev.cursor.Col != 0 {
		t.Errorf("After InsertNewline, expected cursor (1,0), got (%d,%d)",
			ev.cursor.Line, ev.cursor.Col)
	}
}

// TestLineNumberWidth tests line number width calculation.
func TestLineNumberWidth(t *testing.T) {
	tests := []struct {
		lines int // actual lines in buffer (including empty line after final \n)
		width int
	}{
		{1, 2},      // "1 " -> 1 digit + 1 padding
		{9, 2},      // "9 " -> 1 digit + 1 padding
		{10, 3},     // "10 " -> 2 digits + 1 padding
		{99, 3},     // "99 " -> 2 digits + 1 padding
		{100, 4},    // "100 " -> 3 digits + 1 padding
		{999, 4},    // "999 " -> 3 digits + 1 padding
		{1000, 5},   // "1000 " -> 4 digits + 1 padding
	}

	theme := syntax.DefaultTheme()
	for _, tt := range tests {
		// Create buffer with specified number of lines
		// Note: "x\n" repeated N times creates N+1 lines (last is empty)
		// So to get exactly tt.lines, we need tt.lines-1 newlines
		content := ""
		for i := 0; i < tt.lines-1; i++ {
			content += "x\n"
		}
		buf := buffer.NewBuffer(content)

		actualLines := buf.LineCount()
		if actualLines != tt.lines {
			t.Errorf("Test setup error: wanted %d lines, got %d", tt.lines, actualLines)
			continue
		}

		ev := NewEditorView(buf, theme)

		if ev.lineNumWidth != tt.width {
			t.Errorf("For %d lines, expected width %d, got %d", tt.lines, tt.width, ev.lineNumWidth)
		}
	}
}

// TestSelection tests selection creation and extension.
func TestSelection(t *testing.T) {
	buf := buffer.NewBuffer("Line 1\nLine 2\nLine 3\n")
	theme := syntax.DefaultTheme()
	ev := NewEditorView(buf, theme)
	ev.SetBounds(types.Rect{X: 0, Y: 0, Width: 80, Height: 24})

	// Initially no selection
	if ev.Selection() != nil {
		t.Error("Initially should have no selection")
	}

	// Start selection
	ev.StartSelection()
	if ev.Selection() == nil {
		t.Fatal("StartSelection should create a selection")
	}
	if ev.Selection().Start != ev.cursor || ev.Selection().End != ev.cursor {
		t.Error("StartSelection should set both Start and End to cursor position")
	}

	// Move cursor and extend selection
	ev.MoveCursorRight()
	ev.MoveCursorRight()
	ev.ExtendSelection()
	if ev.Selection().End != ev.cursor {
		t.Error("ExtendSelection should update End to cursor position")
	}

	// Clear selection
	ev.ClearSelection()
	if ev.Selection() != nil {
		t.Error("ClearSelection should remove selection")
	}
}

// TestSetBuffer tests changing the buffer.
func TestSetBuffer(t *testing.T) {
	buf1 := buffer.NewBuffer("Buffer 1\n")
	buf2 := buffer.NewBuffer("Buffer 2\n")
	theme := syntax.DefaultTheme()
	ev := NewEditorView(buf1, theme)
	ev.SetBounds(types.Rect{X: 0, Y: 0, Width: 80, Height: 24})

	// Move cursor and scroll
	ev.cursor.Line = 0
	ev.cursor.Col = 5
	ev.scrollY = 10
	ev.scrollX = 5
	ev.StartSelection()

	// Set new buffer
	ev.SetBuffer(buf2)

	// Should reset cursor and scroll
	if ev.cursor.Line != 0 || ev.cursor.Col != 0 {
		t.Errorf("SetBuffer should reset cursor to (0,0), got (%d,%d)", ev.cursor.Line, ev.cursor.Col)
	}
	if ev.scrollY != 0 || ev.scrollX != 0 {
		t.Errorf("SetBuffer should reset scroll to (0,0), got (%d,%d)", ev.scrollY, ev.scrollX)
	}
	if ev.Selection() != nil {
		t.Error("SetBuffer should clear selection")
	}
	if ev.Buffer() != buf2 {
		t.Error("SetBuffer should update buffer")
	}
}

// TestReadOnlyBuffer tests that read-only buffers reject edits.
func TestReadOnlyBuffer(t *testing.T) {
	buf := buffer.NewBuffer("Hello\n")
	buf.ReadOnly = true
	theme := syntax.DefaultTheme()
	ev := NewEditorView(buf, theme)
	ev.SetBounds(types.Rect{X: 0, Y: 0, Width: 80, Height: 24})

	// Try to insert
	ev.InsertChar('X')
	if buf.Line(0) != "Hello" {
		t.Error("InsertChar should be no-op on read-only buffer")
	}

	// Try to delete
	ev.cursor.Col = 1
	ev.DeleteBack()
	if buf.Line(0) != "Hello" {
		t.Error("DeleteBack should be no-op on read-only buffer")
	}

	// Try to insert newline
	ev.InsertNewline()
	if buf.LineCount() != 2 {
		t.Error("InsertNewline should be no-op on read-only buffer")
	}
}

// TestCursorPosition tests the CursorPosition getter.
func TestCursorPosition(t *testing.T) {
	buf := buffer.NewBuffer("Hello\nWorld\n")
	theme := syntax.DefaultTheme()
	ev := NewEditorView(buf, theme)

	pos := ev.CursorPosition()
	if pos.Line != 0 || pos.Col != 0 {
		t.Errorf("Initial CursorPosition should be (0,0), got (%d,%d)", pos.Line, pos.Col)
	}

	ev.cursor.Line = 1
	ev.cursor.Col = 3
	pos = ev.CursorPosition()
	if pos.Line != 1 || pos.Col != 3 {
		t.Errorf("CursorPosition should reflect cursor, got (%d,%d)", pos.Line, pos.Col)
	}
}

// TestScrollPosition tests the ScrollPosition getter.
func TestScrollPosition(t *testing.T) {
	buf := buffer.NewBuffer("Hello\nWorld\n")
	theme := syntax.DefaultTheme()
	ev := NewEditorView(buf, theme)

	y, x := ev.ScrollPosition()
	if y != 0 || x != 0 {
		t.Errorf("Initial ScrollPosition should be (0,0), got (%d,%d)", y, x)
	}

	ev.scrollY = 10
	ev.scrollX = 5
	y, x = ev.ScrollPosition()
	if y != 10 || x != 5 {
		t.Errorf("ScrollPosition should reflect scroll values, got (%d,%d)", y, x)
	}
}
