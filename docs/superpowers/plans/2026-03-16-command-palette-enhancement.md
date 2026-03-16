# Command Palette Enhancement Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enhance the existing command palette with match highlighting, buffer switching, keybinding display, description fallback matching, and improved rendering

**Architecture:** Enhance `internal/view/palette.go` with fuzzy match position tracking for highlighting, add a buffer switching mode (`#` prefix), show keybinding shortcuts next to commands, and support description-based fallback matching. Reference fresh's provider pattern for category handling.

**Tech Stack:** Go, tcell/v2, sahilm/fuzzy (already in use), go-runewidth

---

## Current State

- `internal/view/palette.go`: 3 modes (File/Command/GoLine), fuzzy matching via `sahilm/fuzzy`, basic rendering
- `internal/editor/command.go`: CommandRegistry with Name/Description/Execute
- `internal/editor/commands_builtin.go`: 22 registered commands
- `configs/keybindings.json`: keybinding config

## Improvements (from fresh reference)

1. **Match position highlighting** — highlight matched characters in fuzzy results
2. **Buffer switch mode** (`#` prefix) — switch between open buffers/tabs
3. **Keybinding display** — show shortcuts next to command names
4. **Description fallback matching** — if label doesn't match, try matching description
5. **Border/shadow rendering** — better visual separation from background

---

## Chunk 1: Match Position Highlighting

### Task 1: Track fuzzy match positions in filter results

**Files:**
- Modify: `internal/view/palette.go:23-29` (PaletteItem struct)
- Modify: `internal/view/palette.go:338-353` (fuzzyFilter method)
- Test: `internal/view/palette_test.go` (create)

- [ ] **Step 1: Write the failing test for match position tracking**

```go
// internal/view/palette_test.go
package view

import (
	"testing"
)

func TestFuzzyFilterTracksMatchPositions(t *testing.T) {
	p := NewCommandPalette(nil)
	items := []PaletteItem{
		{Label: "file.save", Description: "Save the current file"},
		{Label: "file.open", Description: "Open a file"},
		{Label: "search.find", Description: "Find text"},
	}
	p.SetItems(items)

	p.query = ">fs"
	p.mode = PaletteModeCommand
	p.filterItems()

	if len(p.filtered) == 0 {
		t.Fatal("expected at least one match for 'fs'")
	}
	if len(p.filtered[0].MatchPositions) == 0 {
		t.Error("expected match positions to be populated")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/seoji/local/ted && go test ./internal/view/ -run TestFuzzyFilterTracksMatchPositions -v`
Expected: FAIL — `MatchPositions` field does not exist

- [ ] **Step 3: Add MatchPositions field to PaletteItem**

```go
// internal/view/palette.go — update PaletteItem struct
type PaletteItem struct {
	Label          string
	Description    string
	Command        string
	FilePath       string
	MatchPositions []int // indices of matched characters in Label
}
```

- [ ] **Step 4: Update fuzzyFilter to populate MatchPositions**

```go
// internal/view/palette.go — replace fuzzyFilter method
func (p *CommandPalette) fuzzyFilter(items []PaletteItem, query string) {
	if query == "" {
		p.filtered = make([]PaletteItem, len(items))
		copy(p.filtered, items)
		return
	}
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = item.Label
	}
	matches := fuzzy.Find(query, labels)
	p.filtered = make([]PaletteItem, len(matches))
	for i, m := range matches {
		item := items[m.Index]
		item.MatchPositions = m.MatchedIndexes
		p.filtered[i] = item
	}
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd /home/seoji/local/ted && go test ./internal/view/ -run TestFuzzyFilterTracksMatchPositions -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/view/palette.go internal/view/palette_test.go
git commit -m "feat(palette): track fuzzy match positions for highlighting"
```

### Task 2: Render highlighted match positions

**Files:**
- Modify: `internal/view/palette.go:196-232` (Render method, item drawing section)

- [ ] **Step 1: Write test for highlight rendering logic**

