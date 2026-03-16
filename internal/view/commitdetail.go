package view

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/seoji/ted/internal/git"
	"github.com/seoji/ted/internal/syntax"
)

// CommitDetailView displays details of a selected commit.
type CommitDetailView struct {
	BaseComponent
	theme   *syntax.Theme
	commit  *git.Commit
	files   []string // "M\tpath/to/file" format
	scrollY int
}

func NewCommitDetailView(theme *syntax.Theme) *CommitDetailView {
	return &CommitDetailView{theme: theme}
}

func (cv *CommitDetailView) SetCommit(commit *git.Commit, files []string) {
	cv.commit = commit
	cv.files = files
	cv.scrollY = 0
}

func (cv *CommitDetailView) Render(screen tcell.Screen) {
	bounds := cv.Bounds()
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return
	}

	defaultStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorWhite)
	if cv.theme != nil {
		defaultStyle = cv.theme.UIStyle("default")
	}
	headerStyle := defaultStyle.Foreground(tcell.ColorSteelBlue).Bold(true)
	hashStyle := defaultStyle.Foreground(tcell.ColorGoldenrod)
	dimStyle := defaultStyle.Foreground(tcell.ColorGray)
	addedStyle := defaultStyle.Foreground(tcell.ColorGreen)
	removedStyle := defaultStyle.Foreground(tcell.ColorRed)
	modifiedStyle := defaultStyle.Foreground(tcell.ColorYellow)

	// Clear area
	for y := bounds.Y; y < bounds.Y+bounds.Height; y++ {
		for x := bounds.X; x < bounds.X+bounds.Width; x++ {
			screen.SetContent(x, y, ' ', nil, defaultStyle)
		}
	}

	// Draw separator line at top
	sepStyle := defaultStyle.Foreground(tcell.ColorDarkCyan)
	for x := bounds.X; x < bounds.X+bounds.Width; x++ {
		screen.SetContent(x, bounds.Y, '─', nil, sepStyle)
	}

	if cv.commit == nil {
		msg := "Select a commit to view details"
		x := bounds.X + 1
		y := bounds.Y + 1
		for _, ch := range msg {
			if x >= bounds.X+bounds.Width {
				break
			}
			screen.SetContent(x, y, ch, nil, dimStyle)
			x++
		}
		return
	}

	y := bounds.Y + 1

	cv.drawLine(screen, bounds.X+1, y, bounds.Width-1, "Commit Details", headerStyle)
	y++

	cv.drawLine(screen, bounds.X+1, y, bounds.Width-1, cv.commit.Hash, hashStyle)
	y++

	cv.drawLine(screen, bounds.X+1, y, bounds.Width-1, cv.commit.Message, defaultStyle)
	y++

	info := fmt.Sprintf("Author: %s · Date: %s", cv.commit.Author, cv.commit.Date.Format("2006-01-02 15:04"))
	cv.drawLine(screen, bounds.X+1, y, bounds.Width-1, info, dimStyle)
	y++

	y++ // blank line

	if y < bounds.Y+bounds.Height {
		filesHeader := fmt.Sprintf("Changed files (%d):", len(cv.files))
		cv.drawLine(screen, bounds.X+1, y, bounds.Width-1, filesHeader, dimStyle)
		y++
	}

	for i := cv.scrollY; i < len(cv.files) && y < bounds.Y+bounds.Height; i++ {
		line := cv.files[i]
		style := defaultStyle
		if len(line) > 0 {
			switch line[0] {
			case 'A':
				style = addedStyle
			case 'D':
				style = removedStyle
			case 'M':
				style = modifiedStyle
			}
		}
		cv.drawLine(screen, bounds.X+3, y, bounds.Width-3, line, style)
		y++
	}
}

func (cv *CommitDetailView) drawLine(screen tcell.Screen, x, y, maxWidth int, text string, style tcell.Style) {
	endX := x + maxWidth
	for _, ch := range text {
		w := runewidth.RuneWidth(ch)
		if x+w > endX {
			break
		}
		screen.SetContent(x, y, ch, nil, style)
		x += w
	}
}

func (cv *CommitDetailView) HandleEvent(ev tcell.Event) bool {
	switch tev := ev.(type) {
	case *tcell.EventMouse:
		bounds := cv.Bounds()
		mx, my := tev.Position()
		if mx < bounds.X || mx >= bounds.X+bounds.Width || my < bounds.Y || my >= bounds.Y+bounds.Height {
			return false
		}
		maxScroll := len(cv.files) - (bounds.Height - 6)
		if maxScroll < 0 {
			maxScroll = 0
		}
		if tev.Buttons()&tcell.WheelUp != 0 {
			cv.scrollY -= 3
			if cv.scrollY < 0 {
				cv.scrollY = 0
			}
			return true
		}
		if tev.Buttons()&tcell.WheelDown != 0 {
			cv.scrollY += 3
			if cv.scrollY > maxScroll {
				cv.scrollY = maxScroll
			}
			return true
		}
	}
	return false
}
