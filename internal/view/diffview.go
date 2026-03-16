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
	title       string // e.g. filename
}

// NewDiffView creates a DiffView from old and new text.
// filePath is used to detect the language for syntax highlighting.
func NewDiffView(theme *syntax.Theme, oldText, newText, title, filePath string) *DiffView {
	dv := &DiffView{
		theme: theme,
		title: title,
	}
	dv.lines = computeSideBySide(oldText, newText)

	// Detect language from file extension for syntax highlighting
	lang := detectLangFromPath(filePath)
	if lang != "" && lang != "text" {
		dv.highlighter = syntax.NewHighlighter(theme, lang)
	}

	return dv
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

	// Diff background colors (syntax highlighting provides foreground)
	addedBg := dv.theme.ResolveColor("#1e3a1e")
	removedBg := dv.theme.ResolveColor("#3a1e1e")

	// Clear
	for y := bounds.Y; y < bounds.Y+bounds.Height; y++ {
		for x := bounds.X; x < bounds.X+bounds.Width; x++ {
			screen.SetContent(x, y, ' ', nil, bgStyle)
		}
	}

	// Header row
	header := fmt.Sprintf(" Diff: %s ", dv.title)
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
		var leftBg, rightBg tcell.Color
		leftBg = tcell.ColorDefault
		rightBg = tcell.ColorDefault
		switch dl.Kind {
		case DiffAdded:
			rightBg = addedBg
		case DiffRemoved:
			leftBg = removedBg
		case DiffModified:
			leftBg = removedBg
			rightBg = addedBg
		}

		// Draw both sides with syntax highlighting + diff background
		dv.drawSide(screen, leftX, y, lineNumW, textW, dl.LeftNum, dl.LeftText, bgStyle, lineNumStyle, leftBg)
		dv.drawSide(screen, rightX, y, lineNumW, textW, dl.RightNum, dl.RightText, bgStyle, lineNumStyle, rightBg)
	}
}

func (dv *DiffView) drawSide(screen tcell.Screen, startX, y, lineNumW, textW, lineNum int, text string, baseStyle, numStyle tcell.Style, diffBg tcell.Color) {
	// Determine the row background
	rowStyle := baseStyle
	if diffBg != tcell.ColorDefault {
		rowStyle = baseStyle.Background(diffBg)
	}

	// Clear the side
	for x := startX; x < startX+lineNumW+textW; x++ {
		screen.SetContent(x, y, ' ', nil, rowStyle)
	}

	// Line number
	if lineNum > 0 {
		numStr := fmt.Sprintf("%*d ", lineNumW-1, lineNum)
		ns := numStyle
		if diffBg != tcell.ColorDefault {
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

	// Get syntax tokens for all lines (always highlight)
	var tokens []syntax.Token
	if dv.highlighter != nil && text != "" {
		tokens = dv.highlighter.HighlightLine(text)
	}

	// Text content
	tx := startX + lineNumW
	maxX := tx + textW
	runeIdx := 0
	for _, ch := range text {
		w := runewidth.RuneWidth(ch)
		if ch == '\t' {
			w = 4
			for j := 0; j < w && tx < maxX; j++ {
				screen.SetContent(tx, y, ' ', nil, rowStyle)
				tx++
			}
			runeIdx++
			continue
		}
		if tx+w > maxX {
			break
		}

		style := rowStyle
		// Apply syntax highlighting with diff background preserved
		if len(tokens) > 0 {
			for _, token := range tokens {
				if runeIdx >= token.Start && runeIdx < token.Start+token.Length {
					tokenStyle := dv.highlighter.StyleForToken(token.Type)
					fg, _, _ := tokenStyle.Decompose()
					if diffBg != tcell.ColorDefault {
						style = tcell.StyleDefault.Foreground(fg).Background(diffBg)
					} else {
						_, bg, _ := baseStyle.Decompose()
						style = tcell.StyleDefault.Foreground(fg).Background(bg)
					}
					break
				}
			}
		}

		screen.SetContent(tx, y, ch, nil, style)
		tx += w
		runeIdx++
	}
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

	switch ev.Key() {
	case tcell.KeyUp:
		if dv.scrollY > 0 {
			dv.scrollY--
		}
		return true
	case tcell.KeyDown:
		if dv.scrollY < maxScroll {
			dv.scrollY++
		}
		return true
	case tcell.KeyPgUp:
		dv.scrollY -= contentHeight
		if dv.scrollY < 0 {
			dv.scrollY = 0
		}
		return true
	case tcell.KeyPgDn:
		dv.scrollY += contentHeight
		if dv.scrollY > maxScroll {
			dv.scrollY = maxScroll
		}
		return true
	case tcell.KeyHome:
		dv.scrollY = 0
		return true
	case tcell.KeyEnd:
		dv.scrollY = maxScroll
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
func computeSideBySide(oldText, newText string) []DiffLine {
	oldLines := splitLines(oldText)
	newLines := splitLines(newText)

	// Simple diff using LCS (Longest Common Subsequence)
	ops := diffOps(oldLines, newLines)

	var result []DiffLine
	oi, ni := 0, 0
	for _, op := range ops {
		switch op {
		case opEqual:
			result = append(result, DiffLine{
				LeftNum:   oi + 1,
				LeftText:  oldLines[oi],
				RightNum:  ni + 1,
				RightText: newLines[ni],
				Kind:      DiffEqual,
			})
			oi++
			ni++
		case opRemove:
			result = append(result, DiffLine{
				LeftNum:  oi + 1,
				LeftText: oldLines[oi],
				Kind:     DiffRemoved,
			})
			oi++
		case opAdd:
			result = append(result, DiffLine{
				RightNum:  ni + 1,
				RightText: newLines[ni],
				Kind:      DiffAdded,
			})
			ni++
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
