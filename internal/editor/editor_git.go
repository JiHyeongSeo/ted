package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/seoji/ted/internal/buffer"
	"github.com/seoji/ted/internal/git"
	"github.com/seoji/ted/internal/view"
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

// gitOpenGraph opens or focuses the git graph tab.
func (e *Editor) gitOpenGraph() {
	if e.diffTracker == nil {
		e.statusBar.SetMessage("Not a git repository")
		return
	}

	// Check if graph tab already open
	for i, tab := range e.tabs.All() {
		if tab.Kind == TabKindGraph {
			e.tabs.SetActive(i)
			e.sidebarFocus = false
			e.panelFocus = false
			e.syncViewToTab()
			return
		}
	}

	// Load commits
	commits, err := git.LoadCommits(e.diffTracker.RepoRoot(), 500)
	if err != nil {
		e.statusBar.SetMessage(err.Error())
		return
	}

	// Check for uncommitted changes and prepend virtual commit
	statusEntries, _ := e.diffTracker.Status()
	if len(statusEntries) > 0 {
		uncommitted := git.Commit{
			Hash:      "uncommitted",
			ShortHash: "•••••••",
			Message:   "Uncommitted Changes",
			Author:    "",
		}
		if len(commits) > 0 {
			uncommitted.Parents = []string{commits[0].Hash}
		}
		commits = append([]git.Commit{uncommitted}, commits...)
	}

	// Create graph views
	rows := git.LayoutGraph(commits)
	e.graphView = view.NewGraphView(e.theme, rows)
	e.commitDetailView = view.NewCommitDetailView(e.theme)

	// Set up selection callback
	repoRoot := e.diffTracker.RepoRoot()
	e.graphView.SetOnSelect(func(commit *git.Commit) {
		if commit == nil {
			e.commitDetailView.SetCommit(nil, nil)
			return
		}
		if commit.Hash == "uncommitted" {
			// Show working tree changes
			var files []string
			for _, entry := range statusEntries {
				files = append(files, entry.Status+"\t"+entry.Path)
			}
			e.commitDetailView.SetCommit(commit, files)
			return
		}
		files, _ := git.LoadChangedFiles(repoRoot, commit.Hash)
		e.commitDetailView.SetCommit(commit, files)
	})

	// Set up file Enter callback — open inline diff view
	e.commitDetailView.SetOnFileEnter(func(commit *git.Commit, fileLine string) {
		if commit == nil || fileLine == "" {
			return
		}
		// Parse "M\tpath/to/file" format
		parts := strings.SplitN(fileLine, "\t", 2)
		if len(parts) < 2 {
			return
		}
		status := parts[0]
		filePath := parts[1]

		var oldText, newText string

		if commit.Hash == "uncommitted" {
			// Uncommitted: compare HEAD vs working tree
			oldText, _ = git.FileAtCommit(repoRoot, "HEAD", filePath)
			fullPath := filepath.Join(repoRoot, filePath)
			data, err := os.ReadFile(fullPath)
			if err == nil {
				newText = string(data)
			}
		} else if status == "A" || status == "?" || status == "??" {
			oldText = ""
			newText, _ = git.FileAtCommit(repoRoot, commit.Hash, filePath)
		} else if status == "D" {
			oldText, _ = git.FileAtCommit(repoRoot, commit.Hash+"~1", filePath)
			newText = ""
		} else {
			oldText, _ = git.FileAtCommit(repoRoot, commit.Hash+"~1", filePath)
			newText, _ = git.FileAtCommit(repoRoot, commit.Hash, filePath)
		}

		title := fmt.Sprintf("%s (%s)", filepath.Base(filePath), commit.ShortHash)
		e.graphDiffView = view.NewDiffView(e.theme, oldText, newText, title, filePath)
		e.graphFocus = 2
		e.statusBar.SetMessage(fmt.Sprintf("Diff: %s  (Esc to go back)", filePath))
	})

	// Select first commit — trigger the same logic as onSelect
	if len(rows) > 0 {
		first := rows[0].Commit
		if first.Hash == "uncommitted" {
			var files []string
			for _, entry := range statusEntries {
				files = append(files, entry.Status+"\t"+entry.Path)
			}
			e.commitDetailView.SetCommit(first, files)
		} else {
			files, _ := git.LoadChangedFiles(repoRoot, first.Hash)
			e.commitDetailView.SetCommit(first, files)
		}
	}

	// Open tab and focus graph
	e.tabs.Open(buffer.NewBuffer(""), "graph")
	e.tabs.Active().Kind = TabKindGraph
	e.sidebarFocus = false
	e.panelFocus = false
	e.syncViewToTab()
	e.statusBar.SetMessage(fmt.Sprintf("Git Graph: %d commits", len(commits)))
}

