package view

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/JiHyeongSeo/ted/internal/git"
	"github.com/JiHyeongSeo/ted/internal/syntax"
)

// RebaseAction represents an action in the interactive rebase plan.
type RebaseAction string

const (
	RebaseActionPick   RebaseAction = "pick"
	RebaseActionSquash RebaseAction = "squash"
	RebaseActionFixup  RebaseAction = "fixup"
	RebaseActionDrop   RebaseAction = "drop"
	RebaseActionReword RebaseAction = "reword"
)

var rebaseActionCycle = []RebaseAction{
	RebaseActionPick,
	RebaseActionSquash,
	RebaseActionFixup,
	RebaseActionDrop,
	RebaseActionReword,
}

// RebaseViewEntry is one row in the RebaseView.
type RebaseViewEntry struct {
	Action RebaseAction
	Commit *git.Commit
}

// RebaseView renders an interactive rebase editor.
// Commits are shown oldest-first (matching git rebase -i order).
type RebaseView struct {
	BaseComponent
	theme     *syntax.Theme
	entries   []RebaseViewEntry
	cursor    int
	scrollY   int
	onExecute func(entries []RebaseViewEntry)
	onCancel  func()
}

// NewRebaseView creates a RebaseView from a slice of commits.
// commits must be ordered newest-first (as they appear in the graph);
// they are reversed internally so the view shows oldest-first.
func NewRebaseView(theme *syntax.Theme, commits []*git.Commit) *RebaseView {
	rv := &RebaseView{theme: theme}
	// Reverse: oldest first
	for i := len(commits) - 1; i >= 0; i-- {
		rv.entries = append(rv.entries, RebaseViewEntry{
			Action: RebaseActionPick,
			Commit: commits[i],
		})
	}
	return rv
}

func (rv *RebaseView) SetOnExecute(fn func(entries []RebaseViewEntry)) { rv.onExecute = fn }
func (rv *RebaseView) SetOnCancel(fn func())                            { rv.onCancel = fn }
func (rv *RebaseView) Entries() []RebaseViewEntry                       { return rv.entries }

func (rv *RebaseView) Render(screen tcell.Screen) {
	bounds := rv.Bounds()
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return
	}

	base := rv.theme.UIStyle("default")
	headerStyle := base.Foreground(rv.theme.ResolveColor("#c8c8c8")).Bold(true)
	sepStyle := base.Foreground(rv.theme.ResolveColor("#505050"))
	cursorStyle := base.Background(rv.theme.ResolveColor("#264f78"))
	hintStyle := base.Foreground(rv.theme.ResolveColor("#808080"))
	hashStyle := base.Foreground(rv.theme.ResolveColor("#808080"))

	actionColors := map[RebaseAction]string{
		RebaseActionPick:   "#6a9955", // green
		RebaseActionSquash: "#dcdcaa", // yellow
		RebaseActionFixup:  "#ce9178", // orange
		RebaseActionDrop:   "#f44747", // red
		RebaseActionReword: "#569cd6", // blue
	}

	// Clear
	for y := bounds.Y; y < bounds.Y+bounds.Height; y++ {
		for x := bounds.X; x < bounds.X+bounds.Width; x++ {
			screen.SetContent(x, y, ' ', nil, base)
		}
	}

	// Header
	header := fmt.Sprintf(" Interactive Rebase  (%d commits)", len(rv.entries))
	drawStr(screen, bounds.X, bounds.Y, header, headerStyle)
	for x := bounds.X + runewidth.StringWidth(header); x < bounds.X+bounds.Width; x++ {
		screen.SetContent(x, bounds.Y, '─', nil, sepStyle)
	}

	// Content rows
	contentH := bounds.Height - 2 // header + hint
	for i := 0; i < contentH; i++ {
		idx := rv.scrollY + i
		if idx >= len(rv.entries) {
			break
		}
		y := bounds.Y + 1 + i
		entry := rv.entries[idx]
		isCursor := idx == rv.cursor

		rowStyle := base
		if isCursor {
			rowStyle = cursorStyle
		}

		// Clear row
		for x := bounds.X; x < bounds.X+bounds.Width; x++ {
			screen.SetContent(x, y, ' ', nil, rowStyle)
		}

		x := bounds.X + 1

		// Action [pick  ]
		actionStr := fmt.Sprintf("[%-6s] ", string(entry.Action))
		aStyle := rowStyle
		if !isCursor {
			if hex, ok := actionColors[entry.Action]; ok {
				aStyle = base.Foreground(rv.theme.ResolveColor(hex))
			}
		}
		for _, ch := range actionStr {
			if x >= bounds.X+bounds.Width {
				break
			}
			screen.SetContent(x, y, ch, nil, aStyle)
			x += runewidth.RuneWidth(ch)
		}

		// Short hash
		hashStr := entry.Commit.ShortHash + "  "
		hs := hashStyle
		if isCursor {
			hs = rowStyle
		}
		for _, ch := range hashStr {
			if x >= bounds.X+bounds.Width {
				break
			}
			screen.SetContent(x, y, ch, nil, hs)
			x += runewidth.RuneWidth(ch)
		}

		// Message
		for _, ch := range entry.Commit.Message {
			if x >= bounds.X+bounds.Width {
				break
			}
			screen.SetContent(x, y, ch, nil, rowStyle)
			x += runewidth.RuneWidth(ch)
		}
	}

	// Hint line
	hint := " Space:cycle action  Enter:execute  Esc:cancel"
	drawStr(screen, bounds.X, bounds.Y+bounds.Height-1, hint, hintStyle)
}

func (rv *RebaseView) HandleEvent(ev tcell.Event) bool {
	kev, ok := ev.(*tcell.EventKey)
	if !ok {
		return false
	}

	contentH := rv.Bounds().Height - 2
	if contentH < 1 {
		contentH = 1
	}

	switch kev.Key() {
	case tcell.KeyUp:
		if rv.cursor > 0 {
			rv.cursor--
			if rv.cursor < rv.scrollY {
				rv.scrollY = rv.cursor
			}
		}
		return true
	case tcell.KeyDown:
		if rv.cursor < len(rv.entries)-1 {
			rv.cursor++
			if rv.cursor >= rv.scrollY+contentH {
				rv.scrollY = rv.cursor - contentH + 1
			}
		}
		return true
	case tcell.KeyRune:
		if kev.Rune() == ' ' {
			rv.cycleAction(rv.cursor)
			return true
		}
	case tcell.KeyEnter:
		if rv.onExecute != nil {
			rv.onExecute(rv.entries)
		}
		return true
	case tcell.KeyEscape:
		if rv.onCancel != nil {
			rv.onCancel()
		}
		return true
	}
	return false
}

func (rv *RebaseView) cycleAction(idx int) {
	if idx < 0 || idx >= len(rv.entries) {
		return
	}
	cur := rv.entries[idx].Action
	for i, a := range rebaseActionCycle {
		if a == cur {
			rv.entries[idx].Action = rebaseActionCycle[(i+1)%len(rebaseActionCycle)]
			return
		}
	}
	rv.entries[idx].Action = RebaseActionPick
}
