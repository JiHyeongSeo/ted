package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// BlameLine holds blame info for a single line.
type BlameLine struct {
	Hash    string // abbreviated commit hash
	Author  string // author name
	Time    time.Time
	Summary string // commit message first line
}

// Blame runs git blame on a file and returns per-line blame info (0-based index).
func (d *DiffTracker) Blame(filePath string) ([]BlameLine, error) {
	cmd := exec.Command("git", "-C", d.repoRoot, "blame", "--porcelain", filePath)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git blame: %w", err)
	}
	return parseBlamePorcelain(string(out))
}

// parseBlamePorcelain parses git blame --porcelain output.
func parseBlamePorcelain(output string) ([]BlameLine, error) {
	lines := strings.Split(output, "\n")
	// First pass: collect commit info
	commitInfo := make(map[string]*BlameLine)
	// Second pass: build ordered list
	var result []BlameLine

	i := 0
	for i < len(lines) {
		line := lines[i]
		if line == "" {
			i++
			continue
		}

		// Each blame entry starts with: <hash> <orig-line> <final-line> [<num-lines>]
		parts := strings.Fields(line)
		if len(parts) < 3 {
			i++
			continue
		}

		hash := parts[0]
		if len(hash) != 40 {
			i++
			continue
		}

		shortHash := hash[:7]

		// Check if we already have this commit's info
		info, exists := commitInfo[hash]
		if !exists {
			info = &BlameLine{Hash: shortHash}
			commitInfo[hash] = info
		}

		// Read header lines until we hit the content line (starts with \t)
		i++
		for i < len(lines) {
			if strings.HasPrefix(lines[i], "\t") {
				i++ // skip content line
				break
			}
			headerLine := lines[i]
			if strings.HasPrefix(headerLine, "author ") {
				info.Author = strings.TrimPrefix(headerLine, "author ")
			} else if strings.HasPrefix(headerLine, "author-time ") {
				if ts, err := strconv.ParseInt(strings.TrimPrefix(headerLine, "author-time "), 10, 64); err == nil {
					info.Time = time.Unix(ts, 0)
				}
			} else if strings.HasPrefix(headerLine, "summary ") {
				info.Summary = strings.TrimPrefix(headerLine, "summary ")
			}
			i++
		}

		result = append(result, *info)
	}

	return result, nil
}

// FormatBlameLine formats a blame line for display in the gutter.
func FormatBlameLine(b BlameLine, maxWidth int) string {
	if b.Hash == "" {
		return ""
	}

	author := b.Author
	if len(author) > 12 {
		author = author[:12]
	}

	age := formatAge(b.Time)

	// Show: hash author date summary
	summary := b.Summary
	remaining := maxWidth - len(b.Hash) - len(author) - len(age) - 4 // 4 spaces
	if remaining > 0 && summary != "" {
		if len(summary) > remaining {
			summary = summary[:remaining-1] + "…"
		}
		text := fmt.Sprintf("%s %s %s %s", b.Hash, author, age, summary)
		if len(text) > maxWidth {
			text = text[:maxWidth]
		}
		return text
	}

	text := fmt.Sprintf("%s %s %s", b.Hash, author, age)
	if len(text) > maxWidth {
		text = text[:maxWidth]
	}
	return text
}

// formatAge returns a human-readable relative time.
func formatAge(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo", int(d.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy", int(d.Hours()/(24*365)))
	}
}
