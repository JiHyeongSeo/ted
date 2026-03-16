package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// StatusEntry represents a single file's git status.
type StatusEntry struct {
	Status string // "M", "A", "D", "??" etc.
	Path   string
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
		status := strings.TrimSpace(line[:2])
		// Skip 2-char status, then trim any whitespace/tab separator
		path := strings.TrimLeft(line[2:], " \t")
		if path == "" {
			continue
		}
		entries = append(entries, StatusEntry{Status: status, Path: path})
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

// RepoRoot returns the repository root path.
func (d *DiffTracker) RepoRoot() string {
	return d.repoRoot
}
