package view

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/JiHyeongSeo/ted/internal/syntax"
)

// DiffLineKind indicates the type of a diff line.
type DiffLineKind int

const (
	DiffEqual    DiffLineKind = iota
	DiffAdded                         // only in right (new)
	DiffRemoved                       // only in left (old)
	DiffModified                      // changed between left and right
)

// DiffLine represents a single line in the side-by-side diff.
type DiffLine struct {
	LeftNum  int    // 1-based line number, 0 = blank
	LeftText string
	RightNum int
	RightText string
	Kind     DiffLineKind
}

// DiffView renders a side-by-side diff of two texts.
type DiffView struct {
	BaseComponent
	theme       *syntax.Theme
	highlighter *syntax.Highlighter
	lines       []DiffLine
	scrollY     int
	scrollX     int // horizontal scroll offset (in columns)
	cursorY     int // absolute line index of cursor (-1 = none)
	title       string // e.g. filename
	// Selection state for copy
	selStart    int // line index of selection start (-1 = none)
	selEnd      int // line index of selection end
	selSide     int // 1 = left, 2 = right
	selecting   bool
	onCopy      func(text string)
}

// NewDiffView creates a DiffView from old and new text.
// filePath is used to detect the language for syntax highlighting.
func NewDiffView(theme *syntax.Theme, oldText, newText, title, filePath string) *DiffView {
	dv := &DiffView{
		theme:   theme,
		title:   title,
		cursorY: 0,
	}
	dv.lines = computeSideBySide(oldText, newText)

	// Detect language from file extension for syntax highlighting
	lang := detectLangFromPath(filePath)
	if lang != "" && lang != "text" {
		dv.highlighter = syntax.NewHighlighter(theme, lang)
	}

	dv.selStart = -1
	return dv
}

// SetOnCopy sets the callback for copying selected text.
func (dv *DiffView) SetOnCopy(fn func(text string)) {
	dv.onCopy = fn
}

// SelectedText returns the selected diff text for the side the user dragged on.
func (dv *DiffView) SelectedText() string {
	if dv.selStart < 0 {
		return ""
	}
	start, end := dv.selStart, dv.selEnd
	if start > end {
		start, end = end, start
	}
	var lines []string
	for i := start; i <= end && i < len(dv.lines); i++ {
		dl := dv.lines[i]
		if dv.selSide == 1 {
			if dl.LeftText != "" {
				lines = append(lines, dl.LeftText)
			}
		} else {
			if dl.RightText != "" {
				lines = append(lines, dl.RightText)
			} else if dl.LeftText != "" {
				lines = append(lines, dl.LeftText)
			}
		}
	}
	return strings.Join(lines, "\n")
}

// LineCount returns the total number of display lines.
func (dv *DiffView) LineCount() int {
	return len(dv.lines)
}

