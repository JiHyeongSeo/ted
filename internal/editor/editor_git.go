package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/JiHyeongSeo/ted/internal/buffer"
	"github.com/JiHyeongSeo/ted/internal/git"
	"github.com/JiHyeongSeo/ted/internal/view"
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

	// Load initial batch of commits
	const initialBatch = 300
	commits, err := git.LoadCommits(e.diffTracker.RepoRoot(), 0, initialBatch)
	if err != nil {
		e.statusBar.SetMessage(err.Error())
		return
	}
	e.graphCommits = commits
	e.graphAllLoaded = len(commits) < initialBatch

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

	// Set up lazy load callback — fires when user scrolls near the bottom
	e.graphView.SetOnNearBottom(func() {
		if !e.graphAllLoaded {
			go e.graphLoadMore()
		}
	})

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
		e.graphDiffView.SetOnCopy(func(text string) {
			e.copyToSystemClipboard(text)
			e.statusBar.SetMessage("Copied to clipboard")
		})
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

// graphLoadMore fetches the next batch of commits and appends them to the graph.
// Must be called from a goroutine; posts a screen event when done.
func (e *Editor) graphLoadMore() {
	if e.diffTracker == nil || e.graphAllLoaded || e.graphView == nil {
		return
	}

	const batchSize = 200
	skip := len(e.graphCommits)
	more, err := git.LoadCommits(e.diffTracker.RepoRoot(), skip, batchSize)
	if err != nil || len(more) == 0 {
		e.graphAllLoaded = true
		if err != nil {
			e.statusBar.SetMessage(err.Error())
		}
		if e.screen != nil {
			e.screen.PostEvent(tcell.NewEventInterrupt(nil))
		}
		return
	}

	e.graphCommits = append(e.graphCommits, more...)
	if len(more) < batchSize {
		e.graphAllLoaded = true
	}

	// Re-layout with full commit set (including the uncommitted virtual commit)
	allCommits := e.graphCommits
	statusEntries, _ := e.diffTracker.Status()
	if len(statusEntries) > 0 {
		uncommitted := git.Commit{
			Hash:      "uncommitted",
			ShortHash: "•••••••",
			Message:   "Uncommitted Changes",
		}
		if len(allCommits) > 0 {
			uncommitted.Parents = []string{allCommits[0].Hash}
		}
		allCommits = append([]git.Commit{uncommitted}, allCommits...)
	}

	rows := git.LayoutGraph(allCommits)
	e.graphView.SetRows(rows)
	e.graphView.ResetNearBottom()

	e.statusBar.SetMessage(fmt.Sprintf("Git Graph: %d commits%s",
		len(e.graphCommits),
		map[bool]string{true: " (all loaded)", false: ""}[e.graphAllLoaded]))

	if e.screen != nil {
		e.screen.PostEvent(tcell.NewEventInterrupt(nil))
	}
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

// graphGitFetch fetches from remote and refreshes graph.
func (e *Editor) graphGitFetch() {
	if e.diffTracker == nil {
		return
	}
	e.statusBar.SetMessage("Fetching...")
	go func() {
		out, err := e.diffTracker.Fetch()
		if err != nil {
			e.statusBar.SetMessage(err.Error())
		} else if out != "" {
			e.statusBar.SetMessage("Fetched: " + strings.Split(out, "\n")[0])
			e.graphRefresh()
		} else {
			e.statusBar.SetMessage("Fetched successfully")
			e.graphRefresh()
		}
		if e.screen != nil {
			e.screen.PostEvent(tcell.NewEventInterrupt(nil))
		}
	}()
}

// graphGitPush shows a branch list and pushes the selected branch to origin.
// Current branch is marked with "* " prefix.
func (e *Editor) graphGitPush() {
	if e.diffTracker == nil {
		return
	}
	branches, err := e.diffTracker.ListBranches()
	if err != nil || len(branches) == 0 {
		e.statusBar.SetMessage("No branches found")
		return
	}
	current := e.diffTracker.CurrentBranch()
	displayed := make([]string, len(branches))
	for i, b := range branches {
		if b == current {
			displayed[i] = "* " + b
		} else {
			displayed[i] = "  " + b
		}
	}
	e.listPicker.Show("Push branch to origin", displayed)
	e.listPicker.SetOnCancel(func() { e.statusBar.SetMessage("Push cancelled") })
	e.listPicker.SetOnSelect(func(item string) {
		branch := strings.TrimLeft(item, "* ")
		e.inputBar.Show(fmt.Sprintf("Push '%s'? (p=push  f=force-with-lease  n=cancel): ", branch))
		e.inputBar.SetOnSubmit(func(answer string) {
			e.inputBar.Hide()
			answer = strings.ToLower(strings.TrimSpace(answer))
			if answer != "p" && answer != "f" {
				e.statusBar.SetMessage("Push cancelled")
				return
			}
			force := answer == "f"
			e.statusBar.SetMessage(fmt.Sprintf("Pushing %s...", branch))
			go func() {
				out, err := e.diffTracker.PushBranch(branch, force)
				if err != nil {
					e.statusBar.SetMessage(err.Error())
				} else if out != "" {
					e.statusBar.SetMessage("Pushed: " + strings.Split(out, "\n")[0])
					e.graphRefresh()
				} else {
					e.statusBar.SetMessage(fmt.Sprintf("Pushed %s successfully", branch))
					e.graphRefresh()
				}
				if e.screen != nil {
					e.screen.PostEvent(tcell.NewEventInterrupt(nil))
				}
			}()
		})
	})
}

// graphGitPushTags shows a tag list and pushes the selected tag to origin.
func (e *Editor) graphGitPushTags() {
	if e.diffTracker == nil {
		return
	}
	tags, err := e.diffTracker.ListTags()
	if err != nil || len(tags) == 0 {
		e.statusBar.SetMessage("No tags found")
		return
	}
	e.listPicker.Show("Push tag to origin", tags)
	e.listPicker.SetOnCancel(func() { e.statusBar.SetMessage("Tag push cancelled") })
	e.listPicker.SetOnSelect(func(tag string) {
		e.statusBar.SetMessage(fmt.Sprintf("Pushing tag %s...", tag))
		go func() {
			_, err := e.diffTracker.PushTag(tag)
			if err != nil {
				e.statusBar.SetMessage(err.Error())
			} else {
				e.statusBar.SetMessage(fmt.Sprintf("Pushed tag '%s' successfully", tag))
				e.graphRefresh()
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
			e.graphRefresh()
		} else {
			e.statusBar.SetMessage("Pulled successfully")
			e.graphRefresh()
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

// graphGitMerge shows branch picker and merges selected branch into current.
// Items are displayed as "branch ──► current" to make direction explicit.
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
	// Show direction: "source ──► current"
	displayed := make([]string, len(others))
	for i, b := range others {
		displayed[i] = fmt.Sprintf("%s  ──►  %s", b, current)
	}
	e.listPicker.Show("Merge  (Enter to execute)", displayed)
	e.listPicker.SetOnSelect(func(item string) {
		branch := strings.SplitN(item, "  ──►  ", 2)[0]
		e.statusBar.SetMessage(fmt.Sprintf("Merging %s into %s...", branch, current))
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
	e.listPicker.SetOnCancel(func() {
		e.statusBar.SetMessage("Merge cancelled")
	})
}

// graphGitRebase shows branch picker and rebases current onto selected branch.
// Items are displayed as "current ──► target" to make direction explicit.
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
	// Show direction: "current ──► target"
	displayed := make([]string, len(others))
	for i, b := range others {
		displayed[i] = fmt.Sprintf("%s  ──►  %s", current, b)
	}
	e.listPicker.Show("Rebase  (Enter to execute)", displayed)
	e.listPicker.SetOnSelect(func(item string) {
		target := strings.SplitN(item, "  ──►  ", 2)[1]
		e.statusBar.SetMessage(fmt.Sprintf("Rebasing %s onto %s...", current, target))
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
	e.listPicker.SetOnCancel(func() {
		e.statusBar.SetMessage("Rebase cancelled")
	})
}

// graphGitDelete is the unified delete flow (d key).
// Step 1: choose Branch or Tag.
// Step 2: pick the target from a list.
// Step 3: choose scope via inputBar (l/r/F/n).
func (e *Editor) graphGitDelete() {
	if e.diffTracker == nil {
		return
	}
	e.listPicker.Show("Delete", []string{"Branch", "Tag"})
	e.listPicker.SetOnCancel(func() {})
	e.listPicker.SetOnSelect(func(kind string) {
		switch kind {
		case "Branch":
			e.graphGitDeleteBranch()
		case "Tag":
			e.graphGitDeleteTag()
		}
	})
}

// graphGitDeleteBranch lists branches and deletes the selected one.
func (e *Editor) graphGitDeleteBranch() {
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
		e.statusBar.SetMessage("No other branches to delete")
		return
	}
	e.listPicker.Show(fmt.Sprintf("Delete branch  (current: %s)", current), others)
	e.listPicker.SetOnCancel(func() { e.statusBar.SetMessage("Branch delete cancelled") })
	e.listPicker.SetOnSelect(func(branch string) {
		e.inputBar.Show(fmt.Sprintf("Delete '%s'? (l=local  r=local+remote  F=force-local  n=cancel): ", branch))
		e.inputBar.SetOnSubmit(func(answer string) {
			e.inputBar.Hide()
			answer = strings.TrimSpace(answer)
			if strings.ToLower(answer) == "n" || answer == "" {
				e.statusBar.SetMessage("Branch delete cancelled")
				return
			}
			force := answer == "F"
			deleteRemote := strings.ToLower(answer) == "r"
			go func() {
				_, err := e.diffTracker.DeleteBranch(branch, force)
				if err != nil {
					e.statusBar.SetMessage(err.Error())
					if e.screen != nil {
						e.screen.PostEvent(tcell.NewEventInterrupt(nil))
					}
					return
				}
				if deleteRemote {
					_, err = e.diffTracker.DeleteRemoteBranch(branch)
					if err != nil {
						e.statusBar.SetMessage(fmt.Sprintf("Local deleted, remote failed: %s", err.Error()))
					} else {
						e.statusBar.SetMessage(fmt.Sprintf("Deleted branch '%s' locally and from remote", branch))
					}
				} else {
					e.statusBar.SetMessage(fmt.Sprintf("Deleted branch '%s' locally", branch))
				}
				e.graphRefresh()
				if e.screen != nil {
					e.screen.PostEvent(tcell.NewEventInterrupt(nil))
				}
			}()
		})
	})
}

// graphGitDeleteTag lists tags and deletes the selected one.
func (e *Editor) graphGitDeleteTag() {
	tags, err := e.diffTracker.ListTags()
	if err != nil || len(tags) == 0 {
		e.statusBar.SetMessage("No tags found")
		return
	}
	e.listPicker.Show("Delete tag", tags)
	e.listPicker.SetOnCancel(func() { e.statusBar.SetMessage("Tag delete cancelled") })
	e.listPicker.SetOnSelect(func(tag string) {
		e.inputBar.Show(fmt.Sprintf("Delete tag '%s'? (l=local  r=local+remote  n=cancel): ", tag))
		e.inputBar.SetOnSubmit(func(answer string) {
			e.inputBar.Hide()
			answer = strings.ToLower(strings.TrimSpace(answer))
			if answer != "l" && answer != "r" {
				e.statusBar.SetMessage("Tag delete cancelled")
				return
			}
			deleteRemote := answer == "r"
			go func() {
				_, err := e.diffTracker.DeleteTag(tag)
				if err != nil {
					e.statusBar.SetMessage(err.Error())
					if e.screen != nil {
						e.screen.PostEvent(tcell.NewEventInterrupt(nil))
					}
					return
				}
				if deleteRemote {
					_, err = e.diffTracker.DeleteRemoteTag(tag)
					if err != nil {
						e.statusBar.SetMessage(fmt.Sprintf("Local deleted, remote failed: %s", err.Error()))
					} else {
						e.statusBar.SetMessage(fmt.Sprintf("Deleted tag '%s' locally and from remote", tag))
					}
				} else {
					e.statusBar.SetMessage(fmt.Sprintf("Deleted tag '%s' locally", tag))
				}
				e.graphRefresh()
				if e.screen != nil {
					e.screen.PostEvent(tcell.NewEventInterrupt(nil))
				}
			}()
		})
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

// graphGitStashPop shows stash list and pops the selected entry.
func (e *Editor) graphGitStashPop() {
	e.showStashPicker("pop")
}

// graphGitStashDrop shows stash list and drops (deletes) the selected entry.
func (e *Editor) graphGitStashDrop() {
	e.showStashPicker("drop")
}

// graphGitBranchMenu is the unified branch menu (b key).
// Actions: Create / Checkout / Set Upstream / Delete
func (e *Editor) graphGitBranchMenu() {
	if e.diffTracker == nil {
		return
	}
	actions := []string{"Create branch", "Checkout", "Set upstream", "Delete branch"}
	e.listPicker.Show("Branch", actions)
	e.listPicker.SetOnCancel(func() {})
	e.listPicker.SetOnSelect(func(action string) {
		switch action {
		case "Create branch":
			e.graphGitCreateBranch()
		case "Checkout":
			e.graphGitCheckout()
		case "Set upstream":
			e.graphGitSetUpstream()
		case "Delete branch":
			e.graphGitDeleteBranch()
		}
	})
}

// graphGitTagMenu is the unified tag menu (t key).
// Actions: Create tag / Push tag / Delete tag
func (e *Editor) graphGitTagMenu() {
	if e.diffTracker == nil {
		return
	}
	actions := []string{"Create tag", "Push tag", "Delete tag"}
	e.listPicker.Show("Tag", actions)
	e.listPicker.SetOnCancel(func() {})
	e.listPicker.SetOnSelect(func(action string) {
		switch action {
		case "Create tag":
			e.graphGitTag()
		case "Push tag":
			e.graphGitPushTags()
		case "Delete tag":
			e.graphGitDeleteTag()
		}
	})
}

// graphGitStashMenu is the unified stash menu (s key).
// Actions: Stash / Pop / Drop
func (e *Editor) graphGitStashMenu() {
	if e.diffTracker == nil {
		return
	}
	actions := []string{"Stash changes", "Pop stash", "Drop stash"}
	e.listPicker.Show("Stash", actions)
	e.listPicker.SetOnCancel(func() {})
	e.listPicker.SetOnSelect(func(action string) {
		switch action {
		case "Stash changes":
			e.graphGitStash()
		case "Pop stash":
			e.graphGitStashPop()
		case "Drop stash":
			e.graphGitStashDrop()
		}
	})
}

// showStashPicker shows a stash list picker, then performs action ("pop" or "drop").
func (e *Editor) showStashPicker(action string) {
	if e.diffTracker == nil {
		return
	}
	stashes, err := e.diffTracker.ListStashes()
	if err != nil || len(stashes) == 0 {
		e.statusBar.SetMessage("No stashes found")
		return
	}
	title := "Pop stash"
	if action == "drop" {
		title = "Drop (delete) stash"
	}
	e.listPicker.Show(title, stashes)
	e.listPicker.SetOnCancel(func() { e.statusBar.SetMessage("Cancelled") })
	e.listPicker.SetOnSelect(func(item string) {
		// item format: "stash@{N}: ..." — extract stash@{N}
		ref := strings.SplitN(item, ":", 2)[0]
		go func() {
			var out string
			var err error
			if action == "drop" {
				out, err = e.diffTracker.StashDropAt(ref)
				if err != nil {
					e.statusBar.SetMessage(err.Error())
				} else {
					_ = out
					e.statusBar.SetMessage(fmt.Sprintf("Dropped %s", ref))
					e.graphRefresh()
				}
			} else {
				e.statusBar.SetMessage(fmt.Sprintf("Popping %s...", ref))
				out, err = e.diffTracker.StashPopAt(ref)
				if err != nil {
					e.statusBar.SetMessage(err.Error())
				} else {
					first := strings.Split(out, "\n")[0]
					e.statusBar.SetMessage("Stash pop: " + first)
					e.graphRefresh()
				}
			}
			if e.screen != nil {
				e.screen.PostEvent(tcell.NewEventInterrupt(nil))
			}
		}()
	})
}

// graphGitCreateBranch creates a new branch.
// If a commit is selected in the graph, offers to branch from that commit.
// Otherwise branches from current HEAD.
// After naming, asks whether to switch to the new branch.
func (e *Editor) graphGitCreateBranch() {
	if e.diffTracker == nil {
		return
	}
	current := e.diffTracker.CurrentBranch()

	// Determine base: selected commit or current HEAD
	base := ""
	baseDesc := fmt.Sprintf("HEAD (%s)", current)
	if e.graphView != nil {
		if commit := e.graphView.SelectedCommit(); commit != nil && commit.Hash != "uncommitted" {
			base = commit.Hash
			baseDesc = commit.ShortHash + " " + commit.Message
		}
	}

	e.inputBar.Show(fmt.Sprintf("New branch from [%s]: ", baseDesc))
	e.inputBar.SetOnSubmit(func(name string) {
		e.inputBar.Hide()
		name = strings.TrimSpace(name)
		if name == "" {
			e.statusBar.SetMessage("Branch creation cancelled")
			return
		}
		e.inputBar.Show(fmt.Sprintf("Switch to '%s' after creating? (y/n): ", name))
		e.inputBar.SetOnSubmit(func(answer string) {
			e.inputBar.Hide()
			checkout := strings.ToLower(strings.TrimSpace(answer)) == "y"
			go func() {
				_, err := e.diffTracker.CreateBranch(name, base, checkout)
				if err != nil {
					e.statusBar.SetMessage(err.Error())
				} else if checkout {
					e.statusBar.SetMessage(fmt.Sprintf("Created and switched to branch '%s'", name))
					e.graphRefresh()
				} else {
					e.statusBar.SetMessage(fmt.Sprintf("Created branch '%s'", name))
					e.graphRefresh()
				}
				if e.screen != nil {
					e.screen.PostEvent(tcell.NewEventInterrupt(nil))
				}
			}()
		})
	})
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

	const blameWidth = 42
	lines := make([]string, len(blameData))
	for i, b := range blameData {
		lines[i] = git.FormatBlameLine(b, blameWidth-1) // -1 for separator
	}

	e.editorView.SetBlameData(lines, blameData, blameWidth)
	e.editorView.SetOnBlameClick(func(hash string) {
		e.gitOpenGraphAtCommit(hash)
	})
	e.statusBar.SetMessage("Blame: on (click hash → graph)")
}

// gitOpenGraphAtCommit opens the git graph and scrolls to the given commit hash.
func (e *Editor) gitOpenGraphAtCommit(shortHash string) {
	// Open graph first
	e.gitOpenGraph()

	// Find and select the commit matching this short hash
	if e.graphView != nil {
		e.graphView.SelectByShortHash(shortHash)
	}
}

// graphGitCheckout shows a combined branch+tag picker and checks out the selected ref.
func (e *Editor) graphGitCheckout() {
	if e.diffTracker == nil {
		return
	}
	current := e.diffTracker.CurrentBranch()

	branches, err := e.diffTracker.ListBranches()
	if err != nil {
		e.statusBar.SetMessage(err.Error())
		return
	}
	tags, err := e.diffTracker.ListTags()
	if err != nil {
		e.statusBar.SetMessage(err.Error())
		return
	}

	var items []string
	for _, b := range branches {
		if b == current {
			items = append(items, "branch  * "+b)
		} else {
			items = append(items, "branch    "+b)
		}
	}
	for _, t := range tags {
		items = append(items, "tag       "+t)
	}

	if len(items) == 0 {
		e.statusBar.SetMessage("No branches or tags found")
		return
	}

	e.listPicker.Show("Checkout branch/tag", items)
	e.listPicker.SetOnSelect(func(selected string) {
		e.listPicker.Hide()
		if selected == "" {
			return
		}
		// Strip prefix ("branch  * ", "branch    ", "tag       ")
		ref := selected
		for _, pfx := range []string{"branch  * ", "branch    ", "tag       "} {
			if strings.HasPrefix(selected, pfx) {
				ref = strings.TrimPrefix(selected, pfx)
				break
			}
		}
		if ref == current {
			e.statusBar.SetMessage("Already on '" + ref + "'")
			return
		}
		go func() {
			_, err := e.diffTracker.Checkout(ref)
			if err != nil {
				e.statusBar.SetMessage(err.Error())
			} else {
				e.statusBar.SetMessage("Switched to '" + ref + "'")
				e.graphRefresh()
			}
			if e.screen != nil {
				e.screen.PostEvent(tcell.NewEventInterrupt(nil))
			}
		}()
	})
}

// graphGitSetUpstream shows a remote branch picker and sets the upstream for the current branch.
func (e *Editor) graphGitSetUpstream() {
	if e.diffTracker == nil {
		return
	}
	current := e.diffTracker.CurrentBranch()
	if current == "" {
		e.statusBar.SetMessage("Not on a branch (detached HEAD)")
		return
	}

	remotes, err := e.diffTracker.ListRemoteBranches()
	if err != nil {
		e.statusBar.SetMessage(err.Error())
		return
	}
	if len(remotes) == 0 {
		e.statusBar.SetMessage("No remote branches found")
		return
	}

	// Pre-select the most likely match (origin/<current>)
	items := make([]string, len(remotes))
	copy(items, remotes)

	e.listPicker.Show(fmt.Sprintf("Set upstream for '%s'", current), items)
	e.listPicker.SetOnSelect(func(selected string) {
		e.listPicker.Hide()
		if selected == "" {
			return
		}
		go func() {
			_, err := e.diffTracker.SetUpstream(current, selected)
			if err != nil {
				e.statusBar.SetMessage(err.Error())
			} else {
				e.statusBar.SetMessage(fmt.Sprintf("'%s' tracks '%s'", current, selected))
			}
			if e.screen != nil {
				e.screen.PostEvent(tcell.NewEventInterrupt(nil))
			}
		}()
	})
}
