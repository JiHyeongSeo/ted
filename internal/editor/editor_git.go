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
			// Show working tree changes with staged info
			var files []string
			var staged []bool
			for _, entry := range statusEntries {
				files = append(files, entry.Status+"\t"+entry.Path)
				staged = append(staged, entry.Staged)
			}
			e.commitDetailView.SetCommitWithStaged(commit, files, staged)
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
			var staged []bool
			for _, entry := range statusEntries {
				files = append(files, entry.Status+"\t"+entry.Path)
				staged = append(staged, entry.Staged)
			}
			e.commitDetailView.SetCommitWithStaged(first, files, staged)
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
		e.statusBar.SetMessage("Committing...")
		go func() {
			out, err := e.diffTracker.Commit(msg)
			if err != nil {
				e.statusBar.SetMessage(err.Error())
			} else {
				lines := strings.Split(out, "\n")
				e.statusBar.SetMessage("Committed: " + lines[0])
				e.graphRefresh()
			}
			if e.screen != nil {
				e.screen.PostEvent(tcell.NewEventInterrupt(nil))
			}
		}()
	})
}

// graphGitPush pushes to remote from graph view with confirmation.
func (e *Editor) graphGitPush() {
	if e.diffTracker == nil {
		return
	}
	branch := e.diffTracker.CurrentBranch()
	e.inputBar.Show(fmt.Sprintf("Push %s to origin? (y/n): ", branch))
	e.inputBar.SetOnSubmit(func(answer string) {
		e.inputBar.Hide()
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			e.statusBar.SetMessage("Push cancelled")
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
	})
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
		go func() {
			_, err := e.diffTracker.Tag(name, commit.Hash)
			if err != nil {
				e.statusBar.SetMessage(err.Error())
			} else {
				e.statusBar.SetMessage(fmt.Sprintf("Tagged %s as %s", commit.ShortHash, name))
				e.graphRefresh()
			}
			if e.screen != nil {
				e.screen.PostEvent(tcell.NewEventInterrupt(nil))
			}
		}()
	})
}

// graphGitMerge shows branch picker and merges selected branch.
func (e *Editor) graphGitMerge() {
	if e.diffTracker == nil {
		return
	}
	branches, err := e.diffTracker.ListBranches()
	if err != nil {
		e.statusBar.SetMessage(err.Error())
		return
	}
	current := e.diffTracker.CurrentBranch()
	// Filter out current branch
	var others []string
	for _, b := range branches {
		if b != current {
			others = append(others, b)
		}
	}
	if len(others) == 0 {
		e.statusBar.SetMessage("No other branches to merge")
		return
	}
	e.listPicker.Show(fmt.Sprintf("Merge into %s", current), others)
	e.listPicker.SetOnSelect(func(branch string) {
		e.inputBar.Show(fmt.Sprintf("Merge %s into %s? (y/n): ", branch, current))
		e.inputBar.SetOnSubmit(func(answer string) {
			e.inputBar.Hide()
			if strings.ToLower(strings.TrimSpace(answer)) != "y" {
				e.statusBar.SetMessage("Merge cancelled")
				return
			}
			e.statusBar.SetMessage("Merging " + branch + "...")
			go func() {
				out, err := e.diffTracker.Merge(branch)
				if err != nil {
					e.statusBar.SetMessage(err.Error())
				} else {
					first := strings.Split(out, "\n")[0]
					e.statusBar.SetMessage("Merge: " + first)
					e.graphRefresh()
				}
				if e.screen != nil {
					e.screen.PostEvent(tcell.NewEventInterrupt(nil))
				}
			}()
		})
	})
	e.listPicker.SetOnCancel(func() {
		e.statusBar.SetMessage("Merge cancelled")
	})
}

// graphGitRebase shows branch picker and rebases onto selected branch.
func (e *Editor) graphGitRebase() {
	if e.diffTracker == nil {
		return
	}
	branches, err := e.diffTracker.ListBranches()
	if err != nil {
		e.statusBar.SetMessage(err.Error())
		return
	}
	current := e.diffTracker.CurrentBranch()
	var others []string
	for _, b := range branches {
		if b != current {
			others = append(others, b)
		}
	}
	if len(others) == 0 {
		e.statusBar.SetMessage("No other branches to rebase onto")
		return
	}
	e.listPicker.Show(fmt.Sprintf("Rebase %s onto", current), others)
	e.listPicker.SetOnSelect(func(target string) {
		e.inputBar.Show(fmt.Sprintf("Rebase %s onto %s? (y/n): ", current, target))
		e.inputBar.SetOnSubmit(func(answer string) {
			e.inputBar.Hide()
			if strings.ToLower(strings.TrimSpace(answer)) != "y" {
				e.statusBar.SetMessage("Rebase cancelled")
				return
			}
			e.statusBar.SetMessage("Rebasing onto " + target + "...")
			go func() {
				out, err := e.diffTracker.Rebase(target)
				if err != nil {
					e.statusBar.SetMessage(err.Error())
				} else {
					first := strings.Split(out, "\n")[0]
					e.statusBar.SetMessage("Rebase: " + first)
					e.graphRefresh()
				}
				if e.screen != nil {
					e.screen.PostEvent(tcell.NewEventInterrupt(nil))
				}
			}()
		})
	})
	e.listPicker.SetOnCancel(func() {
		e.statusBar.SetMessage("Rebase cancelled")
	})
}

