package git

import (
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/JiHyeongSeo/ted/internal/types"
)

// DiffTracker computes git diff markers for files in a repository.
type DiffTracker struct {
	repoRoot string
}

// NewDiffTracker creates a DiffTracker for the given directory.
// Returns nil, nil if the directory is not inside a git repository.
func NewDiffTracker(dir string) (*DiffTracker, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		// Not a git repo
		return nil, nil
	}
	root := strings.TrimSpace(string(out))
	return &DiffTracker{repoRoot: root}, nil
}

// hunkRe matches @@ -oldStart[,oldCount] +newStart[,newCount] @@
var hunkRe = regexp.MustCompile(`@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// ComputeMarkers returns a map of 0-based line index to GutterMark for the given file.
func (d *DiffTracker) ComputeMarkers(filePath string) (map[int]types.GutterMark, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	// Check for binary file
	diffOut, err := d.runGitDiff(absPath)
	if err != nil {
		// If git diff fails, file might be untracked
		return d.checkUntracked(absPath)
	}

	if strings.Contains(diffOut, "Binary files") {
		return map[int]types.GutterMark{}, nil
	}

	if diffOut == "" {
		// No diff — could be untracked
		return d.checkUntracked(absPath)
	}

	return parseHunks(diffOut), nil
}

// runGitDiff runs git diff --unified=0 HEAD -- <file> and returns stdout.
func (d *DiffTracker) runGitDiff(absPath string) (string, error) {
	cmd := exec.Command("git", "-C", d.repoRoot, "diff", "--unified=0", "HEAD", "--", absPath)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// checkUntracked checks if file is untracked and returns all-added markers.
func (d *DiffTracker) checkUntracked(absPath string) (map[int]types.GutterMark, error) {
	cmd := exec.Command("git", "-C", d.repoRoot, "ls-files", "--error-unmatch", absPath)
	if err := cmd.Run(); err != nil {
		// File is untracked — we need to count its lines
		return d.markAllAdded(absPath)
	}
	// File is tracked, no changes
	return map[int]types.GutterMark{}, nil
}

// markAllAdded reads the file and marks all lines as added.
func (d *DiffTracker) markAllAdded(absPath string) (map[int]types.GutterMark, error) {
	cmd := exec.Command("wc", "-l", absPath)
	out, err := cmd.Output()
	if err != nil {
		return map[int]types.GutterMark{}, nil
	}
	parts := strings.Fields(string(out))
	if len(parts) == 0 {
		return map[int]types.GutterMark{}, nil
	}
	count, err := strconv.Atoi(parts[0])
	if err != nil {
		return map[int]types.GutterMark{}, nil
	}
	// wc -l counts newlines; a file with content but no trailing newline may report one less
	if count == 0 {
		count = 1
	}
	markers := make(map[int]types.GutterMark, count)
	for i := 0; i < count; i++ {
		markers[i] = types.MarkAdded
	}
	return markers, nil
}

// parseHunks parses git diff output and returns markers.
func parseHunks(diffOutput string) map[int]types.GutterMark {
	markers := make(map[int]types.GutterMark)

	matches := hunkRe.FindAllStringSubmatch(diffOutput, -1)
	for _, m := range matches {
		oldCount := 1
		newStart := 1
		newCount := 1

		if v, err := strconv.Atoi(m[1]); err == nil {
			_ = v // oldStart not used directly
		}
		if m[2] != "" {
			if v, err := strconv.Atoi(m[2]); err == nil {
				oldCount = v
			}
		}
		if v, err := strconv.Atoi(m[3]); err == nil {
			newStart = v
		}
		if m[4] != "" {
			if v, err := strconv.Atoi(m[4]); err == nil {
				newCount = v
			}
		}

		if oldCount == 0 {
			// Pure addition
			for i := 0; i < newCount; i++ {
				markers[newStart-1+i] = types.MarkAdded
			}
		} else if newCount == 0 {
			// Pure deletion
			idx := newStart - 1
			if idx < 0 {
				idx = 0
			}
			markers[idx] = types.MarkDeleted
		} else {
			// Modification
			for i := 0; i < newCount; i++ {
				markers[newStart-1+i] = types.MarkModified
			}
		}
	}

	return markers
}

// ParseHunksForTest is exported for testing purposes.
func ParseHunksForTest(diffOutput string) map[int]types.GutterMark {
	return parseHunks(diffOutput)
}
