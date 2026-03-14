# Git Gutter Diff Markers Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show git diff status (added/modified/deleted) as line number background colors in EditorView.

**Architecture:** New `internal/git/` package runs `git diff --unified=0 HEAD` and parses hunk headers into a line→marker map. EditorView renders markers as line number background colors. Editor orchestrates: computes markers on file save and tab switch.

**Tech Stack:** Go, tcell, `os/exec` for git CLI

**Spec:** `docs/superpowers/specs/2026-03-14-git-gutter-design.md`

---

## Chunk 1: Core Types and Diff Parsing

### Task 1: GutterMark type in `internal/types`

**Files:**
- Create: `internal/types/gutter.go`

- [ ] **Step 1: Create the GutterMark type**

```go
package types

// GutterMark represents git diff status for a line in the gutter.
type GutterMark int

const (
	MarkNone     GutterMark = iota
	MarkAdded
	MarkModified
	MarkDeleted
)
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /home/seoji/local/ted && go build ./internal/types/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/types/gutter.go
git commit -m "feat(git): add GutterMark type to types package"
```

### Task 2: DiffTracker with hunk parsing

**Files:**
- Create: `internal/git/diff.go`
- Create: `internal/git/diff_test.go`

- [ ] **Step 1: Write tests for hunk header parsing**

Create `internal/git/diff_test.go`:

```go
package git

import (
	"testing"

	"github.com/seoji/ted/internal/types"
)

func TestParseHunks_Added(t *testing.T) {
	// @@ -10,0 +11,3 @$ — 3 lines added at line 11 (1-based)
	output := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -10,0 +11,3 @@ func main()
+line1
+line2
+line3
`
	markers := parseGitDiff(output)
	// 0-based: lines 10, 11, 12
	for _, line := range []int{10, 11, 12} {
		if markers[line] != types.MarkAdded {
			t.Errorf("line %d: want MarkAdded, got %v", line, markers[line])
		}
	}
}

func TestParseHunks_Modified(t *testing.T) {
	// @@ -5,2 +5,2 @$ — 2 lines modified at line 5
	output := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -5,2 +5,2 @@ func foo()
-old1
-old2
+new1
+new2
`
	markers := parseGitDiff(output)
	for _, line := range []int{4, 5} { // 0-based
		if markers[line] != types.MarkModified {
			t.Errorf("line %d: want MarkModified, got %v", line, markers[line])
		}
	}
}

func TestParseHunks_Deleted(t *testing.T) {
	// @@ -5,3 +4,0 $$ — 3 lines deleted, newStart=4 (1-based)
	output := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -5,3 +4,0 @@ func foo()
-del1
-del2
-del3
`
	markers := parseGitDiff(output)
	// 0-based: max(0, 4-1) = 3
	if markers[3] != types.MarkDeleted {
		t.Errorf("line 3: want MarkDeleted, got %v", markers[3])
	}
}

func TestParseHunks_DeletedAtStart(t *testing.T) {
	// @@ -1,2 +0,0 $$ — deletion at beginning
	output := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +0,0 @@
-line1
-line2
`
	markers := parseGitDiff(output)
	if markers[0] != types.MarkDeleted {
		t.Errorf("line 0: want MarkDeleted, got %v", markers[0])
	}
}

func TestParseHunks_CountOmitted(t *testing.T) {
	// @@ -10 +10 $$ — count omitted means 1
	output := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -10 +10 @@ func foo()
-old
+new
`
	markers := parseGitDiff(output)
	if markers[9] != types.MarkModified { // 0-based
		t.Errorf("line 9: want MarkModified, got %v", markers[9])
	}
}

