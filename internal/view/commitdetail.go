package view

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/seoji/ted/internal/git"
	"github.com/seoji/ted/internal/syntax"
)

// CommitDetailView displays details of a selected commit with selectable file list.
type CommitDetailView struct {
	BaseComponent
	theme       *syntax.Theme
	commit      *git.Commit
	files       []string // "M\tpath/to/file" format
	scrollY     int
	selectedIdx int // selected file index (-1 = none)
	onFileEnter func(commit *git.Commit, fileLine string)
}

func NewCommitDetailView(theme *syntax.Theme) *CommitDetailView {
	return &CommitDetailView{theme: theme, selectedIdx: -1}
}

func (cv *CommitDetailView) SetCommit(commit *git.Commit, files []string) {
	cv.commit = commit
	cv.files = files
	cv.scrollY = 0
	if len(files) > 0 {
		cv.selectedIdx = 0
	} else {
		cv.selectedIdx = -1
	}
}

// SetOnFileEnter sets callback when Enter is pressed on a file.
func (cv *CommitDetailView) SetOnFileEnter(fn func(commit *git.Commit, fileLine string)) {
	cv.onFileEnter = fn
}

// SelectedFile returns the currently selected file line (e.g. "M\tpath"), or "".
func (cv *CommitDetailView) SelectedFile() string {
	if cv.selectedIdx >= 0 && cv.selectedIdx < len(cv.files) {
		return cv.files[cv.selectedIdx]
	}
	return ""
}

// headerLines returns how many lines the header takes (before the file list).
func (cv *CommitDetailView) headerLines() int {
	// separator(1) + "Commit Details"(1) + hash(1) + message(1) + author(1) + blank(1) + "Changed files"(1) = 7
	return 7
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
	selectedBg := defaultStyle.Background(tcell.ColorNavy)

	// Clear area
	for y := bounds.Y; y < bounds.Y+bounds.Height; y++ {
		for x := bounds.X; x < bounds.X+bounds.Width; x++ {
			screen.SetContent(x, y, ' ', nil, defaultStyle)
		}
	}

	// Draw separator line at top — highlight when focused
	sepColor := tcell.ColorDarkCyan
	if cv.IsFocused() {
		sepColor = tcell.ColorSteelBlue
	}
	sepStyle := defaultStyle.Foreground(sepColor)
	for x := bounds.X; x < bounds.X+bounds.Width; x++ {
		screen.SetContent(x, bounds.Y, '─', nil, sepStyle)
	}

	// Draw focus indicator label on separator
	if cv.IsFocused() {
		label := " Files (↑↓ Enter:diff  a:stage  Esc:back) "
		lx := bounds.X + 1
		labelStyle := defaultStyle.Foreground(tcell.ColorWhite).Background(tcell.ColorSteelBlue).Bold(true)
		for _, ch := range label {
			if lx >= bounds.X+bounds.Width-1 {
				break
			}
			screen.SetContent(lx, bounds.Y, ch, nil, labelStyle)
			lx++
		}
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
		filesHeader := fmt.Sprintf("Changed files (%d)  [↑↓ select, Enter to diff]:", len(cv.files))
		cv.drawLine(screen, bounds.X+1, y, bounds.Width-1, filesHeader, dimStyle)
		y++
	}

	for i := cv.scrollY; i < len(cv.files) && y < bounds.Y+bounds.Height; i++ {
		line := cv.files[i]

		// Parse status and path from "M\tpath" or "??\tpath" format
		status := ""
		displayPath := line
		if idx := strings.IndexByte(line, '\t'); idx >= 0 {
			status = line[:idx]
			displayPath = line[idx+1:]
		}

		style := defaultStyle
		switch status {
		case "A":
			style = addedStyle
		case "D":
			style = removedStyle
		case "M", "MM":
			style = modifiedStyle
		case "??":
			style = addedStyle // untracked = new
		}

		// Highlight selected file
		if i == cv.selectedIdx {
			for x := bounds.X; x < bounds.X+bounds.Width; x++ {
				screen.SetContent(x, y, ' ', nil, selectedBg)
			}
			fg, _, _ := style.Decompose()
			style = selectedBg.Foreground(fg)
			screen.SetContent(bounds.X+1, y, '>', nil, style)
		}

		// Draw status at fixed column, then path at fixed column
		statusX := bounds.X + 3
		pathX := bounds.X + 7
		// Draw status (e.g. "M", "??", "A")
		sx := statusX
		for _, ch := range status {
			if sx >= pathX {
				break
			}
			screen.SetContent(sx, y, ch, nil, style)
			sx++
		}
		// Draw path
		cv.drawLine(screen, pathX, y, bounds.X+bounds.Width-pathX, displayPath, style)
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

// Decompose helper — tcell.Style doesn't have a Decompose that returns Fg directly,
// so we use a simpler approach in Render.

func (cv *CommitDetailView) HandleEvent(ev tcell.Event) bool {
	switch tev := ev.(type) {
	case *tcell.EventKey:
		return cv.handleKey(tev)
	case *tcell.EventMouse:
		return cv.handleMouse(tev)
	}
	return false
}

func (cv *CommitDetailView) handleKey(ev *tcell.EventKey) bool {
	if len(cv.files) == 0 {
		return false
	}
	switch ev.Key() {
	case tcell.KeyUp:
		if cv.selectedIdx > 0 {
			cv.selectedIdx--
			cv.ensureFileVisible()
		}
		return true
	case tcell.KeyDown:
		if cv.selectedIdx < len(cv.files)-1 {
			cv.selectedIdx++
			cv.ensureFileVisible()
		}
		return true
	case tcell.KeyEnter:
		if cv.selectedIdx >= 0 && cv.selectedIdx < len(cv.files) && cv.onFileEnter != nil {
			cv.onFileEnter(cv.commit, cv.files[cv.selectedIdx])
		}
		return true
	}
	return false
}

func (cv *CommitDetailView) ensureFileVisible() {
	visibleFiles := cv.Bounds().Height - cv.headerLines()
	if visibleFiles <= 0 {
		return
	}
	if cv.selectedIdx < cv.scrollY {
		cv.scrollY = cv.selectedIdx
	}
	if cv.selectedIdx >= cv.scrollY+visibleFiles {
		cv.scrollY = cv.selectedIdx - visibleFiles + 1
	}
}

func (cv *CommitDetailView) handleMouse(ev *tcell.EventMouse) bool {
	bounds := cv.Bounds()
	mx, my := ev.Position()
	if mx < bounds.X || mx >= bounds.X+bounds.Width || my < bounds.Y || my >= bounds.Y+bounds.Height {
		return false
	}
	maxScroll := len(cv.files) - (bounds.Height - cv.headerLines())
	if maxScroll < 0 {
		maxScroll = 0
	}
	if ev.Buttons()&tcell.WheelUp != 0 {
		cv.scrollY -= 3
		if cv.scrollY < 0 {
			cv.scrollY = 0
		}
		return true
	}
	if ev.Buttons()&tcell.WheelDown != 0 {
		cv.scrollY += 3
		if cv.scrollY > maxScroll {
			cv.scrollY = maxScroll
		}
		return true
	}
	// Click to select file
	if ev.Buttons()&tcell.Button1 != 0 {
		fileRow := my - bounds.Y - cv.headerLines() + cv.scrollY
		if fileRow >= 0 && fileRow < len(cv.files) {
			cv.selectedIdx = fileRow
		}
		return true
	}
	return false
}
