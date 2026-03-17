package view

import (
	"encoding/csv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/JiHyeongSeo/ted/internal/syntax"
	"github.com/JiHyeongSeo/ted/internal/types"
)

// CSVView renders CSV content as a formatted table.
type CSVView struct {
	BaseComponent
	theme       *syntax.Theme
	records     [][]string // all rows (including header)
	colWidths   []int      // max width per column
	scrollY     int        // first visible row (excluding header)
	scrollX     int        // horizontal column offset
	cursorRow   int        // selected data row (0-based, excluding header)
	title       string
}

// NewCSVView parses CSV content and returns a CSVView.
func NewCSVView(theme *syntax.Theme, content, title string) *CSVView {
	cv := &CSVView{theme: theme, title: title}
	r := csv.NewReader(strings.NewReader(content))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		// Fallback: split by lines/commas naively
		records = nil
		for _, line := range strings.Split(content, "\n") {
			if line != "" {
				records = append(records, strings.Split(line, ","))
			}
		}
	}
	cv.records = records
	cv.computeColWidths()
	return cv
}

func (cv *CSVView) computeColWidths() {
	if len(cv.records) == 0 {
		return
	}
	maxCols := 0
	for _, row := range cv.records {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}
	cv.colWidths = make([]int, maxCols)
	for _, row := range cv.records {
		for j, cell := range row {
			w := runewidth.StringWidth(cell)
			if w > cv.colWidths[j] {
				cv.colWidths[j] = w
			}
		}
	}
	// Cap each column at 40 chars for readability
	for i := range cv.colWidths {
		if cv.colWidths[i] < 2 {
			cv.colWidths[i] = 2
		}
		if cv.colWidths[i] > 40 {
			cv.colWidths[i] = 40
		}
	}
}

func (cv *CSVView) RowCount() int {
	if len(cv.records) <= 1 {
		return 0
	}
	return len(cv.records) - 1 // exclude header
}

func (cv *CSVView) Render(screen tcell.Screen) {
	bounds := cv.Bounds()
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return
	}

	baseStyle := cv.theme.UIStyle("default")
	headerStyle := baseStyle.Bold(true).Foreground(cv.theme.ResolveColor("#c8c8c8"))
	sepStyle := baseStyle.Foreground(cv.theme.ResolveColor("#505050"))
	cursorStyle := baseStyle.Background(cv.theme.ResolveColor("#264f78"))
	altStyle := baseStyle.Background(cv.theme.ResolveColor("#1e1e2e"))
	titleStyle := baseStyle.Foreground(cv.theme.ResolveColor("#c8c8c8")).Bold(true)

	// Clear
	for y := bounds.Y; y < bounds.Y+bounds.Height; y++ {
		for x := bounds.X; x < bounds.X+bounds.Width; x++ {
			screen.SetContent(x, y, ' ', nil, baseStyle)
		}
	}

	if len(cv.records) == 0 {
		drawStr(screen, bounds.X+1, bounds.Y, "(empty)", baseStyle)
		return
	}

	// Title bar
	titleStr := " CSV: " + cv.title
	if cv.scrollX > 0 {
		titleStr += " [→col:" + itoa(cv.scrollX) + "]"
	}
	drawStr(screen, bounds.X, bounds.Y, titleStr, titleStyle)
	for x := bounds.X + runewidth.StringWidth(titleStr); x < bounds.X+bounds.Width; x++ {
		screen.SetContent(x, bounds.Y, '─', nil, sepStyle)
	}

	// Compute visible columns based on scrollX
	colOffsets := cv.buildColOffsets()
	contentH := bounds.Height - 1 // minus title bar

	// Row 1: header (always visible)
	headerY := bounds.Y + 1
	if len(cv.records) > 0 {
		cv.renderRow(screen, bounds, headerY, cv.records[0], colOffsets, headerStyle, false)
		// Separator under header
		for x := bounds.X; x < bounds.X+bounds.Width; x++ {
			screen.SetContent(x, headerY+1, '─', nil, sepStyle)
		}
	}

	// Data rows (starting at row 2 on screen)
	dataStartY := bounds.Y + 3 // title + header + separator
	visibleRows := contentH - 3
	if visibleRows < 1 {
		return
	}

	for i := 0; i < visibleRows; i++ {
		recIdx := cv.scrollY + i + 1 // +1 to skip header record
		if recIdx >= len(cv.records) {
			break
		}
		y := dataStartY + i
		isCursor := (cv.scrollY+i) == cv.cursorRow
		rowStyle := baseStyle
		if isCursor {
			rowStyle = cursorStyle
		} else if i%2 == 1 {
			rowStyle = altStyle
		}
		cv.renderRow(screen, bounds, y, cv.records[recIdx], colOffsets, rowStyle, isCursor)
	}

	// Status: row/total
	statusStr := " " + itoa(cv.cursorRow+1) + "/" + itoa(len(cv.records)-1) + " rows"
	statusStyle := baseStyle.Foreground(cv.theme.ResolveColor("#808080"))
	drawStr(screen, bounds.X, bounds.Y+bounds.Height-1, statusStr, statusStyle)
}

// buildColOffsets computes screen x offsets for each column, accounting for scrollX.
func (cv *CSVView) buildColOffsets() []int {
	offsets := make([]int, len(cv.colWidths))
	x := 0
	for i, w := range cv.colWidths {
		offsets[i] = x - cv.scrollX*3 // scrollX shifts by ~3 cols at a time (rough)
		x += w + 3                     // col width + " │ " separator
	}
	return offsets
}

