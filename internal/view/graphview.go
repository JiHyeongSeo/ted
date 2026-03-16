package view

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/seoji/ted/internal/git"
	"github.com/seoji/ted/internal/syntax"
)

// BranchColors defines the color palette for graph branches.
var BranchColors = []tcell.Color{
	tcell.ColorRed,
	tcell.ColorGreen,
	tcell.ColorYellow,
	tcell.ColorBlue,
	tcell.ColorDarkCyan,
	tcell.ColorFuchsia,
	tcell.ColorOrange,
	tcell.ColorLightGray,
}

// GraphView renders a git commit graph with selection and scrolling.
type GraphView struct {
	BaseComponent
	theme       *syntax.Theme
	rows        []git.GraphRow
	selectedIdx int
	scrollY     int
	onSelect    func(commit *git.Commit)
	onEnter     func(commit *git.Commit)
}

func NewGraphView(theme *syntax.Theme, rows []git.GraphRow) *GraphView {
	return &GraphView{
		theme: theme,
		rows:  rows,
	}
}

func (gv *GraphView) SetOnSelect(fn func(commit *git.Commit)) {
	gv.onSelect = fn
}

func (gv *GraphView) SetOnEnter(fn func(commit *git.Commit)) {
	gv.onEnter = fn
}

func (gv *GraphView) SelectedIndex() int { return gv.selectedIdx }
func (gv *GraphView) ScrollY() int       { return gv.scrollY }

func (gv *GraphView) SelectedCommit() *git.Commit {
	if len(gv.rows) == 0 || gv.selectedIdx >= len(gv.rows) {
		return nil
	}
	return gv.rows[gv.selectedIdx].Commit
}

func (gv *GraphView) MoveDown() {
	if gv.selectedIdx < len(gv.rows)-1 {
		gv.selectedIdx++
		gv.ensureVisible()
		gv.notifySelect()
	}
}

func (gv *GraphView) MoveUp() {
	if gv.selectedIdx > 0 {
		gv.selectedIdx--
		gv.ensureVisible()
		gv.notifySelect()
	}
}

func (gv *GraphView) ensureVisible() {
	height := gv.Bounds().Height
	if height <= 0 {
		return
	}
	if gv.selectedIdx < gv.scrollY {
		gv.scrollY = gv.selectedIdx
	}
	if gv.selectedIdx >= gv.scrollY+height {
		gv.scrollY = gv.selectedIdx - height + 1
	}
}

func (gv *GraphView) notifySelect() {
	if gv.onSelect != nil {
		gv.onSelect(gv.SelectedCommit())
	}
}

func cellRune(cell git.GraphCell) rune {
	switch cell {
	case git.CellCommit:
		return '●'
	case git.CellPipe:
		return '│'
	case git.CellHorizontal:
		return '─'
	case git.CellMergeRight:
		return '┘'
	case git.CellBranchRight:
		return '┐'
	default:
		return ' '
	}
}