// Render draws the side-by-side diff view.
func (dv *DiffView) Render(screen tcell.Screen) {
	bounds := dv.Bounds()
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return
	}

	bgStyle := dv.theme.UIStyle("default")
	headerStyle := bgStyle.Foreground(dv.theme.ResolveColor("#c8c8c8")).Bold(true)
	sepStyle := bgStyle.Foreground(dv.theme.ResolveColor("#505050"))
	lineNumStyle := dv.theme.UIStyle("linenumber")
	cursorStyle := bgStyle.Background(dv.theme.ResolveColor("#264f78")) // VS Code blue cursor line

	// Diff background colors using 256-color palette for broad terminal support
	// Color 22 = dark green, Color 52 = dark red
	addedBg := tcell.PaletteColor(22)
	removedBg := tcell.PaletteColor(52)

	// Clear
	for y := bounds.Y; y < bounds.Y+bounds.Height; y++ {
		for x := bounds.X; x < bounds.X+bounds.Width; x++ {
			screen.SetContent(x, y, ' ', nil, bgStyle)
		}
	}

	// Header row — show scrollX offset if non-zero
	scrollHint := ""
	if dv.scrollX > 0 {
		scrollHint = fmt.Sprintf("  [←→ col:%d]", dv.scrollX)
	}
	header := fmt.Sprintf(" Diff: %s%s ", dv.title, scrollHint)
	hx := bounds.X
	for _, ch := range header {
		if hx >= bounds.X+bounds.Width {
			break
		}
		screen.SetContent(hx, bounds.Y, ch, nil, headerStyle)
		hx++
	}
	for x := hx; x < bounds.X+bounds.Width; x++ {
		screen.SetContent(x, bounds.Y, '─', nil, sepStyle)
	}

	// Calculate column widths
	contentHeight := bounds.Height - 1 // minus header
	halfWidth := (bounds.Width - 1) / 2 // -1 for center separator
	lineNumW := 5 // width for line numbers
	textW := halfWidth - lineNumW
	if textW < 1 {
		textW = 1
	}
	leftX := bounds.X
	sepX := bounds.X + halfWidth
	rightX := sepX + 1

	// Draw diff lines
	for i := 0; i < contentHeight; i++ {
		idx := dv.scrollY + i
		y := bounds.Y + 1 + i

		// Draw center separator
		screen.SetContent(sepX, y, '│', nil, sepStyle)

		if idx >= len(dv.lines) {
			continue
		}
		dl := dv.lines[idx]

		// Determine background color for each side (syntax highlighting always on)
		leftHasDiff := false
		rightHasDiff := false
		var leftBg, rightBg tcell.Color
		switch dl.Kind {
		case DiffAdded:
			rightBg = addedBg
			rightHasDiff = true
		case DiffRemoved:
			leftBg = removedBg
			leftHasDiff = true
		case DiffModified:
			leftBg = removedBg
			leftHasDiff = true
			rightBg = addedBg
			rightHasDiff = true
		}

		// Check if this line is selected
		isSelected := false
		if dv.selStart >= 0 {
			sStart, sEnd := dv.selStart, dv.selEnd
			if sStart > sEnd {
				sStart, sEnd = sEnd, sStart
			}
			isSelected = idx >= sStart && idx <= sEnd
		}

		// Cursor line overrides background when not already diff-colored
		isCursor := idx == dv.cursorY

		// Draw both sides
		dv.drawSide(screen, leftX, y, lineNumW, textW, dl.LeftNum, dl.LeftText, bgStyle, lineNumStyle,
			leftHasDiff, leftBg, isSelected && dv.selSide == 1, isCursor && !leftHasDiff, cursorStyle, dv.scrollX)
		dv.drawSide(screen, rightX, y, lineNumW, textW, dl.RightNum, dl.RightText, bgStyle, lineNumStyle,
			rightHasDiff, rightBg, isSelected && dv.selSide == 2, isCursor && !rightHasDiff, cursorStyle, dv.scrollX)
	}
}

func (dv *DiffView) drawSide(screen tcell.Screen, startX, y, lineNumW, textW, lineNum int, text string,
	baseStyle, numStyle tcell.Style, hasDiff bool, diffBg tcell.Color, isSelected bool, isCursor bool, cursorStyle tcell.Style, scrollX int) {
	// Determine the row background priority: selection > cursor > diff > base
	rowStyle := baseStyle
	if isCursor {
		rowStyle = cursorStyle
	}
	if hasDiff {
		rowStyle = baseStyle.Background(diffBg)
	}
	if isSelected {
		rowStyle = rowStyle.Reverse(true)
	}

	// Clear the side
	for x := startX; x < startX+lineNumW+textW; x++ {
		screen.SetContent(x, y, ' ', nil, rowStyle)
	}

	// Line number (always unaffected by scrollX)
	if lineNum > 0 {
		numStr := fmt.Sprintf("%*d ", lineNumW-1, lineNum)
		ns := numStyle
		if isCursor {
			ns = numStyle.Background(dv.cursorBg(cursorStyle))
		}
		if hasDiff {
			ns = numStyle.Background(diffBg)
		}
		nx := startX
		for _, ch := range numStr {
			if nx >= startX+lineNumW {
				break
			}
			screen.SetContent(nx, y, ch, nil, ns)
			nx++
		}
	} else {
		for nx := startX; nx < startX+lineNumW; nx++ {
			screen.SetContent(nx, y, ' ', nil, rowStyle)
		}
	}

	// Get syntax tokens
	var tokens []syntax.Token
	if dv.highlighter != nil && text != "" {
		tokens = dv.highlighter.HighlightLine(text)
	}

	// Build rune list with widths, applying scrollX offset
	tx := startX + lineNumW
	maxX := tx + textW
	col := 0      // visual column (accounting for tab expansion)
	runeIdx := 0
	for _, ch := range text {
		w := runewidth.RuneWidth(ch)
		if ch == '\t' {
			w = 4
		}
		// Skip columns before scrollX
		if col < scrollX {
			col += w
			runeIdx++
			continue
		}
		if tx+w > maxX {
			break
		}

		style := rowStyle
		if len(tokens) > 0 {
			for _, token := range tokens {
				if runeIdx >= token.Start && runeIdx < token.Start+token.Length {
					tokenStyle := dv.highlighter.StyleForToken(token.Type)
					fg, _, _ := tokenStyle.Decompose()
					if hasDiff {
						style = tcell.StyleDefault.Foreground(fg).Background(diffBg)
					} else {
						_, bg, _ := rowStyle.Decompose()
						style = tcell.StyleDefault.Foreground(fg).Background(bg)
					}
					break
				}
			}
		}

		if ch == '\t' {
			for j := 0; j < 4 && tx < maxX; j++ {
				screen.SetContent(tx, y, ' ', nil, style)
				tx++
			}
		} else {
			screen.SetContent(tx, y, ch, nil, style)
			tx += w
		}
		col += w
		runeIdx++
	}
}

