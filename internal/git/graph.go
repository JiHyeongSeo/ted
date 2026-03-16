package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Commit represents a single git commit with DAG relationships.
type Commit struct {
	Hash      string
	ShortHash string
	Parents   []string
	Author    string
	Date      time.Time
	Message   string   // first line only
	Refs      []string // branch/tag names
}

// ParseCommits parses the output of git log with a custom format.
// Format: short_hash\0full_hash\0parents\0author\0timestamp\0message\0refs
func ParseCommits(raw string) ([]Commit, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	lines := strings.Split(raw, "\n")
	commits := make([]Commit, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\x00")
		if len(parts) < 7 {
			continue
		}

		ts, _ := strconv.ParseInt(parts[4], 10, 64)

		var parents []string
		if parts[2] != "" {
			parents = strings.Split(parts[2], " ")
		}

		var refs []string
		if parts[6] != "" {
			refs = strings.Split(parts[6], ", ")
		}

		commits = append(commits, Commit{
			ShortHash: parts[0],
			Hash:      parts[1],
			Parents:   parents,
			Author:    parts[3],
			Date:      time.Unix(ts, 0),
			Message:   parts[5],
			Refs:      refs,
		})
	}

	return commits, nil
}

// LoadCommits loads commit history from a git repository.
func LoadCommits(repoRoot string, maxCount int) ([]Commit, error) {
	if maxCount <= 0 {
		maxCount = 500
	}
	format := "%h%x00%H%x00%P%x00%an%x00%at%x00%s%x00%D"
	cmd := exec.Command("git", "-C", repoRoot, "log",
		"--all",
		fmt.Sprintf("--max-count=%d", maxCount),
		fmt.Sprintf("--format=%s", format),
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}
	return ParseCommits(string(out))
}

// ShowCommit returns the full git show output for a commit.
func ShowCommit(repoRoot, hash string) (string, error) {
	cmd := exec.Command("git", "-C", repoRoot, "show", "--stat", "--patch", hash)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git show: %w", err)
	}
	return string(out), nil
}

// LoadChangedFiles returns the list of changed files for a commit.
func LoadChangedFiles(repoRoot, hash string) ([]string, error) {
	cmd := exec.Command("git", "-C", repoRoot, "diff-tree",
		"--no-commit-id", "--name-status", "-r", hash)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff-tree: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var files []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}