// graphGitStash stashes working tree changes.
func (e *Editor) graphGitStash() {
	if e.diffTracker == nil {
		return
	}
	e.statusBar.SetMessage("Stashing...")
	go func() {
		out, err := e.diffTracker.Stash()
		if err != nil {
			e.statusBar.SetMessage(err.Error())
		} else {
			first := strings.Split(out, "\n")[0]
			e.statusBar.SetMessage("Stash: " + first)
			e.graphRefresh()
		}
		if e.screen != nil {
			e.screen.PostEvent(tcell.NewEventInterrupt(nil))
		}
	}()
}

// graphGitStashPop pops the most recent stash.
func (e *Editor) graphGitStashPop() {
	if e.diffTracker == nil {
		return
	}
	e.statusBar.SetMessage("Popping stash...")
	go func() {
		out, err := e.diffTracker.StashPop()
		if err != nil {
			e.statusBar.SetMessage(err.Error())
		} else {
			first := strings.Split(out, "\n")[0]
			e.statusBar.SetMessage("Stash pop: " + first)
			e.graphRefresh()
		}
		if e.screen != nil {
			e.screen.PostEvent(tcell.NewEventInterrupt(nil))
		}
	}()
}

// graphGitStageAll stages all changes from graph view.
func (e *Editor) graphGitStageAll() {
	if e.diffTracker == nil {
		return
	}
	e.statusBar.SetMessage("Staging all...")
	go func() {
		if err := e.diffTracker.StageAll(); err != nil {
			e.statusBar.SetMessage(err.Error())
		} else {
			e.statusBar.SetMessage("Staged all changes")
		}
		e.sendGraphFileUpdate()
	}()
}

// graphGitUnstageFile unstages the currently selected file.
func (e *Editor) graphGitUnstageFile() {
	if e.diffTracker == nil || e.commitDetailView == nil {
		return
	}
	fileLine := e.commitDetailView.SelectedFile()
	if fileLine == "" {
		return
	}
	parts := strings.SplitN(fileLine, "\t", 2)
	if len(parts) < 2 {
		return
	}
	filePath := parts[1]

	go func() {
		if err := e.diffTracker.UnstageFile(filePath); err != nil {
			e.statusBar.SetMessage(err.Error())
		} else {
			e.statusBar.SetMessage(fmt.Sprintf("Unstaged: %s", filePath))
		}
		e.sendGraphFileUpdate()
	}()
}

// graphRefreshUncommitted refreshes the uncommitted file list in the detail view.
func (e *Editor) graphRefreshUncommitted() {
	if e.graphView == nil || e.commitDetailView == nil {
		return
	}
	commit := e.graphView.SelectedCommit()
	if commit != nil && commit.Hash == "uncommitted" {
		entries, _ := e.diffTracker.Status()
		var files []string
		var staged []bool
		for _, entry := range entries {
			files = append(files, entry.Status+"\t"+entry.Path)
			staged = append(staged, entry.Staged)
		}
		e.commitDetailView.UpdateFilesWithStaged(files, staged)
	}
}

// sendGraphFileUpdate runs git status in the goroutine and sends results to main thread via channel.
func (e *Editor) sendGraphFileUpdate() {
	if e.diffTracker == nil {
		return
	}
	entries, _ := e.diffTracker.Status()
	var files []string
	var staged []bool
	for _, entry := range entries {
		files = append(files, entry.Status+"\t"+entry.Path)
		staged = append(staged, entry.Staged)
	}
	// Non-blocking send (drop if channel full — next update will refresh)
	select {
	case e.graphFileUpdates <- graphFileUpdate{files: files, staged: staged}:
	default:
	}
	if e.screen != nil {
		e.screen.PostEvent(tcell.NewEventInterrupt(nil))
	}
}

// graphGitStageFile stages the currently selected file in the commit detail view.
func (e *Editor) graphGitStageFile() {
	if e.diffTracker == nil || e.commitDetailView == nil {
		return
	}
	fileLine := e.commitDetailView.SelectedFile()
	if fileLine == "" {
		return
	}
	parts := strings.SplitN(fileLine, "\t", 2)
	if len(parts) < 2 {
		return
	}
	filePath := parts[1]

	go func() {
		err := e.diffTracker.StageFile(filePath)
		if err != nil {
			e.statusBar.SetMessage(fmt.Sprintf("Stage error: %s", err.Error()))
		} else {
			e.statusBar.SetMessage(fmt.Sprintf("Staged: %s", filePath))
		}
		e.sendGraphFileUpdate()
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