// cursorBg extracts the background color from cursorStyle.
func (dv *DiffView) cursorBg(cursorStyle tcell.Style) tcell.Color {
	_, bg, _ := cursorStyle.Decompose()
	return bg
}

// HandleEvent processes key/mouse events for scrolling.
func (dv *DiffView) HandleEvent(ev tcell.Event) bool {
	switch tev := ev.(type) {
	case *tcell.EventKey:
		return dv.handleKey(tev)
	case *tcell.EventMouse:
		return dv.handleMouse(tev)
	}
	return false
}

func (dv *DiffView) handleKey(ev *tcell.EventKey) bool {
	contentHeight := dv.Bounds().Height - 1
	maxScroll := len(dv.lines) - contentHeight
	if maxScroll < 0 {
		maxScroll = 0
	}

	// Ctrl+C to copy selection
	if ev.Key() == tcell.KeyRune && ev.Modifiers()&tcell.ModCtrl != 0 && ev.Rune() == 'c' {
		if dv.selStart >= 0 && dv.onCopy != nil {
			dv.onCopy(dv.SelectedText())
		}
		return true
	}

	switch ev.Key() {
	case tcell.KeyUp:
		if dv.cursorY > 0 {
			dv.cursorY--
		}
		// Scroll up if cursor above viewport
		if dv.cursorY < dv.scrollY {
			dv.scrollY = dv.cursorY
		}
		return true
	case tcell.KeyDown:
		if dv.cursorY < len(dv.lines)-1 {
			dv.cursorY++
		}
		// Scroll down if cursor below viewport
		if dv.cursorY >= dv.scrollY+contentHeight {
			dv.scrollY = dv.cursorY - contentHeight + 1
		}
		return true
	case tcell.KeyLeft:
		dv.scrollX -= 8
		if dv.scrollX < 0 {
			dv.scrollX = 0
		}
		return true
	case tcell.KeyRight:
		dv.scrollX += 8
		return true
	case tcell.KeyPgUp:
		dv.scrollY -= contentHeight
		if dv.scrollY < 0 {
			dv.scrollY = 0
		}
		dv.cursorY = dv.scrollY
		return true
	case tcell.KeyPgDn:
		dv.scrollY += contentHeight
		if dv.scrollY > maxScroll {
			dv.scrollY = maxScroll
		}
		dv.cursorY = dv.scrollY
		return true
	case tcell.KeyHome:
		dv.scrollY = 0
		dv.cursorY = 0
		dv.scrollX = 0
		return true
	case tcell.KeyEnd:
		dv.scrollY = maxScroll
		dv.cursorY = len(dv.lines) - 1
		return true
	}
	return false
}