// graphRefresh reloads the git graph after an operation.
func (e *Editor) graphRefresh() {
	// Close and reopen graph to reflect changes
	e.closeCurrentTab()
	e.gitOpenGraph()
}

// graphGitCommit prompts for commit message and commits from graph view.
func (e *Editor) graphGitCommit() {
	if e.diffTracker == nil {
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
		lines := strings.Split(out, "\n")
		e.statusBar.SetMessage("Committed: " + lines[0])
		e.graphRefresh()
	})
}

// graphGitPush pushes to remote from graph view.
func (e *Editor) graphGitPush() {
	if e.diffTracker == nil {
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

// graphGitPull pulls from remote and refreshes graph.
func (e *Editor) graphGitPull() {
	if e.diffTracker == nil {
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

// graphGitTag creates a tag on the selected commit.
func (e *Editor) graphGitTag() {
	if e.diffTracker == nil || e.graphView == nil {
		return
	}
	commit := e.graphView.SelectedCommit()
	if commit == nil || commit.Hash == "uncommitted" {
		e.statusBar.SetMessage("Select a commit to tag")
		return
	}
	e.inputBar.Show(fmt.Sprintf("Tag name (on %s): ", commit.ShortHash))
	e.inputBar.SetOnSubmit(func(name string) {
		e.inputBar.Hide()
		name = strings.TrimSpace(name)
		if name == "" {
			e.statusBar.SetMessage("Tag aborted: empty name")
			return
		}
		_, err := e.diffTracker.Tag(name, commit.Hash)
		if err != nil {
			e.statusBar.SetMessage(err.Error())
			return
		}
		e.statusBar.SetMessage(fmt.Sprintf("Tagged %s as %s", commit.ShortHash, name))
		e.graphRefresh()
	})
}

// graphGitMerge prompts for branch name and merges.
func (e *Editor) graphGitMerge() {
	if e.diffTracker == nil {
		return
	}
	current := e.diffTracker.CurrentBranch()
	e.inputBar.Show(fmt.Sprintf("Merge into %s from branch: ", current))
	e.inputBar.SetOnSubmit(func(branch string) {
		e.inputBar.Hide()
		branch = strings.TrimSpace(branch)
		if branch == "" {
			e.statusBar.SetMessage("Merge aborted")
			return
		}
		out, err := e.diffTracker.Merge(branch)
		if err != nil {
			e.statusBar.SetMessage(err.Error())
			return
		}
		first := strings.Split(out, "\n")[0]
		e.statusBar.SetMessage("Merge: " + first)
		e.graphRefresh()
	})
}

// graphGitRebase prompts for target and rebases.
func (e *Editor) graphGitRebase() {
	if e.diffTracker == nil {
		return
	}
	current := e.diffTracker.CurrentBranch()
	e.inputBar.Show(fmt.Sprintf("Rebase %s onto: ", current))
	e.inputBar.SetOnSubmit(func(target string) {
		e.inputBar.Hide()
		target = strings.TrimSpace(target)
		if target == "" {
			e.statusBar.SetMessage("Rebase aborted")
			return
		}
		out, err := e.diffTracker.Rebase(target)
		if err != nil {
			e.statusBar.SetMessage(err.Error())
			return
		}
		first := strings.Split(out, "\n")[0]
		e.statusBar.SetMessage("Rebase: " + first)
		e.graphRefresh()
	})
}

// graphGitStash stashes working tree changes.
func (e *Editor) graphGitStash() {
	if e.diffTracker == nil {
		return
	}
	out, err := e.diffTracker.Stash()
	if err != nil {
		e.statusBar.SetMessage(err.Error())
		return
	}
	first := strings.Split(out, "\n")[0]
	e.statusBar.SetMessage("Stash: " + first)
	e.graphRefresh()
}

// graphGitStashPop pops the most recent stash.
func (e *Editor) graphGitStashPop() {
	if e.diffTracker == nil {
		return
	}
	out, err := e.diffTracker.StashPop()
	if err != nil {
		e.statusBar.SetMessage(err.Error())
		return
	}
	first := strings.Split(out, "\n")[0]
	e.statusBar.SetMessage("Stash pop: " + first)
	e.graphRefresh()
}

// graphGitStageAll stages all changes from graph view.
func (e *Editor) graphGitStageAll() {
	if e.diffTracker == nil {
		return
	}
	if err := e.diffTracker.StageAll(); err != nil {
		e.statusBar.SetMessage(err.Error())
		return
	}
	e.statusBar.SetMessage("Staged all changes")
	e.graphRefresh()
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
