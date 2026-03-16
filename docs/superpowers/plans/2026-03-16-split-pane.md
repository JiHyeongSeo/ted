# Split Pane Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add vertical 2-pane split to ted, allowing two editor views side by side with independent cursor/scroll

**Architecture:** Add split state to Layout (left/right regions + separator), manage two EditorView instances in Editor, track active pane for focus/event routing. Each pane has its own tab association and view state.

**Tech Stack:** Go, tcell/v2, existing view.Layout and view.EditorView

---

## Current State

- `internal/view/layout.go`: Computes single `"editor"` region
- `internal/editor/editor.go`: Single `editorView *view.EditorView` field
- `internal/editor/tab.go`: TabManager with single active tab
- `syncViewToTab()` / `syncTabFromView()`: Copy cursor/scroll between tab and view

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/view/layout.go` | Modify | Add split mode, compute left/right/separator regions |
| `internal/editor/split.go` | Create | SplitManager: pane state, active pane, split/close logic |
| `internal/editor/editor.go` | Modify | Use SplitManager, route events/render to active pane |
| `internal/editor/commands_builtin.go` | Modify | Register split commands |
| `configs/keybindings.json` | Modify | Add split keybindings |

---

## Chunk 1: Layout Split Support

### Task 1: Extend Layout to compute split regions

**Files:**
- Modify: `internal/view/layout.go`
- Test: `internal/view/layout_test.go`

- [ ] **Step 1: Write failing test for split layout**

```go
func TestLayoutSplit(t *testing.T) {
	l := NewLayout()
	l.SetSplitMode(true)
	regions := l.Compute(80, 24)

	left, ok := regions["editor.left"]
	if !ok {
		t.Fatal("expected editor.left region")
	}
	right, ok := regions["editor.right"]
	if !ok {
		t.Fatal("expected editor.right region")
	}
	sep, ok := regions["editor.separator"]
	if !ok {
		t.Fatal("expected editor.separator region")
	}

	// Left + separator + right should equal total editor width
	if sep.Width != 1 {
		t.Errorf("separator width should be 1, got %d", sep.Width)
	}
	if left.Width+sep.Width+right.Width != left.Width+1+right.Width {
		t.Error("widths don't add up")
	}
	if left.X+left.Width != sep.X {
		t.Error("separator should be adjacent to left pane")
	}
	if sep.X+1 != right.X {
		t.Error("right pane should be adjacent to separator")
	}
}

