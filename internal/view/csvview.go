package view

import (
	"bytes"
	"encoding/csv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/JiHyeongSeo/ted/internal/syntax"
	"github.com/JiHyeongSeo/ted/internal/types"
)

// CSVView renders CSV content as an editable table.
type CSVView struct {
	BaseComponent
	theme     *syntax.Theme
	records   [][]string // all rows (including header at index 0)
	colWidths []int      // max display width per column
	scrollY   int        // first visible data row index (0-based, excluding header)
	scrollX   int        // first visible column index
	cursorRow int        // selected data row (0-based, excluding header)
	cursorCol int        // selected column
	title     string
	// onEdit is called when the user requests to edit the current cell.
	// setValue(newVal) must be called by the consumer to commit the new value.
	onEdit func(row, col int, current string, setValue func(string))
}

// NewCSVView parses CSV content and returns a CSVView.
func NewCSVView(theme *syntax.Theme, content, title string) *CSVView {
	cv := &CSVView{theme: theme, title: title}
	r := csv.NewReader(strings.NewReader(content))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
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

// SetOnEdit sets the callback invoked when the user wants to edit a cell.
func (cv *CSVView) SetOnEdit(fn func(row, col int, current string, setValue func(string))) {
	cv.onEdit = fn
}

// Serialize encodes the current records back to CSV text.
func (cv *CSVView) Serialize() string {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.WriteAll(cv.records)
	w.Flush()
	return buf.String()
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
	for i := range cv.colWidths {
		if cv.colWidths[i] < 3 {
			cv.colWidths[i] = 3
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
	return len(cv.records) - 1
}

func (cv *CSVView) ColCount() int {
	return len(cv.colWidths)
}

// buildColOffsets returns screen-relative x offsets for each column.
// Columns before scrollX are mapped to -99999 (hidden).
func (cv *CSVView) buildColOffsets() []int {
	offsets := make([]int, len(cv.colWidths))
	x := 0
	for i, w := range cv.colWidths {
		if i < cv.scrollX {
			offsets[i] = -99999
		} else {
			offsets[i] = x
			x += w + 3 // col width + " │ " separator
		}
	}
	return offsets
}

// ensureCursorColVisible adjusts scrollX so the cursor column is visible.
func (cv *CSVView) ensureCursorColVisible() {
	if cv.cursorCol < cv.scrollX {
		cv.scrollX = cv.cursorCol
		return
	}
	bounds := cv.Bounds()
	contentW := bounds.Width - 2
	x := 0
	for i := cv.scrollX; i < len(cv.colWidths); i++ {
		if i == cv.cursorCol {
			if x+cv.colWidths[i] <= contentW {
				return
			}
			cv.scrollX++
			cv.ensureCursorColVisible()
			return
		}
		x += cv.colWidths[i] + 3
	}
}

func (cv *CSVView) Render(screen tcell.Screen) {
	bounds := cv.Bounds()
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return
	}

	baseStyle := cv.theme.UIStyle("default")
	headerStyle := baseStyle.Bold(true).Foreground(cv.theme.ResolveColor("#c8c8c8"))
	sepStyle := baseStyle.Foreground(cv.theme.ResolveColor("#505050"))
	cursorRowStyle := baseStyle.Background(cv.theme.ResolveColor("#264f78"))
	cursorCellStyle := baseStyle.Background(cv.theme.ResolveColor("#1e6ab4")).Bold(true)
	altStyle := baseStyle.Background(cv.theme.ResolveColor("#1e1e2e"))
	titleStyle := baseStyle.Foreground(cv.theme.ResolveColor("#c8c8c8")).Bold(true)
	statusStyle := baseStyle.Foreground(cv.theme.ResolveColor("#808080"))

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
		titleStr += " [col+" + itoa(cv.scrollX) + "]"
	}
	drawStr(screen, bounds.X, bounds.Y, titleStr, titleStyle)
	for x := bounds.X + runewidth.StringWidth(titleStr); x < bounds.X+bounds.Width; x++ {
		screen.SetContent(x, bounds.Y, '─', nil, sepStyle)
	}

	colOffsets := cv.buildColOffsets()
	contentH := bounds.Height - 1

	// Header row
	headerY := bounds.Y + 1
	cv.renderRow(screen, bounds, headerY, cv.records[0], colOffsets, headerStyle, -1)
	for x := bounds.X; x < bounds.X+bounds.Width; x++ {
		screen.SetContent(x, headerY+1, '─', nil, sepStyle)
	}

	// Data rows
	dataStartY := bounds.Y + 3
	visibleRows := contentH - 3
	if visibleRows < 1 {
		return
	}

	for i := 0; i < visibleRows; i++ {
		recIdx := cv.scrollY + i + 1
		if recIdx >= len(cv.records) {
			break
		}
		y := dataStartY + i
		isCursor := (cv.scrollY + i) == cv.cursorRow
		rowStyle := baseStyle
		if isCursor {
			rowStyle = cursorRowStyle
		} else if i%2 == 1 {
			rowStyle = altStyle
		}
		highlightCol := -1
		if isCursor {
			highlightCol = cv.cursorCol
		}
		cv.renderRow(screen, bounds, y, cv.records[recIdx], colOffsets, rowStyle, highlightCol)
		// Draw cursor cell style on top for the active cell
		if isCursor {
			cv.renderCell(screen, bounds, y, cv.records[recIdx], colOffsets, cursorCellStyle, cv.cursorCol)
		}
	}

	// Status bar: row/col position + hint
	col := cv.cursorCol
	colName := ""
	if len(cv.records) > 0 && col < len(cv.records[0]) {
		colName = " [" + cv.records[0][col] + "]"
	}
	statusStr := " R" + itoa(cv.cursorRow+1) + " C" + itoa(cv.cursorCol+1) + colName +
		"  " + itoa(cv.cursorRow+1) + "/" + itoa(len(cv.records)-1) + " rows" +
		"  Enter:edit  Tab:next col"
	drawStr(screen, bounds.X, bounds.Y+bounds.Height-1, statusStr, statusStyle)
}

func (cv *CSVView) renderRow(screen tcell.Screen, bounds types.Rect, y int, row []string, colOffsets []int, style tcell.Style, _ int) {
	for x := bounds.X; x < bounds.X+bounds.Width; x++ {
		screen.SetContent(x, y, ' ', nil, style)
	}
	sepStyle := style.Foreground(cv.theme.ResolveColor("#505050"))
	for j, offset := range colOffsets {
		if offset == -99999 {
			continue
		}
		sx := bounds.X + 1 + offset
		if sx >= bounds.X+bounds.Width {
			break
		}
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
		cx := sx
		drawn := 0
		for _, ch := range cell {
			if cx >= bounds.X+bounds.Width {
				break
			}
			w := runewidth.RuneWidth(ch)
			if drawn+w > maxW {
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

// renderCell re-draws a single cell with a different style (used for cursor cell highlight).
func (cv *CSVView) renderCell(screen tcell.Screen, bounds types.Rect, y int, row []string, colOffsets []int, style tcell.Style, col int) {
	if col < 0 || col >= len(colOffsets) {
		return
	}
	offset := colOffsets[col]
	if offset == -99999 {
		return
	}
	sx := bounds.X + 1 + offset
	if sx < bounds.X || sx >= bounds.X+bounds.Width {
		return
	}
	maxW := cv.colWidths[col]
	// Clear cell area
	for x := sx; x < sx+maxW && x < bounds.X+bounds.Width; x++ {
		screen.SetContent(x, y, ' ', nil, style)
	}
	cell := ""
	if col < len(row) {
		cell = row[col]
	}
	cx := sx
	drawn := 0
	for _, ch := range cell {
		if cx >= bounds.X+bounds.Width {
			break
		}
		w := runewidth.RuneWidth(ch)
		if drawn+w > maxW {
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
	contentH := bounds.Height - 4
	if contentH < 1 {
		contentH = 1
	}
	dataRows := len(cv.records) - 1
	if dataRows < 0 {
		dataRows = 0
	}
	numCols := len(cv.colWidths)

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
		if cv.cursorCol > 0 {
			cv.cursorCol--
			cv.ensureCursorColVisible()
		}
		return true
	case tcell.KeyRight:
		if cv.cursorCol < numCols-1 {
			cv.cursorCol++
			cv.ensureCursorColVisible()
		}
		return true
	case tcell.KeyTab:
		// Tab: advance to next cell, wrap to next row
		cv.cursorCol++
		if cv.cursorCol >= numCols {
			cv.cursorCol = 0
			if cv.cursorRow < dataRows-1 {
				cv.cursorRow++
				if cv.cursorRow >= cv.scrollY+contentH {
					cv.scrollY = cv.cursorRow - contentH + 1
				}
			}
		}
		cv.ensureCursorColVisible()
		return true
	case tcell.KeyBacktab: // Shift+Tab
		cv.cursorCol--
		if cv.cursorCol < 0 {
			cv.cursorCol = numCols - 1
			if cv.cursorRow > 0 {
				cv.cursorRow--
				if cv.cursorRow < cv.scrollY {
					cv.scrollY = cv.cursorRow
				}
			}
		}
		cv.ensureCursorColVisible()
		return true
	case tcell.KeyEnter, tcell.KeyF2:
		cv.triggerEdit()
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
		cv.cursorCol = 0
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
	mx, my := ev.Position()
	if mx < bounds.X || mx >= bounds.X+bounds.Width || my < bounds.Y || my >= bounds.Y+bounds.Height {
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
			// Determine column from x position
			colOffsets := cv.buildColOffsets()
			relX := mx - bounds.X - 1
			for j := len(colOffsets) - 1; j >= 0; j-- {
				if colOffsets[j] != -99999 && relX >= colOffsets[j] {
					cv.cursorCol = j
					break
				}
			}
		}
		return true
	}
	return false
}

// triggerEdit calls the onEdit callback for the current cell.
func (cv *CSVView) triggerEdit() {
	if cv.onEdit == nil {
		return
	}
	recIdx := cv.cursorRow + 1
	if recIdx >= len(cv.records) {
		return
	}
	col := cv.cursorCol
	current := ""
	if col < len(cv.records[recIdx]) {
		current = cv.records[recIdx][col]
	}
	setValue := func(newVal string) {
		for len(cv.records[recIdx]) <= col {
			cv.records[recIdx] = append(cv.records[recIdx], "")
		}
		cv.records[recIdx][col] = newVal
		cv.computeColWidths()
	}
	cv.onEdit(cv.cursorRow, col, current, setValue)
}

// helper: draw a string
func drawStr(screen tcell.Screen, x, y int, s string, style tcell.Style) {
	for _, ch := range s {
		screen.SetContent(x, y, ch, nil, style)
		x += runewidth.RuneWidth(ch)
	}
}

func itoa(n int) string {
	if n <= 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}