```go
// internal/view/palette_test.go — add test
func TestMatchPositionSet(t *testing.T) {
	positions := []int{0, 2, 5}
	set := makePositionSet(positions)

	if !set[0] {
		t.Error("expected position 0 to be in set")
	}
	if set[1] {
		t.Error("expected position 1 to NOT be in set")
	}
	if !set[2] {
		t.Error("expected position 2 to be in set")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/seoji/local/ted && go test ./internal/view/ -run TestMatchPositionSet -v`
Expected: FAIL — `makePositionSet` undefined

- [ ] **Step 3: Add makePositionSet helper and update Render to highlight matches**

```go
// internal/view/palette.go — add helper
func makePositionSet(positions []int) map[int]bool {
	set := make(map[int]bool, len(positions))
	for _, p := range positions {
		set[p] = true
	}
	return set
}
```

Then update the item rendering loop in `Render()` — replace the label drawing section (lines ~217-230) with:

```go
			label := p.filtered[itemIdx].Label
			desc := p.filtered[itemIdx].Description
			matchSet := makePositionSet(p.filtered[itemIdx].MatchPositions)
			highlightStyle := style.Bold(true).Foreground(tcell.ColorYellow)

			// Draw "  " prefix
			x := startX + 1
			screen.SetContent(x, y, ' ', nil, style)
			x++
			screen.SetContent(x, y, ' ', nil, style)
			x++

			// Draw label with highlights
			for ci, ch := range label {
				w := runewidth.RuneWidth(ch)
				if x+w >= startX+paletteWidth-1 {
					break
				}
				s := style
				if matchSet[ci] {
					s = highlightStyle
				}
				screen.SetContent(x, y, ch, nil, s)
				x += w
			}

			// Draw description
			if desc != "" {
				descStyle := style.Foreground(tcell.ColorDarkGray)
				x += 2 // gap
				for _, ch := range desc {
					w := runewidth.RuneWidth(ch)
					if x+w >= startX+paletteWidth-1 {
						break
					}
					screen.SetContent(x, y, ch, nil, descStyle)
					x += w
				}
			}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/seoji/local/ted && go test ./internal/view/ -run TestMatchPositionSet -v`
Expected: PASS

- [ ] **Step 5: Manual verification**

Run: `cd /home/seoji/local/ted && go build -o ted ./cmd/ted/ && ./ted`
Open palette with Ctrl+P, type a query, verify matched characters are highlighted in yellow/bold.

- [ ] **Step 6: Commit**

```bash
git add internal/view/palette.go internal/view/palette_test.go
git commit -m "feat(palette): highlight matched characters in fuzzy results"
```

---

## Chunk 2: Buffer Switch Mode

### Task 3: Add buffer switch mode with `#` prefix

**Files:**
- Modify: `internal/view/palette.go:14-21` (PaletteMode constants)
- Modify: `internal/view/palette.go:31-46` (CommandPalette struct)
- Modify: `internal/view/palette.go:314-322` (detectMode)
- Modify: `internal/view/palette.go:324-336` (filterItems)
- Modify: `internal/view/palette.go:283-312` (handleSelect)
- Test: `internal/view/palette_test.go`

- [ ] **Step 1: Write failing test for buffer mode detection**

```go
// internal/view/palette_test.go
func TestDetectModeBuffer(t *testing.T) {
	p := NewCommandPalette(nil)

	p.query = "#"
	p.detectMode()
	if p.mode != PaletteModeBuffer {
		t.Errorf("expected PaletteModeBuffer, got %d", p.mode)
	}

	p.query = "#main"
	p.detectMode()
	if p.mode != PaletteModeBuffer {
		t.Errorf("expected PaletteModeBuffer for '#main', got %d", p.mode)
	}

	p.query = "test"
	p.detectMode()
	if p.mode != PaletteModeFile {
		t.Errorf("expected PaletteModeFile for 'test', got %d", p.mode)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/seoji/local/ted && go test ./internal/view/ -run TestDetectModeBuffer -v`