func TestLayoutNoSplit(t *testing.T) {
	l := NewLayout()
	// splitMode defaults to false
	regions := l.Compute(80, 24)

	if _, ok := regions["editor"]; !ok {
		t.Fatal("expected single editor region when not split")
	}
	if _, ok := regions["editor.left"]; ok {
		t.Error("should not have editor.left when not split")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/seoji/local/ted && go test ./internal/view/ -run TestLayoutSplit -v`
Expected: FAIL — `SetSplitMode` undefined

- [ ] **Step 3: Add split fields and update Compute**

Add to Layout struct:
```go
type Layout struct {
	// ... existing fields ...
	splitMode  bool
	splitRatio float64 // 0.0-1.0, how much left pane gets (default 0.5)
}
```

Update NewLayout:
```go
func NewLayout() *Layout {
	return &Layout{
		// ... existing ...
		splitRatio: 0.5,
	}
}
```

Add setters/getters:
```go
func (l *Layout) SetSplitMode(v bool)    { l.splitMode = v }
func (l *Layout) SplitMode() bool        { return l.splitMode }
func (l *Layout) SetSplitRatio(r float64) {
	if r < 0.2 { r = 0.2 }
	if r > 0.8 { r = 0.8 }
	l.splitRatio = r
}
```

In `Compute()`, replace the single `regions["editor"]` line with:
```go
	if l.splitMode {
		sepWidth := 1
		leftWidth := int(float64(editorWidth-sepWidth) * l.splitRatio)
		rightWidth := editorWidth - sepWidth - leftWidth
		editorX := sidebarWidth + separatorWidth

		regions["editor.left"] = types.Rect{
			X: editorX, Y: y, Width: leftWidth, Height: editorHeight,
		}
		regions["editor.separator"] = types.Rect{
			X: editorX + leftWidth, Y: y, Width: 1, Height: editorHeight,
		}
		regions["editor.right"] = types.Rect{
			X: editorX + leftWidth + sepWidth, Y: y, Width: rightWidth, Height: editorHeight,
		}
	} else {
		regions["editor"] = types.Rect{
			X: sidebarWidth + separatorWidth, Y: y, Width: editorWidth, Height: editorHeight,
		}
	}
```

Also update panel region to span full editor width (already does via editorWidth).

- [ ] **Step 4: Run tests**

Run: `cd /home/seoji/local/ted && go test ./internal/view/ -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/view/layout.go internal/view/layout_test.go
git commit -m "feat(layout): add split mode with left/right editor regions"
```

---

## Chunk 2: SplitManager

### Task 2: Create SplitManager to track pane state

**Files:**
- Create: `internal/editor/split.go`
- Test: `internal/editor/split_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/editor/split_test.go
package editor

import (
	"testing"

	"github.com/seoji/ted/internal/buffer"
)

func TestSplitManagerDefault(t *testing.T) {
	sm := NewSplitManager()
	if sm.IsSplit() {
		t.Error("should not be split initially")
	}
	if sm.ActivePane() != PaneMain {
		t.Errorf("expected PaneMain, got %d", sm.ActivePane())
	}
}

func TestSplitManagerSplit(t *testing.T) {
	sm := NewSplitManager()
	buf := buffer.New("")
	sm.Split(buf, "go")
	if !sm.IsSplit() {
		t.Error("should be split after Split()")
	}
	if sm.RightTab() == nil {
		t.Error("right tab should be set")
	}
}

func TestSplitManagerClose(t *testing.T) {
	sm := NewSplitManager()
	buf := buffer.New("")
	sm.Split(buf, "go")
	sm.SetActivePane(PaneRight)
	sm.CloseSplit()
	if sm.IsSplit() {
		t.Error("should not be split after CloseSplit()")
	}
	if sm.ActivePane() != PaneMain {
		t.Error("should return to PaneMain")
	}
}

func TestSplitManagerFocusToggle(t *testing.T) {
	sm := NewSplitManager()
	buf := buffer.New("")
	sm.Split(buf, "go")

	if sm.ActivePane() != PaneLeft {
		t.Error("should default to left after split")
	}
	sm.FocusOther()
	if sm.ActivePane() != PaneRight {
		t.Error("should switch to right")
	}
	sm.FocusOther()
	if sm.ActivePane() != PaneLeft {
		t.Error("should switch back to left")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/seoji/local/ted && go test ./internal/editor/ -run TestSplitManager -v`

- [ ] **Step 3: Implement SplitManager**

```go
// internal/editor/split.go
package editor

import (
	"github.com/seoji/ted/internal/buffer"
	"github.com/seoji/ted/internal/types"
)

// Pane identifies which pane is active.
type Pane int

const (
	PaneMain  Pane = iota // no split, single editor
	PaneLeft              // left pane in split mode
	PaneRight             // right pane in split mode
)

// PaneState holds per-pane view state.
type PaneState struct {
	Buffer   *buffer.Buffer
	Cursor   types.Position
	ScrollY  int
	ScrollX  int
	Language string
}

// SplitManager manages the split pane state.
type SplitManager struct {
	split      bool
	activePane Pane
	rightPane  *PaneState // nil when not split; left pane uses main TabManager
}

func NewSplitManager() *SplitManager {
	return &SplitManager{
		activePane: PaneMain,
	}
}

func (sm *SplitManager) IsSplit() bool {
	return sm.split
}

func (sm *SplitManager) ActivePane() Pane {
	return sm.activePane
}

func (sm *SplitManager) SetActivePane(p Pane) {
	sm.activePane = p
}

// Split opens a right pane with the given buffer.
// The left pane continues showing the current tab.
func (sm *SplitManager) Split(buf *buffer.Buffer, language string) {
	sm.split = true
	sm.activePane = PaneLeft
	sm.rightPane = &PaneState{
		Buffer:   buf,
		Language: language,
	}
}

// SplitWithCurrentBuffer splits showing the same buffer in both panes.
func (sm *SplitManager) SplitWithCurrentBuffer(buf *buffer.Buffer, language string) {
	sm.Split(buf, language)
}

// CloseSplit closes the split and returns to single pane.
func (sm *SplitManager) CloseSplit() {
	sm.split = false
	sm.activePane = PaneMain
	sm.rightPane = nil
}

// FocusOther switches focus to the other pane.
func (sm *SplitManager) FocusOther() {
	if !sm.split {
		return
	}
	if sm.activePane == PaneLeft {
		sm.activePane = PaneRight
	} else {
		sm.activePane = PaneLeft
	}
}

// RightTab returns the right pane state, or nil.
func (sm *SplitManager) RightTab() *PaneState {
	return sm.rightPane
}
```

- [ ] **Step 4: Run tests**

Run: `cd /home/seoji/local/ted && go test ./internal/editor/ -run TestSplitManager -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/editor/split.go internal/editor/split_test.go
git commit -m "feat: add SplitManager for pane state tracking"
```

---

## Chunk 3: Editor Integration

### Task 3: Wire SplitManager into Editor

**Files:**
- Modify: `internal/editor/editor.go` (struct, New, render, sync, event handling)

This is the largest task. The key changes:

- [ ] **Step 1: Add split fields to Editor struct**

```go
type Editor struct {
	// ... existing fields ...
	splitManager   *SplitManager
	rightEditorView *view.EditorView // nil when not split
}
```

In `New()`:
```go
	e.splitManager = NewSplitManager()
```

- [ ] **Step 2: Update render() for split mode**

Find the section that renders the editor view (around line 585). Replace with:

```go
	if e.splitManager.IsSplit() {
		if r, ok := regions["editor.left"]; ok {
			e.editorView.SetBounds(r)
			e.editorView.Render(e.screen)
		}
		if r, ok := regions["editor.separator"]; ok {
			// Draw separator line
			sepStyle := e.theme.UIStyle("panel").Foreground(tcell.ColorGray)
			for y := r.Y; y < r.Y+r.Height; y++ {
				e.screen.SetContent(r.X, y, '│', nil, sepStyle)
			}
		}
		if r, ok := regions["editor.right"]; ok && e.rightEditorView != nil {
			e.rightEditorView.SetBounds(r)
			e.rightEditorView.Render(e.screen)
		}
	} else {
		if r, ok := regions["editor"]; ok {
			e.editorView.SetBounds(r)
			e.editorView.Render(e.screen)
		}
	}
```

- [ ] **Step 3: Update event handling for active pane**

In the key event handling, where `e.editorView.HandleEvent(ev)` is called, replace with:

```go
	activeView := e.activeEditorView()
	if activeView != nil {
		activeView.HandleEvent(ev)
	}
```

Add helper:
```go
func (e *Editor) activeEditorView() *view.EditorView {
	if e.splitManager.IsSplit() && e.splitManager.ActivePane() == PaneRight {
		return e.rightEditorView
	}
	return e.editorView
}
```

- [ ] **Step 4: Update syncViewToTab and syncTabFromView for split**

Add right pane sync methods:
```go
func (e *Editor) syncRightView() {
	ps := e.splitManager.RightTab()
	if ps == nil {
		e.rightEditorView = nil
		return
	}
	e.rightEditorView = view.NewEditorView(ps.Buffer, e.theme)
	e.rightEditorView.SetLanguage(ps.Language)
	e.rightEditorView.SetCursorPosition(ps.Cursor)
	e.rightEditorView.SetScrollY(ps.ScrollY)
}

func (e *Editor) syncRightTab() {
	ps := e.splitManager.RightTab()
	if ps == nil || e.rightEditorView == nil {
		return
	}
	ps.Cursor = e.rightEditorView.CursorPosition()
	ps.ScrollY, ps.ScrollX = e.rightEditorView.ScrollPosition()
}
```

Call `syncRightTab()` in the same places `syncTabFromView()` is called (before tab switch, etc).

- [ ] **Step 5: Build and run all tests**

Run: `cd /home/seoji/local/ted && go build ./cmd/ted/ && go test ./... -v`
Expected: BUILD OK, ALL PASS

- [ ] **Step 6: Commit**

```bash
git add internal/editor/editor.go
git commit -m "feat: wire SplitManager into Editor for split pane rendering and events"
```

---

## Chunk 4: Commands and Keybindings

### Task 4: Add split commands and keybindings

**Files:**
- Modify: `internal/editor/editor.go` (ExecuteCommand switch)
- Modify: `internal/editor/commands_builtin.go`
- Modify: `configs/keybindings.json`

- [ ] **Step 1: Register split commands in commands_builtin.go**

Add:
```go
	reg.Register(&Command{
		Name:        "split.vertical",
		Description: "Split editor vertically",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("split.vertical")
		},
	})
	reg.Register(&Command{
		Name:        "split.close",
		Description: "Close current split pane",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("split.close")
		},
	})
	reg.Register(&Command{
		Name:        "split.focus",
		Description: "Switch focus to other pane",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("split.focus")
		},
	})
```

- [ ] **Step 2: Handle split commands in ExecuteCommand**

In the ExecuteCommand switch in editor.go, add:
```go
	case "split.vertical":
		if !e.splitManager.IsSplit() {
			tab := e.tabs.Active()
			if tab != nil {
				e.syncTabFromView()
				e.splitManager.SplitWithCurrentBuffer(tab.Buffer, tab.Language)
				e.layout.SetSplitMode(true)
				e.syncRightView()
			}
		}
	case "split.close":
		if e.splitManager.IsSplit() {
			e.syncRightTab()
			e.splitManager.CloseSplit()
			e.layout.SetSplitMode(false)
			e.rightEditorView = nil
		}
	case "split.focus":
		if e.splitManager.IsSplit() {
			// Save current pane state
			if e.splitManager.ActivePane() == PaneLeft {
				e.syncTabFromView()
			} else {
				e.syncRightTab()
			}
			e.splitManager.FocusOther()
		}
```

- [ ] **Step 3: Add keybindings**

In `configs/keybindings.json`, add:
```json
	{ "key": "ctrl+\\", "command": "split.vertical" },
	{ "key": "ctrl+w", "command": "split.focus" },
	{ "key": "ctrl+shift+w", "command": "split.close" }
```

Or add in LoadKeybindings() if keybindings are loaded programmatically.

- [ ] **Step 4: Build and test manually**

Run: `cd /home/seoji/local/ted && go build -o ted ./cmd/ted/ && go test ./...`
Then run `./ted somefile.go` and test:
- `Ctrl+\` to split
- `Ctrl+W` to switch focus
- Edit in each pane independently
- `Ctrl+Shift+W` to close split

- [ ] **Step 5: Commit**

```bash
git add internal/editor/editor.go internal/editor/commands_builtin.go configs/keybindings.json
git commit -m "feat: add split.vertical, split.close, split.focus commands with keybindings"
```

### Task 5: Visual indicator for active pane

**Files:**
- Modify: `internal/editor/editor.go` (render method)

- [ ] **Step 1: Highlight active pane separator or border**

When rendering split separator, use a different color based on which pane is active. For example, highlight the separator on the active side:

```go
	if r, ok := regions["editor.separator"]; ok {
		sepStyle := e.theme.UIStyle("panel").Foreground(tcell.ColorGray)
		for y := r.Y; y < r.Y+r.Height; y++ {
			e.screen.SetContent(r.X, y, '│', nil, sepStyle)
		}
	}
```

Alternatively, show a small indicator in the status bar showing which pane is active (e.g., "[L]" or "[R]").

- [ ] **Step 2: Update status bar to show pane indicator when split**

In the render section for statusbar, if split mode is active, append pane indicator to status text.

- [ ] **Step 3: Build and verify visually**

- [ ] **Step 4: Commit**

```bash
git add internal/editor/editor.go
git commit -m "feat: visual indicator for active split pane"
```
