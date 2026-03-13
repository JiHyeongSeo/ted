package view

import (
	"fmt"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/seoji/ted/internal/buffer"
	"github.com/seoji/ted/internal/syntax"
	"github.com/seoji/ted/internal/types"
)

// SearchHighlight represents a highlighted search match in the editor.
type SearchHighlight struct {
	Line   int
	Col    int // rune-based column
	Length int // length in runes
}

// EditorView renders a text buffer with line numbers, cursor, and scrolling.
type EditorView struct {
	BaseComponent
	buf              *buffer.Buffer
	theme            *syntax.Theme
	highlighter      *syntax.Highlighter // syntax highlighter for the current language
	cursor           types.Position      // cursor position (rune-based col)
	scrollY          int                 // first visible line
	scrollX          int                 // horizontal scroll offset (display columns)
	selection        *types.Selection    // current selection range
	lineNumWidth     int                 // width of line number gutter
	cursorScreenX    int                 // last computed screen X of cursor (for ShowCursor)
	cursorScreenY    int                 // last computed screen Y of cursor
	clipboard        string              // internal clipboard
	searchHighlights []SearchHighlight   // search match highlights
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

// tabWidth is the number of spaces per tab stop.
const tabWidth = 4

// runeDisplayWidth returns the display width of a line up to runeCol runes.
func (e *EditorView) runeDisplayWidth(line int, runeCol int) int {
	lineText := e.buf.Line(line)
	runes := []rune(lineText)
	w := 0
	for i := 0; i < runeCol && i < len(runes); i++ {
		if runes[i] == '\t' {
			w += tabWidth - (w % tabWidth)
		} else {
			w += runewidth.RuneWidth(runes[i])
		}
	}
	return w
}

// lineDisplayWidth returns the total display width of a line.
func (e *EditorView) lineDisplayWidth(line int) int {
	return e.runeDisplayWidth(line, e.runeLen(line))
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
		screen.SetContent(bounds.X+e.lineNumWidth-1, bounds.Y+row, ' ', nil, lineNumStyle)

		// Draw line content with wide character support
		lineText := e.buf.Line(lineNum)
		lineRunes := []rune(lineText)
		textStyle := e.theme.UIStyle("default")

		// Get syntax highlighting tokens
		var tokens []syntax.Token
		if e.highlighter != nil {
			tokens = e.highlighter.HighlightLine(lineText)
		}

		// Clear the text area for this row first
		for x := textAreaX; x < textAreaX+textAreaWidth; x++ {
			screen.SetContent(x, bounds.Y+row, ' ', nil, textStyle)
		}

		// Render runes, tracking display column
		screenCol := 0 // display column relative to textAreaX
		for runeIdx := 0; runeIdx < len(lineRunes); runeIdx++ {
			ch := lineRunes[runeIdx]

			// Calculate display width for this rune
			var w int
			if ch == '\t' {
				w = tabWidth - (screenCol % tabWidth)
			} else {
				w = runewidth.RuneWidth(ch)
			}

			// Skip runes that are scrolled off to the left
			if screenCol+w <= e.scrollX {
				screenCol += w
				continue
			}
			// Stop if past the visible area
			dispCol := screenCol - e.scrollX
			if dispCol >= textAreaWidth {
				break
			}

			style := textStyle

			// Apply syntax highlighting
			if e.highlighter != nil {
				for _, token := range tokens {
					if runeIdx >= token.Start && runeIdx < token.Start+token.Length {
						style = e.highlighter.StyleForToken(token.Type)
						break
					}
				}
			}

			// Search highlights
			if len(e.searchHighlights) > 0 {
				for _, h := range e.searchHighlights {
					if h.Line == lineNum && runeIdx >= h.Col && runeIdx < h.Col+h.Length {
						style = style.Background(tcell.ColorYellow).Foreground(tcell.ColorBlack)
						break
					}
				}
			}

			// Selection (overrides search highlight)
			if e.selection != nil && e.isInSelection(lineNum, runeIdx) {
				style = e.theme.UIStyle("selection")
			}

			// Track cursor screen position
			if lineNum == e.cursor.Line && runeIdx == e.cursor.Col {
				e.cursorScreenX = textAreaX + dispCol
				e.cursorScreenY = bounds.Y + row
			}

			screenX := textAreaX + dispCol

			if ch == '\t' {
				// Render tab as spaces
				for i := 0; i < w && dispCol+i < textAreaWidth; i++ {
					screen.SetContent(screenX+i, bounds.Y+row, ' ', nil, style)
				}
			} else if dispCol+w > textAreaWidth {
				// Handle wide chars that would be partially clipped at right edge
				screen.SetContent(screenX, bounds.Y+row, ' ', nil, style)
			} else {
				screen.SetContent(screenX, bounds.Y+row, ch, nil, style)
			}

			screenCol += w
		}

		// Track cursor position when at end of line
		if lineNum == e.cursor.Line && e.cursor.Col >= len(lineRunes) {
			cursorDispCol := e.runeDisplayWidth(lineNum, e.cursor.Col) - e.scrollX
			if cursorDispCol >= 0 && cursorDispCol < textAreaWidth {
				e.cursorScreenX = textAreaX + cursorDispCol
				e.cursorScreenY = bounds.Y + row
			}
		}

		// Place the hardware cursor (thin beam)
		if lineNum == e.cursor.Line {
			screen.ShowCursor(e.cursorScreenX, e.cursorScreenY)
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
	switch event := ev.(type) {
	case *tcell.EventKey:
		return e.handleKeyEvent(event)
	}
	return false
}

// handleKeyEvent processes keyboard input.
func (e *EditorView) handleKeyEvent(ev *tcell.EventKey) bool {
	key := ev.Key()
	mod := ev.Modifiers()
	shift := mod&tcell.ModShift != 0

	// Shift+arrow for selection
	switch key {
	case tcell.KeyUp:
		if shift {
			e.startOrExtendSelection()
		} else {
			e.ClearSelection()
		}
		e.MoveCursorUp()
		if shift {
			e.ExtendSelection()
		}
		return true
	case tcell.KeyDown:
		if shift {
			e.startOrExtendSelection()
		} else {
			e.ClearSelection()
		}
		e.MoveCursorDown()
		if shift {
			e.ExtendSelection()
		}
		return true
	case tcell.KeyLeft:
		if shift {
			e.startOrExtendSelection()
		} else {
			e.ClearSelection()
		}
		e.MoveCursorLeft()
		if shift {
			e.ExtendSelection()
		}
		return true
	case tcell.KeyRight:
		if shift {
			e.startOrExtendSelection()
		} else {
			e.ClearSelection()
		}
		e.MoveCursorRight()
		if shift {
			e.ExtendSelection()
		}
		return true
	case tcell.KeyHome:
		if shift {
			e.startOrExtendSelection()
		} else {
			e.ClearSelection()
		}
		e.MoveCursorToLineStart()
		if shift {
			e.ExtendSelection()
		}
		return true
	case tcell.KeyEnd:
		if shift {
			e.startOrExtendSelection()
		} else {
			e.ClearSelection()
		}
		e.MoveCursorToLineEnd()
		if shift {
			e.ExtendSelection()
		}
		return true
	case tcell.KeyCtrlA:
		e.ClearSelection()
		e.MoveCursorToLineStart()
		return true
	case tcell.KeyCtrlE:
		e.ClearSelection()
		e.MoveCursorToLineEnd()
		return true
	case tcell.KeyTab:
		e.deleteSelectionIfAny()
		e.InsertTab()
		return true
	case tcell.KeyEnter:
		e.deleteSelectionIfAny()
		e.InsertNewlineWithIndent()
		return true
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if e.selection != nil {
			e.deleteSelectionIfAny()
		} else {
			e.DeleteBack()
		}
		return true
	case tcell.KeyDelete:
		if e.selection != nil {
			e.deleteSelectionIfAny()
		} else {
			e.DeleteForward()
		}
		return true
	case tcell.KeyRune:
		e.deleteSelectionIfAny()
		e.InsertChar(ev.Rune())
		return true
	}

	return false
}

// startOrExtendSelection starts selection if none exists.
func (e *EditorView) startOrExtendSelection() {
	if e.selection == nil {
		e.StartSelection()
	}
}

// deleteSelectionIfAny deletes selected text and clears selection.
func (e *EditorView) deleteSelectionIfAny() {
	if e.selection == nil {
		return
	}
	start, end := e.selection.Start, e.selection.End
	if start.Line > end.Line || (start.Line == end.Line && start.Col > end.Col) {
		start, end = end, start
	}

	// Calculate byte range and delete
	startByte := e.runeColToByteCol(start.Line, start.Col)
	startOffset := e.buf.LineOffset(start.Line) + startByte
	endByte := e.runeColToByteCol(end.Line, end.Col)
	endOffset := e.buf.LineOffset(end.Line) + endByte

	length := endOffset - startOffset
	if length > 0 {
		e.buf.Delete(start.Line, startByte, length)
	}

	e.cursor = start
	e.selection = nil
	e.ensureCursorVisible()
}

// SelectedText returns the currently selected text.
func (e *EditorView) SelectedText() string {
	if e.selection == nil {
		return ""
	}
	start, end := e.selection.Start, e.selection.End
	if start.Line > end.Line || (start.Line == end.Line && start.Col > end.Col) {
		start, end = end, start
	}

	startByte := e.runeColToByteCol(start.Line, start.Col)
	startOffset := e.buf.LineOffset(start.Line) + startByte
	endByte := e.runeColToByteCol(end.Line, end.Col)
	endOffset := e.buf.LineOffset(end.Line) + endByte

	return e.buf.Text()[startOffset:endOffset]
}

// Copy copies selected text to internal clipboard.
func (e *EditorView) Copy() {
	text := e.SelectedText()
	if text != "" {
		e.clipboard = text
	}
}

// Cut copies selected text to clipboard and deletes it.
func (e *EditorView) Cut() {
	e.Copy()
	e.deleteSelectionIfAny()
}

// Paste inserts clipboard text at cursor position.
func (e *EditorView) Paste() {
	if e.clipboard == "" {
		return
	}
	e.deleteSelectionIfAny()
	byteCol := e.runeColToByteCol(e.cursor.Line, e.cursor.Col)
	e.buf.Insert(e.cursor.Line, byteCol, e.clipboard)

	// Move cursor to end of pasted text
	runes := []rune(e.clipboard)
	for _, r := range runes {
		if r == '\n' {
			e.cursor.Line++
			e.cursor.Col = 0
		} else {
			e.cursor.Col++
		}
	}
	e.ensureCursorVisible()
}

// Clipboard returns the internal clipboard contents.
func (e *EditorView) Clipboard() string {
	return e.clipboard
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
		e.cursor.Line--
		e.cursor.Col = e.runeLen(e.cursor.Line)
	}
	e.ensureCursorVisible()
}

// MoveCursorRight moves the cursor right one character.
func (e *EditorView) MoveCursorRight() {
	lineLen := e.runeLen(e.cursor.Line)
	if e.cursor.Col < lineLen {
		e.cursor.Col++
	} else if e.cursor.Line < e.buf.LineCount()-1 {
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
	e.cursor.Col = e.runeLen(e.cursor.Line)
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
		e.cursor.Col = e.runeLen(e.cursor.Line)
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
	lineLen := e.runeLen(e.cursor.Line)
	if e.cursor.Col > lineLen {
		e.cursor.Col = lineLen
	}
}

// runeLen returns the number of runes in a line.
func (e *EditorView) runeLen(line int) int {
	return utf8.RuneCountInString(e.buf.Line(line))
}

// runeColToByteCol converts a rune-based column to a byte offset within a line.
func (e *EditorView) runeColToByteCol(line, runeCol int) int {
	lineText := e.buf.Line(line)
	byteOffset := 0
	for i := 0; i < runeCol && byteOffset < len(lineText); i++ {
		_, size := utf8.DecodeRuneInString(lineText[byteOffset:])
		byteOffset += size
	}
	return byteOffset
}

// ensureCursorVisible adjusts scroll position to keep cursor visible.
func (e *EditorView) ensureCursorVisible() {
	bounds := e.Bounds()

	// Skip if bounds haven't been set yet
	if bounds.Width == 0 || bounds.Height == 0 {
		return
	}

	// Vertical scrolling
	if e.cursor.Line < e.scrollY {
		e.scrollY = e.cursor.Line
	} else if e.cursor.Line >= e.scrollY+bounds.Height {
		e.scrollY = e.cursor.Line - bounds.Height + 1
	}

	// Horizontal scrolling — based on display width, not rune count
	textAreaWidth := bounds.Width - e.lineNumWidth
	if textAreaWidth <= 0 {
		return
	}
	cursorDispX := e.runeDisplayWidth(e.cursor.Line, e.cursor.Col)
	if cursorDispX < e.scrollX {
		e.scrollX = cursorDispX
	} else if cursorDispX >= e.scrollX+textAreaWidth {
		e.scrollX = cursorDispX - textAreaWidth + 1
	}
}

// InsertChar inserts a character at the cursor position.
func (e *EditorView) InsertChar(ch rune) {
	if e.buf.ReadOnly {
		return
	}
	byteCol := e.runeColToByteCol(e.cursor.Line, e.cursor.Col)
	e.buf.Insert(e.cursor.Line, byteCol, string(ch))
	e.cursor.Col++
	e.ensureCursorVisible()
}

// InsertTab inserts a tab (as spaces) at the cursor position.
func (e *EditorView) InsertTab() {
	if e.buf.ReadOnly {
		return
	}
	tabSize := 4
	// Insert spaces to next tab stop
	col := e.cursor.Col
	spacesToInsert := tabSize - (col % tabSize)
	indent := ""
	for i := 0; i < spacesToInsert; i++ {
		indent += " "
	}
	byteCol := e.runeColToByteCol(e.cursor.Line, e.cursor.Col)
	e.buf.Insert(e.cursor.Line, byteCol, indent)
	e.cursor.Col += spacesToInsert
	e.ensureCursorVisible()
}

// InsertNewlineWithIndent inserts a newline and copies leading whitespace from current line.
func (e *EditorView) InsertNewlineWithIndent() {
	if e.buf.ReadOnly {
		return
	}
	// Get leading whitespace of current line
	lineText := e.buf.Line(e.cursor.Line)
	indent := ""
	for _, ch := range lineText {
		if ch == ' ' || ch == '\t' {
			indent += string(ch)
		} else {
			break
		}
	}

	byteCol := e.runeColToByteCol(e.cursor.Line, e.cursor.Col)
	e.buf.Insert(e.cursor.Line, byteCol, "\n"+indent)
	e.cursor.Line++
	e.cursor.Col = utf8.RuneCountInString(indent)
	e.ensureCursorVisible()
}

// InsertNewline inserts a newline at the cursor position.
func (e *EditorView) InsertNewline() {
	if e.buf.ReadOnly {
		return
	}
	byteCol := e.runeColToByteCol(e.cursor.Line, e.cursor.Col)
	e.buf.Insert(e.cursor.Line, byteCol, "\n")
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
		byteCol := e.runeColToByteCol(e.cursor.Line, e.cursor.Col-1)
		lineText := e.buf.Line(e.cursor.Line)
		_, size := utf8.DecodeRuneInString(lineText[byteCol:])
		e.buf.Delete(e.cursor.Line, byteCol, size)
		e.cursor.Col--
	} else if e.cursor.Line > 0 {
		prevRuneLen := e.runeLen(e.cursor.Line - 1)
		prevByteLen := len(e.buf.Line(e.cursor.Line - 1))
		e.buf.Delete(e.cursor.Line-1, prevByteLen, 1)
		e.cursor.Line--
		e.cursor.Col = prevRuneLen
	}
	e.ensureCursorVisible()
}

// DeleteForward deletes the character at the cursor position (delete key).
func (e *EditorView) DeleteForward() {
	if e.buf.ReadOnly {
		return
	}
	lineRuneLen := e.runeLen(e.cursor.Line)
	if e.cursor.Col < lineRuneLen {
		byteCol := e.runeColToByteCol(e.cursor.Line, e.cursor.Col)
		lineText := e.buf.Line(e.cursor.Line)
		_, size := utf8.DecodeRuneInString(lineText[byteCol:])
		e.buf.Delete(e.cursor.Line, byteCol, size)
	} else if e.cursor.Line < e.buf.LineCount()-1 {
		byteCol := e.runeColToByteCol(e.cursor.Line, e.cursor.Col)
		e.buf.Delete(e.cursor.Line, byteCol, 1)
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

// SetCursorPosition sets the cursor to the given position.
func (e *EditorView) SetCursorPosition(pos types.Position) {
	e.cursor = pos
	e.clampCursorCol()
	e.ensureCursorVisible()
}

// SetScrollY sets the vertical scroll offset.
func (e *EditorView) SetScrollY(y int) {
	e.scrollY = y
}

// SetSearchHighlights sets the search match highlights.
func (e *EditorView) SetSearchHighlights(highlights []SearchHighlight) {
	e.searchHighlights = highlights
}

// ClearSearchHighlights removes all search highlights.
func (e *EditorView) ClearSearchHighlights() {
	e.searchHighlights = nil
}

// HandleMouseClick moves the cursor to the screen position clicked.
func (e *EditorView) HandleMouseClick(screenX, screenY int) {
	bounds := e.Bounds()
	textAreaX := bounds.X + e.lineNumWidth

	// Calculate line from screen Y
	row := screenY - bounds.Y
	line := e.scrollY + row
	if line < 0 {
		line = 0
	}
	if line >= e.buf.LineCount() {
		line = e.buf.LineCount() - 1
	}

	// Calculate rune column from screen X
	clickDispCol := (screenX - textAreaX) + e.scrollX
	if clickDispCol < 0 {
		clickDispCol = 0
	}

	// Walk through the line to find the rune at this display column
	lineRunes := []rune(e.buf.Line(line))
	dispCol := 0
	runeCol := 0
	for i, ch := range lineRunes {
		var w int
		if ch == '\t' {
			w = tabWidth - (dispCol % tabWidth)
		} else {
			w = runewidth.RuneWidth(ch)
		}
		if dispCol+w > clickDispCol {
			runeCol = i
			break
		}
		dispCol += w
		runeCol = i + 1
	}

	e.ClearSelection()
	e.cursor = types.Position{Line: line, Col: runeCol}
	e.clampCursorCol()
	e.ensureCursorVisible()
}

// SetLanguage sets the language for syntax highlighting.
func (e *EditorView) SetLanguage(language string) {
	if e.theme == nil {
		e.highlighter = nil
		return
	}
	e.highlighter = syntax.NewHighlighter(e.theme, language)
}