Expected: FAIL — `PaletteModeBuffer` undefined

- [ ] **Step 3: Implement buffer mode**

Add `PaletteModeBuffer` constant:
```go
const (
	PaletteModeFile    PaletteMode = iota
	PaletteModeCommand
	PaletteModeGoLine
	PaletteModeBuffer // "#" prefix: switch open buffer
)
```

Add buffer items and callback to struct:
```go
type CommandPalette struct {
	// ... existing fields ...
	bufferItems  []PaletteItem
	onBufferOpen func(path string)
}
```

Add setter:
```go
func (p *CommandPalette) SetBufferItems(items []PaletteItem) {
	p.bufferItems = items
}

func (p *CommandPalette) SetOnBufferOpen(fn func(path string)) {
	p.onBufferOpen = fn
}
```

Update `detectMode`:
```go
func (p *CommandPalette) detectMode() {
	if strings.HasPrefix(p.query, ">") {
		p.mode = PaletteModeCommand
	} else if strings.HasPrefix(p.query, ":") {
		p.mode = PaletteModeGoLine
	} else if strings.HasPrefix(p.query, "#") {
		p.mode = PaletteModeBuffer
	} else {
		p.mode = PaletteModeFile
	}
}
```

Update `filterItems`:
```go
func (p *CommandPalette) filterItems() {
	switch p.mode {
	case PaletteModeCommand:
		searchQuery := strings.TrimPrefix(p.query, ">")
		searchQuery = strings.TrimSpace(searchQuery)
		p.fuzzyFilter(p.commandItems, searchQuery)
	case PaletteModeBuffer:
		searchQuery := strings.TrimPrefix(p.query, "#")
		searchQuery = strings.TrimSpace(searchQuery)
		p.fuzzyFilter(p.bufferItems, searchQuery)
	case PaletteModeGoLine:
		p.filtered = nil
	default:
		p.fuzzyFilter(p.fileItems, p.query)
	}
	p.selectedIdx = 0
}
```

Update `handleSelect` — add case before default:
```go
	case PaletteModeBuffer:
		if p.selectedIdx >= 0 && p.selectedIdx < len(p.filtered) {
			item := p.filtered[p.selectedIdx]
			p.Hide()
			if item.FilePath != "" && p.onBufferOpen != nil {
				p.onBufferOpen(item.FilePath)
			}
		}
```

Update `Render` prompt section:
```go
	case PaletteModeBuffer:
		prompt = "# " + strings.TrimPrefix(p.query, "#")
```

Update hint text:
```go
	hint := "Search files... (> commands, : line, # buffers)"
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/seoji/local/ted && go test ./internal/view/ -v`
Expected: ALL PASS

- [ ] **Step 5: Wire buffer items in editor**

In `internal/editor/editor.go`, where palette is shown, populate buffer items from open tabs. Find where `palette.Show()` is called and add buffer item population before it. The exact wiring depends on how tabs are tracked in the editor — look for tab/buffer list and create PaletteItems from them.

- [ ] **Step 6: Commit**

```bash
git add internal/view/palette.go internal/view/palette_test.go internal/editor/editor.go
git commit -m "feat(palette): add buffer switch mode with # prefix"
```

---

## Chunk 3: Keybinding Display & Description Fallback

### Task 4: Show keybindings next to commands

**Files:**
- Modify: `internal/view/palette.go:23-29` (PaletteItem struct)
- Modify: `internal/view/palette.go` (Render method)
- Modify: `internal/editor/editor.go` (where palette command items are populated)
- Test: `internal/view/palette_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestPaletteItemKeybinding(t *testing.T) {
	item := PaletteItem{
		Label:      "file.save",
		Description: "Save the current file",
		Keybinding: "Ctrl+S",
	}
	if item.Keybinding != "Ctrl+S" {
		t.Errorf("expected Ctrl+S, got %s", item.Keybinding)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/seoji/local/ted && go test ./internal/view/ -run TestPaletteItemKeybinding -v`