func TestParseHunks_MultipleHunks(t *testing.T) {
	output := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -3,0 +4,1 @@ package main
+new
@@ -10,1 +12,1 @@ func main()
-old
+modified
`
	markers := parseGitDiff(output)
	if markers[3] != types.MarkAdded {
		t.Errorf("line 3: want MarkAdded, got %v", markers[3])
	}
	if markers[11] != types.MarkModified {
		t.Errorf("line 11: want MarkModified, got %v", markers[11])
	}
}

func TestParseHunks_Empty(t *testing.T) {
	markers := parseGitDiff("")
	if len(markers) != 0 {
		t.Errorf("want empty map, got %d entries", len(markers))
	}
}

func TestParseHunks_Binary(t *testing.T) {
	output := "Binary files a/image.png and b/image.png differ\n"
	markers := parseGitDiff(output)
	if len(markers) != 0 {
		t.Errorf("want empty map for binary, got %d entries", len(markers))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/seoji/local/ted && go test ./internal/git/ -v`
Expected: compilation error (parseGitDiff not defined)

- [ ] **Step 3: Implement DiffTracker and parseGitDiff**

Create `internal/git/diff.go`:

```go
package git

import (
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/seoji/ted/internal/types"
)

// DiffTracker computes git diff markers for files.
type DiffTracker struct {
	repoRoot string
}

// NewDiffTracker creates a DiffTracker for the given project root.
// Returns nil, nil if the directory is not inside a git repository.
func NewDiffTracker(projectRoot string) (*DiffTracker, error) {
	cmd := exec.Command("git", "-C", projectRoot, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return nil, nil // not a git repo
	}
	root := strings.TrimSpace(string(out))
	return &DiffTracker{repoRoot: root}, nil
}

// ComputeMarkers returns gutter markers for the given file path.
// Returns a map of 0-based line index → GutterMark.
func (dt *DiffTracker) ComputeMarkers(filePath string) (map[int]types.GutterMark, error) {
	relPath, err := filepath.Rel(dt.repoRoot, filePath)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("git", "-C", dt.repoRoot, "diff", "--unified=0", "HEAD", "--", relPath)
	out, err := cmd.Output()
	if err != nil {
		// Exit code 1 can mean untracked file — check if tracked
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 128 {
			return nil, nil // no HEAD yet (empty repo)
		}
	}

	output := string(out)
	if output == "" {
		return dt.checkUntracked(filePath, relPath)
	}

	markers := parseGitDiff(output)
	return markers, nil
}

// checkUntracked checks if a file is untracked and marks all lines as added.
func (dt *DiffTracker) checkUntracked(filePath, relPath string) (map[int]types.GutterMark, error) {
	cmd := exec.Command("git", "-C", dt.repoRoot, "ls-files", relPath)
	out, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	if strings.TrimSpace(string(out)) == "" {
		// Untracked: count lines and mark all as added
		content, err := exec.Command("wc", "-l", filePath).Output()
		if err != nil {
			return nil, nil
		}
		lineStr := strings.Fields(strings.TrimSpace(string(content)))
		if len(lineStr) == 0 {
			return nil, nil
		}
		lineCount, err := strconv.Atoi(lineStr[0])
		if err != nil {
			return nil, nil
		}
		markers := make(map[int]types.GutterMark, lineCount)
		for i := 0; i < lineCount; i++ {
			markers[i] = types.MarkAdded
		}
		return markers, nil
	}
	return nil, nil // tracked, no changes
}

var hunkRe = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// parseGitDiff parses git diff --unified=0 output into gutter markers.
func parseGitDiff(output string) map[int]types.GutterMark {
	markers := make(map[int]types.GutterMark)
	for _, line := range strings.Split(output, "\n") {
		m := hunkRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		oldCount := 1
		if m[2] != "" {
			oldCount, _ = strconv.Atoi(m[2])
		}
		newStart, _ := strconv.Atoi(m[3])
		newCount := 1
		if m[4] != "" {
			newCount, _ = strconv.Atoi(m[4])
		}

		if oldCount == 0 {
			// Pure addition
			for i := 0; i < newCount; i++ {
				markers[newStart-1+i] = types.MarkAdded
			}
		} else if newCount == 0 {
			// Pure deletion — mark at max(0, newStart-1)
			line := newStart - 1
			if line < 0 {
				line = 0
			}
			markers[line] = types.MarkDeleted
		} else {
			// Modification
			for i := 0; i < newCount; i++ {
				markers[newStart-1+i] = types.MarkModified
			}
		}
	}
	return markers
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/seoji/local/ted && go test ./internal/git/ -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/git/
git commit -m "feat(git): add DiffTracker with hunk parsing and tests"
```

## Chunk 2: Theme, EditorView, and Editor Integration

### Task 3: Add git colors to theme

**Files:**
- Modify: `internal/syntax/theme.go:53-69` (DefaultTheme UI map)

- [ ] **Step 1: Add git color keys to DefaultTheme()**

In `internal/syntax/theme.go`, add 3 entries to the `UI` map inside `DefaultTheme()`, after the `"matchHighlight"` entry:

```go
"gitAdded":    "#2d4a2d",
"gitModified": "#0c3d4d",
"gitDeleted":  "#4d1a1a",
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /home/seoji/local/ted && go build ./internal/syntax/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/syntax/theme.go
git commit -m "feat(git): add gitAdded/Modified/Deleted colors to default theme"
```

### Task 4: EditorView gutter marker rendering

**Files:**
- Modify: `internal/view/editorview.go:12` (add types import if needed)
- Modify: `internal/view/editorview.go:22-37` (add gutterMarkers field)
- Modify: `internal/view/editorview.go:107-116` (line number rendering)
- Add method: `SetGutterMarkers`

- [ ] **Step 1: Add gutterMarkers field to EditorView struct**

After `searchHighlights []SearchHighlight` (line 36), add:

```go
gutterMarkers    map[int]types.GutterMark // git diff gutter markers (0-based line → mark)
```

- [ ] **Step 2: Add SetGutterMarkers method**

After the `ClearSearchHighlights()` method (line 944), add:

```go
// SetGutterMarkers sets the git diff gutter markers.
func (e *EditorView) SetGutterMarkers(markers map[int]types.GutterMark) {
	e.gutterMarkers = markers
}
```

- [ ] **Step 3: Modify Render() to apply gutter background colors**

In `Render()`, replace the line number drawing block (lines 108-116):

```go
		// Draw line number
		lineNumStyle := e.theme.UIStyle("linenumber")
		if lineNum == e.cursor.Line {
			lineNumStyle = e.theme.UIStyle("linenumber.active")
		}
```

With:

```go
		// Draw line number
		lineNumStyle := e.theme.UIStyle("linenumber")
		if lineNum == e.cursor.Line {
			lineNumStyle = e.theme.UIStyle("linenumber.active")
		}
		// Apply git gutter marker background
		if mark, ok := e.gutterMarkers[lineNum]; ok && mark != types.MarkNone {
			var colorKey string
			switch mark {
			case types.MarkAdded:
				colorKey = "gitAdded"
			case types.MarkModified:
				colorKey = "gitModified"
			case types.MarkDeleted:
				colorKey = "gitDeleted"
			}
			if colorKey != "" {
				if hex := e.theme.UI[colorKey]; hex != "" {
					lineNumStyle = lineNumStyle.Background(e.theme.ResolveColor(hex))
				}
			}
		}
```

Note: The `types` package is already imported in editorview.go (line 11).

- [ ] **Step 4: Verify it compiles**

Run: `cd /home/seoji/local/ted && go build ./internal/view/`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add internal/view/editorview.go
git commit -m "feat(git): render gutter markers as line number background colors"
```

### Task 5: Editor integration — DiffTracker wiring

**Files:**
- Modify: `internal/editor/editor.go:17` (add git import)
- Modify: `internal/editor/editor.go:22-53` (add diffTracker field)
- Modify: `internal/editor/editor.go:56-100` (initialize in New())
- Modify: `internal/editor/editor.go:603-614` (syncViewToTab)
- Modify: `internal/editor/editor.go:769-778` (file.save)

- [ ] **Step 1: Add import and field**

Add to imports:

```go
"github.com/seoji/ted/internal/git"
```

Add field to Editor struct, after `pythonEnv`:

```go
diffTracker *git.DiffTracker
```

- [ ] **Step 2: Add updateGutterMarkers helper**

After `syncTabFromView()` (line 623), add:

```go
func (e *Editor) updateGutterMarkers() {
	if e.diffTracker == nil || e.editorView == nil {
		return
	}
	tab := e.tabs.Active()
	if tab == nil || tab.Buffer.Path() == "" {
		return
	}
	markers, err := e.diffTracker.ComputeMarkers(tab.Buffer.Path())
	if err != nil {
		return
	}
	e.editorView.SetGutterMarkers(markers)
}
```

- [ ] **Step 3: Initialize DiffTracker in New()**

In the `New()` function, after `e.recentFiles = LoadRecentFiles()` (line 90), add:

```go
e.diffTracker, _ = git.NewDiffTracker("")
```

Note: the projectRoot is set later when opening a directory. We'll update it then.

- [ ] **Step 4: Set DiffTracker when projectRoot is set**

Find where `e.projectRoot` is assigned (in OpenDirectory or similar). After that assignment, add:

```go
e.diffTracker, _ = git.NewDiffTracker(e.projectRoot)
```

- [ ] **Step 5: Call updateGutterMarkers after syncViewToTab()**

In `syncViewToTab()`, add at the end (after line 613 `e.editorView.SetScrollY(tab.ScrollY)`):

```go
e.updateGutterMarkers()
```

- [ ] **Step 6: Call updateGutterMarkers after file.save**

In `ExecuteCommand()`, in the `"file.save"` case, after `lsp.DidSave(...)` block (line 777), add:

```go
e.updateGutterMarkers()
```

- [ ] **Step 7: Verify it compiles and runs**

Run: `cd /home/seoji/local/ted && go build ./cmd/ted/`
Expected: no errors

- [ ] **Step 8: Commit**

```bash
git add internal/editor/editor.go
git commit -m "feat(git): wire DiffTracker to editor save and tab switch"
```

### Task 6: Manual testing

- [ ] **Step 1: Build and run**

```bash
cd /home/seoji/local/ted && CGO_ENABLED=1 go build ./cmd/ted/ && ./ted .
```

- [ ] **Step 2: Verify gutter markers**

1. Open a tracked file, make changes, save → modified lines should show blue gutter
2. Add new lines, save → added lines should show green gutter
3. Delete lines, save → line after deletion should show red gutter
4. Open an untracked file → all lines should show green gutter
5. Open a file with no changes → no gutter colors
6. Switch tabs → markers should update for each file

- [ ] **Step 3: Final commit with any fixes**

```bash
git add -A
git commit -m "feat(git): complete git gutter diff markers (M2)"
```
