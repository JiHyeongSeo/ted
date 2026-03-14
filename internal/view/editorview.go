package view

import (
	"fmt"
	"unicode"
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
	highlighter      *syntax.Highlighter   // keyword-based fallback
	tsHighlighter    *syntax.TSHighlighter // tree-sitter highlighter
	cursor           types.Position        // cursor position (rune-based col)
	scrollY          int                   // first visible line
	scrollX          int                   // horizontal scroll offset (display columns)
	selection        *types.Selection      // current selection range
	lineNumWidth     int                   // width of line number gutter
	cursorScreenX    int                   // last computed screen X of cursor (for ShowCursor)
	cursorScreenY    int                   // last computed screen Y of cursor
	clipboard        string                // internal clipboard
	searchHighlights []SearchHighlight     // search match highlights
	gutterMarkers    map[int]types.GutterMark
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
		// Apply git gutter marker background
		if mark, ok := e.gutterMarkers[lineNum]; ok && mark != types.MarkNone {
			var colorKey string
			switch mark {
			case types.MarkAdded:
				colorKey = "gitAdded"
			case types.MarkModified:
				colorKey = "gitModified"
			case types.MarkDeleted:
				colorKey = "gitDeleted"
			}
			if colorKey != "" {
				if hex := e.theme.UI[colorKey]; hex != "" {
					lineNumStyle = lineNumStyle.Background(e.theme.ResolveColor(hex)).
						Foreground(e.theme.ResolveColor(e.theme.UI["lineNumberActive"]))
				}
			}
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

		// Get syntax highlighting tokens (prefer tree-sitter)
		var tokens []syntax.Token
		if e.tsHighlighter != nil {
			tokens = e.tsHighlighter.HighlightLine(lineNum)
		} else if e.highlighter != nil {
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
			for _, token := range tokens {
				if runeIdx >= token.Start && runeIdx < token.Start+token.Length {
					style = e.theme.TokenStyle(string(token.Type))
					break
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
		ctrl := mod&tcell.ModCtrl != 0
		alt := mod&tcell.ModAlt != 0
		if shift {
			e.startOrExtendSelection()
		} else {
			e.ClearSelection()
		}
		if ctrl && shift {
			e.MoveCursorToLineStart()
		} else if ctrl {
			e.MoveCursorWordLeft()
		} else if alt {
			e.MoveCursorToLineStart()
		} else {
			e.MoveCursorLeft()
		}
		if shift {
			e.ExtendSelection()
		}
		return true
	case tcell.KeyRight:
		ctrl := mod&tcell.ModCtrl != 0
		alt := mod&tcell.ModAlt != 0
		if shift {
			e.startOrExtendSelection()
		} else {
			e.ClearSelection()
		}
		if ctrl && shift {
			e.MoveCursorToLineEnd()
		} else if ctrl {
			e.MoveCursorWordRight()
		} else if alt {
			e.MoveCursorToLineEnd()
		} else {
			e.MoveCursorRight()
		}
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
	case tcell.KeyPgUp:
		if shift {
			e.startOrExtendSelection()
		} else {
			e.ClearSelection()
		}
		e.MoveCursorPageUp()
		if shift {
			e.ExtendSelection()
		}
		return true
	case tcell.KeyPgDn:
		if shift {
			e.startOrExtendSelection()
		} else {
			e.ClearSelection()
		}
		e.MoveCursorPageDown()
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

// MoveCursorPageUp moves the cursor up by one page.
func (e *EditorView) MoveCursorPageUp() {
	pageSize := e.Bounds().Height
	if pageSize < 1 {
		pageSize = 1
	}
	e.cursor.Line -= pageSize
	if e.cursor.Line < 0 {
		e.cursor.Line = 0
	}
	e.scrollY -= pageSize
	if e.scrollY < 0 {
		e.scrollY = 0
	}
	e.clampCursorCol()
	e.ensureCursorVisible()
}

// MoveCursorPageDown moves the cursor down by one page.
func (e *EditorView) MoveCursorPageDown() {
	pageSize := e.Bounds().Height
	if pageSize < 1 {
		pageSize = 1
	}
	e.cursor.Line += pageSize
	lastLine := e.buf.LineCount() - 1
	if lastLine < 0 {
		lastLine = 0
	}
	if e.cursor.Line > lastLine {
		e.cursor.Line = lastLine
	}
	e.scrollY += pageSize
	e.clampCursorCol()
	e.ensureCursorVisible()
}

// ScrollUp scrolls the view up by n lines without moving the cursor.
func (e *EditorView) ScrollUp(n int) {
	e.scrollY -= n
	if e.scrollY < 0 {
		e.scrollY = 0
	}
}

// ScrollDown scrolls the view down by n lines without moving the cursor.
func (e *EditorView) ScrollDown(n int) {
	maxScroll := e.buf.LineCount() - 1
	if maxScroll < 0 {
		maxScroll = 0
	}
	e.scrollY += n
	if e.scrollY > maxScroll {
		e.scrollY = maxScroll
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

// MoveCursorWordLeft moves the cursor left by one word.
func (e *EditorView) MoveCursorWordLeft() {
	runes := []rune(e.buf.Line(e.cursor.Line))
	col := e.cursor.Col

	if col == 0 {
		// Move to end of previous line
		if e.cursor.Line > 0 {
			e.cursor.Line--
			e.cursor.Col = e.runeLen(e.cursor.Line)
		}
		e.ensureCursorVisible()
		return
	}

	// Skip whitespace backward
	for col > 0 && col <= len(runes) && unicode.IsSpace(runes[col-1]) {
		col--
	}
	// Skip word characters backward
	if col > 0 && col <= len(runes) && unicode.IsPunct(runes[col-1]) {
		for col > 0 && col <= len(runes) && unicode.IsPunct(runes[col-1]) {
			col--
		}
	} else {
		for col > 0 && col <= len(runes) && !unicode.IsSpace(runes[col-1]) && !unicode.IsPunct(runes[col-1]) {
			col--
		}
	}

	e.cursor.Col = col
	e.ensureCursorVisible()
}

// MoveCursorWordRight moves the cursor right by one word.
func (e *EditorView) MoveCursorWordRight() {
	runes := []rune(e.buf.Line(e.cursor.Line))
	col := e.cursor.Col
	lineLen := len(runes)

	if col >= lineLen {
		// Move to start of next line
		if e.cursor.Line < e.buf.LineCount()-1 {
			e.cursor.Line++
			e.cursor.Col = 0
		}
		e.ensureCursorVisible()
		return
	}

	// Skip current word characters forward
	if unicode.IsPunct(runes[col]) {
		for col < lineLen && unicode.IsPunct(runes[col]) {
			col++
		}
	} else {
		for col < lineLen && !unicode.IsSpace(runes[col]) && !unicode.IsPunct(runes[col]) {
			col++
		}
	}
	// Skip whitespace forward
	for col < lineLen && unicode.IsSpace(runes[col]) {
		col++
	}

	e.cursor.Col = col
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

// reparseTS re-parses the buffer with tree-sitter after edits.
func (e *EditorView) reparseTS() {
	if e.tsHighlighter != nil {
		e.tsHighlighter.Parse([]byte(e.buf.Text()))
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
	e.reparseTS()
	e.ensureCursorVisible()
}

// InsertTab inserts a tab (as spaces) at the cursor position.
func (e *EditorView) InsertTab() {
	if e.buf.ReadOnly {
		return
	}
	tabSize := 4
	col := e.cursor.Col
	spacesToInsert := tabSize - (col % tabSize)
	indent := ""
	for i := 0; i < spacesToInsert; i++ {
		indent += " "
	}
	byteCol := e.runeColToByteCol(e.cursor.Line, e.cursor.Col)
	e.buf.Insert(e.cursor.Line, byteCol, indent)
	e.cursor.Col += spacesToInsert
	e.reparseTS()
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
	e.reparseTS()
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
	e.reparseTS()
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
	e.reparseTS()
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
	e.reparseTS()
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

// ScreenYToLine converts a screen Y coordinate to a buffer line number.
func (e *EditorView) ScreenYToLine(screenY int) int {
	bounds := e.Bounds()
	row := screenY - bounds.Y
	return e.scrollY + row
}

// ScreenXToCol converts a screen X coordinate to a rune column for a given screen Y.
func (e *EditorView) ScreenXToCol(screenX int) int {
	bounds := e.Bounds()
	textAreaX := bounds.X + e.lineNumWidth
	clickDispCol := (screenX - textAreaX) + e.scrollX
	if clickDispCol < 0 {
		return 0
	}

	// Determine the line from cursor position context
	// Use the last known line from screen position
	row := e.scrollY + (e.cursorScreenY - bounds.Y)
	if row < 0 || row >= e.buf.LineCount() {
		return 0
	}

	lineRunes := []rune(e.buf.Line(row))
	dispCol := 0
	for i, ch := range lineRunes {
		var w int
		if ch == '\t' {
			w = tabWidth - (dispCol % tabWidth)
		} else {
			w = runewidth.RuneWidth(ch)
		}
		if dispCol+w > clickDispCol {
			return i
		}
		dispCol += w
	}
	return len(lineRunes)
}

// ScreenXToColForLine converts a screen X coordinate to a rune column for a specific line.
func (e *EditorView) ScreenXToColForLine(screenX, line int) int {
	bounds := e.Bounds()
	textAreaX := bounds.X + e.lineNumWidth
	clickDispCol := (screenX - textAreaX) + e.scrollX
	if clickDispCol < 0 {
		return 0
	}
	if line < 0 || line >= e.buf.LineCount() {
		return 0
	}

	lineRunes := []rune(e.buf.Line(line))
	dispCol := 0
	for i, ch := range lineRunes {
		var w int
		if ch == '\t' {
			w = tabWidth - (dispCol % tabWidth)
		} else {
			w = runewidth.RuneWidth(ch)
		}
		if dispCol+w > clickDispCol {
			return i
		}
		dispCol += w
	}
	return len(lineRunes)
}

// CursorScreenX returns the last computed screen X of the cursor relative to editor bounds.
func (e *EditorView) CursorScreenX() int {
	return e.cursorScreenX - e.Bounds().X
}

// CursorScreenY returns the last computed screen Y of the cursor.
func (e *EditorView) CursorScreenY() int {
	return e.cursorScreenY
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

// SetGutterMarkers sets the git diff gutter markers.
func (e *EditorView) SetGutterMarkers(markers map[int]types.GutterMark) {
	e.gutterMarkers = markers
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
		e.tsHighlighter = nil
		return
	}
	// Try tree-sitter first, fall back to keyword-based
	if syntax.TSSupported(language) {
		e.tsHighlighter = syntax.NewTSHighlighter(e.theme, language)
		if e.tsHighlighter != nil {
			e.tsHighlighter.Parse([]byte(e.buf.Text()))
		}
		e.highlighter = nil
	} else {
		e.tsHighlighter = nil
		e.highlighter = syntax.NewHighlighter(e.theme, language)
	}
}