func (dv *DiffView) handleMouse(ev *tcell.EventMouse) bool {
	bounds := dv.Bounds()
	mx, my := ev.Position()
	if mx < bounds.X || mx >= bounds.X+bounds.Width || my < bounds.Y || my >= bounds.Y+bounds.Height {
		return false
	}

	contentHeight := bounds.Height - 1
	maxScroll := len(dv.lines) - contentHeight
	if maxScroll < 0 {
		maxScroll = 0
	}

	// Mouse click/drag — start or extend selection
	halfWidth := (bounds.Width - 1) / 2
	sepX := bounds.X + halfWidth

	if ev.Buttons()&tcell.Button1 != 0 {
		row := my - bounds.Y - 1 // -1 for header
		if row >= 0 {
			lineIdx := dv.scrollY + row
			if lineIdx < len(dv.lines) {
				dv.cursorY = lineIdx
				if !dv.selecting {
					dv.selecting = true
					dv.selStart = lineIdx
					dv.selEnd = lineIdx
					if mx < sepX {
						dv.selSide = 1 // left
					} else {
						dv.selSide = 2 // right
					}
				} else {
					dv.selEnd = lineIdx
				}
			}
		}
		return true
	}

	// Button released — end drag
	if ev.Buttons() == tcell.ButtonNone {
		dv.selecting = false
	}

	if ev.Buttons()&tcell.WheelUp != 0 {
		dv.scrollY -= 3
		if dv.scrollY < 0 {
			dv.scrollY = 0
		}
		return true
	}
	if ev.Buttons()&tcell.WheelDown != 0 {
		dv.scrollY += 3
		if dv.scrollY > maxScroll {
			dv.scrollY = maxScroll
		}
		return true
	}
	return false
}

// detectLangFromPath returns a language identifier from a file path extension.
func detectLangFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".rs":
		return "rust"
	case ".c", ".h":
		return "c"
	case ".cpp", ".hpp", ".cc":
		return "cpp"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	case ".sh", ".bash":
		return "bash"
	case ".yaml", ".yml":
		return "yaml"
	default:
		return "text"
	}
}

// computeSideBySide builds side-by-side diff lines using a simple LCS-based diff.
// Consecutive remove+add blocks are paired as DiffModified so they appear on the
// same row (aligned), instead of being shown on separate rows.
func computeSideBySide(oldText, newText string) []DiffLine {
	oldLines := splitLines(oldText)
	newLines := splitLines(newText)

	ops := diffOps(oldLines, newLines)

	var result []DiffLine
	oi, ni := 0, 0
	i := 0
	for i < len(ops) {
		if ops[i] == opEqual {
			result = append(result, DiffLine{
				LeftNum:   oi + 1,
				LeftText:  oldLines[oi],
				RightNum:  ni + 1,
				RightText: newLines[ni],
				Kind:      DiffEqual,
			})
			oi++
			ni++
			i++
			continue
		}

		// Collect a contiguous block of removes and adds (in any order).
		var removeIdxs, addIdxs []int
		for i < len(ops) && (ops[i] == opRemove || ops[i] == opAdd) {
			if ops[i] == opRemove {
				removeIdxs = append(removeIdxs, oi)
				oi++
			} else {
				addIdxs = append(addIdxs, ni)
				ni++
			}
			i++
		}

		// Pair removes with adds as DiffModified for visual alignment.
		pairCount := len(removeIdxs)
		if len(addIdxs) < pairCount {
			pairCount = len(addIdxs)
		}
		for k := 0; k < pairCount; k++ {
			result = append(result, DiffLine{
				LeftNum:   removeIdxs[k] + 1,
				LeftText:  oldLines[removeIdxs[k]],
				RightNum:  addIdxs[k] + 1,
				RightText: newLines[addIdxs[k]],
				Kind:      DiffModified,
			})
		}
		// Extra removes (more deleted than added)
		for k := pairCount; k < len(removeIdxs); k++ {
			result = append(result, DiffLine{
				LeftNum:  removeIdxs[k] + 1,
				LeftText: oldLines[removeIdxs[k]],
				Kind:     DiffRemoved,
			})
		}
		// Extra adds (more added than deleted)
		for k := pairCount; k < len(addIdxs); k++ {
			result = append(result, DiffLine{
				RightNum:  addIdxs[k] + 1,
				RightText: newLines[addIdxs[k]],
				Kind:      DiffAdded,
			})
		}
	}
	return result
}

func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	// Remove trailing empty line from final newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

type diffOp int

const (
	opEqual  diffOp = iota
	opAdd
	opRemove
)

// diffOps computes edit operations using Myers' diff algorithm (simplified).
func diffOps(a, b []string) []diffOp {
	n, m := len(a), len(b)

	// Build LCS table
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Backtrack to get operations
	var ops []diffOp
	i, j := n, m
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && a[i-1] == b[j-1] {
			ops = append(ops, opEqual)
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			ops = append(ops, opAdd)
			j--
		} else {
			ops = append(ops, opRemove)
			i--
		}
	}

	// Reverse
	for l, r := 0, len(ops)-1; l < r; l, r = l+1, r-1 {
		ops[l], ops[r] = ops[r], ops[l]
	}
	return ops
}
