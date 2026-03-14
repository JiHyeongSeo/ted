package editor

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/seoji/ted/internal/git"
)

// gitShowStatus shows git status in the bottom panel.
func (e *Editor) gitShowStatus() {
	if e.diffTracker == nil {
		e.statusBar.SetMessage("Not a git repository")
		return
	}

	entries, err := e.diffTracker.Status()
	if err != nil {
		e.statusBar.SetMessage(err.Error())
		return
	}

	if len(entries) == 0 {
		e.statusBar.SetMessage("Git: working tree clean")
		return
	}

	lines := []string{fmt.Sprintf("Git Status (%d changed):", len(entries))}
	for _, entry := range entries {
		icon := " "
		switch entry.Status {
		case "M":
			icon = "M"
		case "A":
			icon = "A"
		case "D":
			icon = "D"
		case "??":
			icon = "?"
		case "R":
			icon = "R"
		case "MM":
			icon = "M"
		}
		lines = append(lines, fmt.Sprintf("  %s  %s", icon, entry.Path))
	}

	e.panel.SetContent(1, lines) // Output tab
	e.panel.SetActiveTab(1)
	e.layout.SetPanelVisible(true)
	e.panelFocus = true
	e.statusBar.SetMessage(fmt.Sprintf("Git: %d changed files", len(entries)))
}

// gitStageCurrentFile stages the current file.
func (e *Editor) gitStageCurrentFile() {
	if e.diffTracker == nil {
		e.statusBar.SetMessage("Not a git repository")
		return
	}
	tab := e.tabs.Active()
	if tab == nil || tab.Buffer.Path() == "" {
		e.statusBar.SetMessage("No file to stage")
		return
	}

	relPath, _ := filepath.Rel(e.diffTracker.RepoRoot(), tab.Buffer.Path())
	if err := e.diffTracker.StageFile(tab.Buffer.Path()); err != nil {
		e.statusBar.SetMessage(err.Error())
		return
	}
	e.statusBar.SetMessage(fmt.Sprintf("Staged: %s", relPath))
	e.updateGutterMarkers()
}

// gitStageAll stages all changes.
func (e *Editor) gitStageAll() {
	if e.diffTracker == nil {
		e.statusBar.SetMessage("Not a git repository")
		return
	}
	if err := e.diffTracker.StageAll(); err != nil {
		e.statusBar.SetMessage(err.Error())
		return
	}
	e.statusBar.SetMessage("Staged all changes")
	e.updateGutterMarkers()
}

// gitCommitPrompt opens the input bar for a commit message.
func (e *Editor) gitCommitPrompt() {
	if e.diffTracker == nil {
		e.statusBar.SetMessage("Not a git repository")
		return
	}

	e.inputBar.Show("Commit message: ")
	e.inputBar.SetOnSubmit(func(msg string) {
		e.inputBar.Hide()
		if strings.TrimSpace(msg) == "" {
			e.statusBar.SetMessage("Commit aborted: empty message")
			return
		}
		out, err := e.diffTracker.Commit(msg)
		if err != nil {
			e.statusBar.SetMessage(err.Error())
			return
		}
		// Show first line of git commit output
		lines := strings.Split(out, "\n")
		e.statusBar.SetMessage("Committed: " + lines[0])
		e.updateGutterMarkers()
	})
}

// gitPush pushes to the remote.
func (e *Editor) gitPush() {
	if e.diffTracker == nil {
		e.statusBar.SetMessage("Not a git repository")
		return
	}
	e.statusBar.SetMessage("Pushing...")
	go func() {
		out, err := e.diffTracker.Push()
		if err != nil {
			e.statusBar.SetMessage(err.Error())
		} else if out != "" {
			e.statusBar.SetMessage("Pushed: " + strings.Split(out, "\n")[0])
		} else {
			e.statusBar.SetMessage("Pushed successfully")
		}
		if e.screen != nil {
			e.screen.PostEvent(tcell.NewEventInterrupt(nil))
		}
	}()
}

// gitPull pulls from the remote.
func (e *Editor) gitPull() {
	if e.diffTracker == nil {
		e.statusBar.SetMessage("Not a git repository")
		return
	}
	e.statusBar.SetMessage("Pulling...")
	go func() {
		out, err := e.diffTracker.Pull()
		if err != nil {
			e.statusBar.SetMessage(err.Error())
		} else if out != "" {
			e.statusBar.SetMessage("Pulled: " + strings.Split(out, "\n")[0])
		} else {
			e.statusBar.SetMessage("Pulled successfully")
		}
		if e.screen != nil {
			e.screen.PostEvent(tcell.NewEventInterrupt(nil))
		}
	}()
}

// gitToggleBlame toggles git blame display.
func (e *Editor) gitToggleBlame() {
	if e.editorView == nil {
		return
	}
	if e.editorView.HasBlame() {
		e.editorView.ClearBlame()
		e.statusBar.SetMessage("Blame: off")
		return
	}

	if e.diffTracker == nil {
		e.statusBar.SetMessage("Not a git repository")
		return
	}
	tab := e.tabs.Active()
	if tab == nil || tab.Buffer.Path() == "" {
		e.statusBar.SetMessage("No file for blame")
		return
	}

	blameData, err := e.diffTracker.Blame(tab.Buffer.Path())
	if err != nil {
		e.statusBar.SetMessage(err.Error())
		return
	}

	const blameWidth = 28
	lines := make([]string, len(blameData))
	for i, b := range blameData {
		lines[i] = git.FormatBlameLine(b, blameWidth-1) // -1 for separator
	}

	e.editorView.SetBlame(lines, blameWidth)
	e.statusBar.SetMessage("Blame: on")
}