Expected: FAIL — `Keybinding` field undefined

- [ ] **Step 3: Add Keybinding field to PaletteItem**

```go
type PaletteItem struct {
	Label          string
	Description    string
	Command        string
	FilePath       string
	Keybinding     string // display shortcut (e.g. "Ctrl+S")
	MatchPositions []int
}
```

- [ ] **Step 4: Update Render to show keybinding right-aligned**

In the item rendering section of `Render()`, after drawing label and description, add keybinding rendering:

```go
			// Draw keybinding right-aligned
			if kb := p.filtered[itemIdx].Keybinding; kb != "" {
				kbStyle := style.Foreground(tcell.ColorDarkCyan)
				kbWidth := runewidth.StringWidth(kb)
				kbX := startX + paletteWidth - kbWidth - 2
				if kbX > x+1 { // only if there's room
					for _, ch := range kb {
						screen.SetContent(kbX, y, ch, nil, kbStyle)
						kbX += runewidth.RuneWidth(ch)
					}
				}
			}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd /home/seoji/local/ted && go test ./internal/view/ -run TestPaletteItemKeybinding -v`
Expected: PASS

- [ ] **Step 6: Populate keybindings when building command items**

In `internal/editor/editor.go`, where palette command items are built, look up keybindings from the keymap for each command and set the `Keybinding` field. This requires adding a reverse lookup method to the keymap or iterating bindings.

- [ ] **Step 7: Commit**

```bash
git add internal/view/palette.go internal/view/palette_test.go internal/editor/editor.go
git commit -m "feat(palette): display keybinding shortcuts next to commands"
```

### Task 5: Description fallback matching

**Files:**
- Modify: `internal/view/palette.go:338-353` (fuzzyFilter method)
- Test: `internal/view/palette_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestFuzzyFilterFallsBackToDescription(t *testing.T) {
	p := NewCommandPalette(nil)
	items := []PaletteItem{
		{Label: "file.save", Description: "Save the current file"},
		{Label: "file.open", Description: "Open a file"},
	}
	p.SetItems(items)

	// "save" doesn't appear in "file.save" label strongly, but matches description
	p.query = ">Save the"
	p.mode = PaletteModeCommand
	p.filterItems()

	found := false
	for _, item := range p.filtered {
		if item.Label == "file.save" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'file.save' to match via description fallback")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/seoji/local/ted && go test ./internal/view/ -run TestFuzzyFilterFallsBackToDescription -v`
Expected: FAIL (or might pass if fuzzy is lenient — adjust query if needed)

- [ ] **Step 3: Update fuzzyFilter for description fallback**

```go
func (p *CommandPalette) fuzzyFilter(items []PaletteItem, query string) {
	if query == "" {
		p.filtered = make([]PaletteItem, len(items))
		copy(p.filtered, items)
		return
	}

	// Primary: match on labels
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = item.Label
	}
	matches := fuzzy.Find(query, labels)

	matched := make(map[int]bool)
	p.filtered = make([]PaletteItem, 0, len(matches))
	for _, m := range matches {
		item := items[m.Index]
		item.MatchPositions = m.MatchedIndexes
		p.filtered = append(p.filtered, item)
		matched[m.Index] = true
	}

	// Fallback: match on descriptions for unmatched items
	var unmatched []string
	var unmatchedIdx []int
	for i, item := range items {
		if !matched[i] {
			unmatched = append(unmatched, item.Description)
			unmatchedIdx = append(unmatchedIdx, i)
		}
	}
	if len(unmatched) > 0 {
		descMatches := fuzzy.Find(query, unmatched)
		for _, m := range descMatches {
			origIdx := unmatchedIdx[m.Index]
			item := items[origIdx]
			// No label match positions for description matches
			p.filtered = append(p.filtered, item)
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/seoji/local/ted && go test ./internal/view/ -run TestFuzzyFilterFallsBackToDescription -v`
Expected: PASS

