package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// StatusEntry represents a single file's git status.
type StatusEntry struct {
	Status string // display status: "M", "A", "D", "??" etc.
	Path   string
	Staged bool // true if file is staged (index has changes)
}

// Status returns the git status of the repository.
func (d *DiffTracker) Status() ([]StatusEntry, error) {
	cmd := exec.Command("git", "-C", d.repoRoot, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}
	var entries []StatusEntry
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if len(line) < 3 {
			continue
		}
		// XY format: X=index(staged), Y=worktree(unstaged)
		x := line[0] // index status
		y := line[1] // worktree status
		path := strings.TrimLeft(line[2:], " \t")
		if path == "" {
			continue
		}
		staged := x != ' ' && x != '?'
		// Display status: prefer worktree status, fall back to index
		status := strings.TrimSpace(string([]byte{x, y}))
		if status == "" {
			continue
		}
		entries = append(entries, StatusEntry{Status: status, Path: path, Staged: staged})
	}
	return entries, nil
}

// StageFile stages a file for commit.
func (d *DiffTracker) StageFile(path string) error {
	cmd := exec.Command("git", "-C", d.repoRoot, "add", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// UnstageFile unstages a file.
func (d *DiffTracker) UnstageFile(path string) error {
	cmd := exec.Command("git", "-C", d.repoRoot, "reset", "HEAD", "--", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git reset: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// StageAll stages all changes.
func (d *DiffTracker) StageAll() error {
	cmd := exec.Command("git", "-C", d.repoRoot, "add", "-A")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add -A: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// Commit creates a commit with the given message.
func (d *DiffTracker) Commit(message string) (string, error) {
	cmd := exec.Command("git", "-C", d.repoRoot, "commit", "-m", message)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git commit: %s", strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// Push pushes to the remote.
func (d *DiffTracker) Push() (string, error) {
	cmd := exec.Command("git", "-C", d.repoRoot, "push")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git push: %s", strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// Pull pulls from the remote.
func (d *DiffTracker) Pull() (string, error) {
	cmd := exec.Command("git", "-C", d.repoRoot, "pull")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git pull: %s", strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// Tag creates a tag on the given commit hash.
func (d *DiffTracker) Tag(name, hash string) (string, error) {
	cmd := exec.Command("git", "-C", d.repoRoot, "tag", name, hash)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git tag: %s", strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// DeleteTag deletes a tag.
func (d *DiffTracker) DeleteTag(name string) (string, error) {
	cmd := exec.Command("git", "-C", d.repoRoot, "tag", "-d", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git tag -d: %s", strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// Merge merges the given branch into the current branch.
func (d *DiffTracker) Merge(branch string) (string, error) {
	cmd := exec.Command("git", "-C", d.repoRoot, "merge", branch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git merge: %s", strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// Rebase rebases the current branch onto the given target.
func (d *DiffTracker) Rebase(target string) (string, error) {
	cmd := exec.Command("git", "-C", d.repoRoot, "rebase", target)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rebase: %s", strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// RebaseAbort aborts an in-progress rebase.
func (d *DiffTracker) RebaseAbort() (string, error) {
	cmd := exec.Command("git", "-C", d.repoRoot, "rebase", "--abort")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rebase --abort: %s", strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// Stash stashes the current working tree changes.
func (d *DiffTracker) Stash() (string, error) {
	cmd := exec.Command("git", "-C", d.repoRoot, "stash")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git stash: %s", strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// StashPop pops the most recent stash.
func (d *DiffTracker) StashPop() (string, error) {
	cmd := exec.Command("git", "-C", d.repoRoot, "stash", "pop")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git stash pop: %s", strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// ListBranches returns local branch names.
func (d *DiffTracker) ListBranches() ([]string, error) {
	cmd := exec.Command("git", "-C", d.repoRoot, "branch", "--format=%(refname:short)")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git branch: %w", err)
	}
	var branches []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

// CurrentBranch returns the current branch name.
func (d *DiffTracker) CurrentBranch() string {
	cmd := exec.Command("git", "-C", d.repoRoot, "branch", "--show-current")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// RepoRoot returns the repository root path.
func (d *DiffTracker) RepoRoot() string {
	return d.repoRoot
}