func (cv *CSVView) renderRow(screen tcell.Screen, bounds types.Rect, y int, row []string, colOffsets []int, style tcell.Style, isCursor bool) {
	// Fill row background
	for x := bounds.X; x < bounds.X+bounds.Width; x++ {
		screen.SetContent(x, y, ' ', nil, style)
	}

	sepStyle := style.Foreground(cv.theme.ResolveColor("#505050"))

	for j, offset := range colOffsets {
		sx := bounds.X + 1 + offset
		if sx >= bounds.X+bounds.Width {
			break
		}

		// Draw separator before each column except first
		if j > 0 {
			sepX := sx - 2
			if sepX >= bounds.X && sepX < bounds.X+bounds.Width {
				screen.SetContent(sepX, y, '│', nil, sepStyle)
			}
		}

		if sx < bounds.X {
			continue
		}

		cell := ""
		if j < len(row) {
			cell = row[j]
		}
		maxW := cv.colWidths[j]
		if j < len(cv.colWidths) {
			maxW = cv.colWidths[j]
		}

		cx := sx
		drawn := 0
		for _, ch := range cell {
			if cx >= bounds.X+bounds.Width {
				break
			}
			w := runewidth.RuneWidth(ch)
			if drawn+w > maxW {
				// Truncate with ellipsis
				if cx < bounds.X+bounds.Width {
					screen.SetContent(cx, y, '…', nil, style)
				}
				break
			}
			screen.SetContent(cx, y, ch, nil, style)
			cx += w
			drawn += w
		}
	}
}

func (cv *CSVView) HandleEvent(ev tcell.Event) bool {
	switch tev := ev.(type) {
	case *tcell.EventKey:
		return cv.handleKey(tev)
	case *tcell.EventMouse:
		return cv.handleMouse(tev)
	}
	return false
}

func (cv *CSVView) handleKey(ev *tcell.EventKey) bool {
	bounds := cv.Bounds()
	contentH := bounds.Height - 4 // title + header + sep + status
	if contentH < 1 {
		contentH = 1
	}
	dataRows := len(cv.records) - 1
	if dataRows < 0 {
		dataRows = 0
	}

	switch ev.Key() {
	case tcell.KeyUp:
		if cv.cursorRow > 0 {
			cv.cursorRow--
			if cv.cursorRow < cv.scrollY {
				cv.scrollY = cv.cursorRow
			}
		}
		return true
	case tcell.KeyDown:
		if cv.cursorRow < dataRows-1 {
			cv.cursorRow++
			if cv.cursorRow >= cv.scrollY+contentH {
				cv.scrollY = cv.cursorRow - contentH + 1
			}
		}
		return true
	case tcell.KeyLeft:
		if cv.scrollX > 0 {
			cv.scrollX--
		}
		return true
	case tcell.KeyRight:
		cv.scrollX++
		return true
	case tcell.KeyPgUp:
		cv.cursorRow -= contentH
		if cv.cursorRow < 0 {
			cv.cursorRow = 0
		}
		cv.scrollY = cv.cursorRow
		return true
	case tcell.KeyPgDn:
		cv.cursorRow += contentH
		if cv.cursorRow >= dataRows {
			cv.cursorRow = dataRows - 1
		}
		if cv.cursorRow >= cv.scrollY+contentH {
			cv.scrollY = cv.cursorRow - contentH + 1
		}
		return true
	case tcell.KeyHome:
		cv.cursorRow = 0
		cv.scrollY = 0
		cv.scrollX = 0
		return true
	case tcell.KeyEnd:
		cv.cursorRow = dataRows - 1
		if cv.cursorRow < 0 {
			cv.cursorRow = 0
		}
		cv.scrollY = cv.cursorRow - contentH + 1
		if cv.scrollY < 0 {
			cv.scrollY = 0
		}
		return true
	}
	return false
}

func (cv *CSVView) handleMouse(ev *tcell.EventMouse) bool {
	bounds := cv.Bounds()
	_, my := ev.Position()
	if my < bounds.Y || my >= bounds.Y+bounds.Height {
		return false
	}

	dataRows := len(cv.records) - 1
	contentH := bounds.Height - 4

	if ev.Buttons()&tcell.WheelUp != 0 {
		cv.scrollY -= 3
		if cv.scrollY < 0 {
			cv.scrollY = 0
		}
		cv.cursorRow = cv.scrollY
		return true
	}
	if ev.Buttons()&tcell.WheelDown != 0 {
		cv.scrollY += 3
		maxScroll := dataRows - contentH
		if maxScroll < 0 {
			maxScroll = 0
		}
		if cv.scrollY > maxScroll {
			cv.scrollY = maxScroll
		}
		cv.cursorRow = cv.scrollY
		return true
	}
	if ev.Buttons()&tcell.Button1 != 0 {
		dataStartY := bounds.Y + 3
		row := my - dataStartY
		if row >= 0 {
			cv.cursorRow = cv.scrollY + row
			if cv.cursorRow >= dataRows {
				cv.cursorRow = dataRows - 1
			}
		}
		return true
	}
	return false
}

// helper: draw a string
func drawStr(screen tcell.Screen, x, y int, s string, style tcell.Style) {
	for _, ch := range s {
		screen.SetContent(x, y, ch, nil, style)
		x += runewidth.RuneWidth(ch)
	}
}

func itoa(n int) string {
	if n < 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	if n == 0 {
		return "0"
	}
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}