func (gv *GraphView) Render(screen tcell.Screen) {
	bounds := gv.Bounds()
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return
	}

	defaultStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorWhite)
	if gv.theme != nil {
		defaultStyle = gv.theme.UIStyle("default")
	}
	selectedStyle := defaultStyle.Background(tcell.ColorNavy)
	hashStyle := defaultStyle.Foreground(tcell.ColorGoldenrod)
	dimStyle := defaultStyle.Foreground(tcell.ColorGray)
	refStyle := defaultStyle.Foreground(tcell.ColorGreen).Bold(true)

	// Clear area
	for y := bounds.Y; y < bounds.Y+bounds.Height; y++ {
		for x := bounds.X; x < bounds.X+bounds.Width; x++ {
			screen.SetContent(x, y, ' ', nil, defaultStyle)
		}
	}

	if len(gv.rows) == 0 {
		msg := "No commits yet"
		x := bounds.X + (bounds.Width-len(msg))/2
		y := bounds.Y + bounds.Height/2
		for _, ch := range msg {
			if x < bounds.X+bounds.Width {
				screen.SetContent(x, y, ch, nil, dimStyle)
				x++
			}
		}
		return
	}

	// Determine graph column width
	maxCells := 0
	for i := 0; i < bounds.Height && i+gv.scrollY < len(gv.rows); i++ {
		row := gv.rows[i+gv.scrollY]
		if len(row.Cells) > maxCells {
			maxCells = len(row.Cells)
		}
	}
	graphWidth := maxCells * 2
	if graphWidth > bounds.Width/3 {
		graphWidth = bounds.Width / 3
	}
	if graphWidth < 2 {
		graphWidth = 2
	}

	hashWidth := 8
	authorWidth := 12
	ageWidth := 6
	msgStart := bounds.X + graphWidth + hashWidth
	msgEnd := bounds.X + bounds.Width - authorWidth - ageWidth

	for i := 0; i < bounds.Height; i++ {
		idx := gv.scrollY + i
		if idx >= len(gv.rows) {
			break
		}
		row := gv.rows[idx]
		y := bounds.Y + i

		baseStyle := defaultStyle
		if idx == gv.selectedIdx {
			baseStyle = selectedStyle
			for x := bounds.X; x < bounds.X+bounds.Width; x++ {
				screen.SetContent(x, y, ' ', nil, baseStyle)
			}
		}

		x := bounds.X

		// Draw graph cells
		for j, cell := range row.Cells {
			if x >= bounds.X+graphWidth {
				break
			}
			ch := cellRune(cell)
			style := baseStyle
			if cell != git.CellEmpty && j < len(row.Colors) {
				colorIdx := row.Colors[j] % len(BranchColors)
				style = baseStyle.Foreground(BranchColors[colorIdx])
			}
			screen.SetContent(x, y, ch, nil, style)
			x++
			if x < bounds.X+graphWidth {
				screen.SetContent(x, y, ' ', nil, baseStyle)
				x++
			}
		}
		for x < bounds.X+graphWidth {
			screen.SetContent(x, y, ' ', nil, baseStyle)
			x++
		}

		// Draw hash
		hs := hashStyle
		if idx == gv.selectedIdx {
			hs = selectedStyle.Foreground(tcell.ColorGoldenrod)
		}
		hash := row.Commit.ShortHash
		for _, ch := range hash {
			if x >= msgStart {
				break
			}
			screen.SetContent(x, y, ch, nil, hs)
			x++
		}
		screen.SetContent(x, y, ' ', nil, baseStyle)
		x++

		// Draw refs
		if len(row.Commit.Refs) > 0 {
			rs := refStyle
			if idx == gv.selectedIdx {
				rs = selectedStyle.Foreground(tcell.ColorGreen).Bold(true)
			}
			refText := "(" + joinRefs(row.Commit.Refs) + ") "
			for _, ch := range refText {
				if x >= msgEnd {
					break
				}
				screen.SetContent(x, y, ch, nil, rs)
				x++
			}
		}

		// Draw message
		ms := baseStyle
		for _, ch := range row.Commit.Message {
			if x >= msgEnd {
				break
			}
			w := runewidth.RuneWidth(ch)
			if x+w > msgEnd {
				break
			}
			screen.SetContent(x, y, ch, nil, ms)
			x += w
		}

		// Draw author
		ax := bounds.X + bounds.Width - authorWidth - ageWidth
		ds := dimStyle
		if idx == gv.selectedIdx {
			ds = selectedStyle.Foreground(tcell.ColorGray)
		}
		author := row.Commit.Author
		if len(author) > authorWidth-1 {
			author = author[:authorWidth-1]
		}
		for _, ch := range author {
			if ax >= bounds.X+bounds.Width-ageWidth {
				break
			}
			screen.SetContent(ax, y, ch, nil, ds)
			ax++
		}

		// Draw age
		age := formatAge(row.Commit.Date)
		ax = bounds.X + bounds.Width - ageWidth
		for _, ch := range age {
			if ax >= bounds.X+bounds.Width {
				break
			}
			screen.SetContent(ax, y, ch, nil, ds)
			ax++
		}
	}
}

func joinRefs(refs []string) string {
	if len(refs) == 0 {
		return ""
	}
	result := refs[0]
	for i := 1; i < len(refs); i++ {
		result += ", " + refs[i]
	}
	return result
}

func formatAge(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo", int(d.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy", int(d.Hours()/(24*365)))
	}
}

func (gv *GraphView) HandleEvent(ev tcell.Event) bool {
	switch tev := ev.(type) {
	case *tcell.EventKey:
		return gv.handleKey(tev)
	case *tcell.EventMouse:
		return gv.handleMouse(tev)
	}
	return false
}

func (gv *GraphView) handleKey(ev *tcell.EventKey) bool {
	height := gv.Bounds().Height
	switch ev.Key() {
	case tcell.KeyUp:
		gv.MoveUp()
		return true
	case tcell.KeyDown:
		gv.MoveDown()
		return true
	case tcell.KeyPgUp:
		for i := 0; i < height; i++ {
			gv.MoveUp()
		}
		return true
	case tcell.KeyPgDn:
		for i := 0; i < height; i++ {
			gv.MoveDown()
		}
		return true
	case tcell.KeyHome:
		gv.selectedIdx = 0
		gv.scrollY = 0
		gv.notifySelect()
		return true
	case tcell.KeyEnd:
		if len(gv.rows) > 0 {
			gv.selectedIdx = len(gv.rows) - 1
			gv.ensureVisible()
			gv.notifySelect()
		}
		return true
	case tcell.KeyEnter:
		if gv.onEnter != nil {
			c := gv.SelectedCommit()
			if c != nil {
				gv.onEnter(c)
			}
		}
		return true
	}
	return false
}

func (gv *GraphView) handleMouse(ev *tcell.EventMouse) bool {
	bounds := gv.Bounds()
	mx, my := ev.Position()
	if mx < bounds.X || mx >= bounds.X+bounds.Width || my < bounds.Y || my >= bounds.Y+bounds.Height {
		return false
	}

	maxScroll := len(gv.rows) - bounds.Height
	if maxScroll < 0 {
		maxScroll = 0
	}

	if ev.Buttons()&tcell.WheelUp != 0 {
		gv.scrollY -= 3
		if gv.scrollY < 0 {
			gv.scrollY = 0
		}
		return true
	}
	if ev.Buttons()&tcell.WheelDown != 0 {
		gv.scrollY += 3
		if gv.scrollY > maxScroll {
			gv.scrollY = maxScroll
		}
		return true
	}

	if ev.Buttons()&tcell.Button1 != 0 {
		row := my - bounds.Y + gv.scrollY
		if row >= 0 && row < len(gv.rows) {
			gv.selectedIdx = row
			gv.notifySelect()
		}
		return true
	}
	return false
}