- [ ] **Step 5: Run all palette tests**

Run: `cd /home/seoji/local/ted && go test ./internal/view/ -v`
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add internal/view/palette.go internal/view/palette_test.go
git commit -m "feat(palette): description fallback matching for commands"
```

---

## Chunk 4: Visual Polish

### Task 6: Border and shadow rendering

**Files:**
- Modify: `internal/view/palette.go` (Render method)

- [ ] **Step 1: Add border drawing to Render**

Before drawing the input row and items, draw a border around the palette area and a shadow effect:

```go
	// Draw shadow (1px offset)
	shadowStyle := tcell.StyleDefault.Background(tcell.ColorBlack)
	for x := startX + 1; x <= startX+paletteWidth; x++ {
		screen.SetContent(x, startY+paletteHeight, ' ', nil, shadowStyle)
	}
	for y := startY + 1; y <= startY+paletteHeight; y++ {
		screen.SetContent(startX+paletteWidth, y, ' ', nil, shadowStyle)
	}

	// Draw border
	borderStyle := p.theme.UIStyle("panel").Foreground(tcell.ColorGray)
	// Top border
	screen.SetContent(startX, startY-1, '┌', nil, borderStyle)
	for x := startX + 1; x < startX+paletteWidth-1; x++ {
		screen.SetContent(x, startY-1, '─', nil, borderStyle)
	}
	screen.SetContent(startX+paletteWidth-1, startY-1, '┐', nil, borderStyle)
	// Side borders
	for y := startY; y < startY+paletteHeight; y++ {
		screen.SetContent(startX, y, '│', nil, borderStyle)
		screen.SetContent(startX+paletteWidth-1, y, '│', nil, borderStyle)
	}
	// Bottom border
	screen.SetContent(startX, startY+paletteHeight, '└', nil, borderStyle)
	for x := startX + 1; x < startX+paletteWidth-1; x++ {
		screen.SetContent(x, startY+paletteHeight, '─', nil, borderStyle)
	}
	screen.SetContent(startX+paletteWidth-1, startY+paletteHeight, '┘', nil, borderStyle)
```

Adjust startX/startY and paletteWidth insets accordingly to account for border.

- [ ] **Step 2: Build and manually verify**

Run: `cd /home/seoji/local/ted && go build -o ted ./cmd/ted/ && ./ted`
Open palette, verify border and shadow are visible.

- [ ] **Step 3: Commit**

```bash
git add internal/view/palette.go
git commit -m "feat(palette): add border and shadow rendering"
```

### Task 7: Update hint text per mode

**Files:**
- Modify: `internal/view/palette.go` (Render method, hint section)

- [ ] **Step 1: Update hint to be mode-aware**

Replace the static hint with mode-specific hints:

```go
	if p.query == "" || (p.mode == PaletteModeCommand && strings.TrimPrefix(p.query, ">") == "") ||
		(p.mode == PaletteModeBuffer && strings.TrimPrefix(p.query, "#") == "") {
		var hint string
		switch p.mode {
		case PaletteModeCommand:
			hint = "Type to search commands..."
		case PaletteModeBuffer:
			hint = "Type to search open buffers..."
		case PaletteModeGoLine:
			hint = "Type line number..."
		default:
			hint = "Search files... (> commands, : line, # buffers)"
		}
		// ... existing hint rendering code
	}
```

- [ ] **Step 2: Build and manually verify**

Run: `cd /home/seoji/local/ted && go build -o ted ./cmd/ted/ && ./ted`
Type `>` in palette — hint should change to "Type to search commands..."

- [ ] **Step 3: Commit**

```bash
git add internal/view/palette.go
git commit -m "feat(palette): mode-specific hint text"
```

- [ ] **Step 4: Run full test suite**

Run: `cd /home/seoji/local/ted && go test ./... -v`
Expected: ALL PASS
