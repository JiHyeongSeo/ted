# Large File Handling Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Support opening and editing GB-scale files with low memory footprint using mmap-backed piece table

**Architecture:** Introduce mmap-backed original buffer for large files (>10MB threshold). Change piece table `original` from `string` to a `ContentSource` interface that abstracts over string (small files) and mmap'd []byte (large files). Eliminate `buf.Text()` calls in rendering paths. Build line index incrementally for large files.

**Tech Stack:** Go, golang.org/x/exp/mmap or syscall mmap, existing piece table

---

## Current Problems

1. `os.ReadFile()` loads entire file into heap memory
2. `PieceTable.original` is a `string` — entire file duplicated on heap
3. `pieceBytes()` converts string slice to `[]byte` — allocates on every call
4. `buf.Text()` materializes entire content — called by syntax highlighting
5. `rebuildLineIndex()` calls `pt.Text()` — materializes entire content

## Strategy: mmap + Interface Abstraction

- **Small files (<10MB)**: Keep current behavior (fast, simple)
- **Large files (>=10MB)**: mmap the file, piece table references mmap'd memory
- **ContentSource interface**: Abstracts byte access — `string` for small, `mmap` for large
- **No buf.Text() in hot paths**: Syntax highlight line-by-line only

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/buffer/content.go` | Create | ContentSource interface + string/mmap implementations |
| `internal/buffer/piece.go` | Modify | Use ContentSource instead of `string` for original |
| `internal/buffer/buffer.go` | Modify | Large file detection, mmap open path, fix rebuildLineIndex |
| `internal/view/editorview.go` | Modify | Remove buf.Text() calls, line-by-line highlighting |
| `go.mod` | Modify | Add mmap dependency if needed |

---

## Chunk 1: ContentSource Interface

### Task 1: Create ContentSource abstraction

**Files:**
- Create: `internal/buffer/content.go`
- Test: `internal/buffer/content_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/buffer/content_test.go
package buffer

import "testing"

func TestStringContent(t *testing.T) {
	cs := NewStringContent("hello world")
	if cs.Len() != 11 {
		t.Errorf("expected length 11, got %d", cs.Len())
	}
	got := cs.Slice(0, 5)
	if string(got) != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
	got = cs.Slice(6, 11)
	if string(got) != "world" {
		t.Errorf("expected 'world', got %q", got)
	}
}

func TestStringContentByteAt(t *testing.T) {
	cs := NewStringContent("abc")
	if cs.ByteAt(0) != 'a' {
		t.Errorf("expected 'a', got %c", cs.ByteAt(0))
	}
	if cs.ByteAt(2) != 'c' {
		t.Errorf("expected 'c', got %c", cs.ByteAt(2))
	}
}
```

- [ ] **Step 2: Implement ContentSource interface and StringContent**

```go
// internal/buffer/content.go
package buffer

// ContentSource provides byte-level access to original file content.
// Implementations: StringContent (heap), MmapContent (memory-mapped file).
type ContentSource interface {
	Slice(start, end int) []byte
	ByteAt(offset int) byte
	Len() int
	Close() error
}

// StringContent wraps a string for small files.
type StringContent struct {
	data string
}

func NewStringContent(s string) *StringContent {
	return &StringContent{data: s}
}

func (sc *StringContent) Slice(start, end int) []byte {
	return []byte(sc.data[start:end])
}

func (sc *StringContent) ByteAt(offset int) byte {
	return sc.data[offset]
}

func (sc *StringContent) Len() int {
	return len(sc.data)
}

func (sc *StringContent) Close() error {
	return nil
}
```

- [ ] **Step 3: Run tests, commit**

```bash
go test ./internal/buffer/ -run TestStringContent -v
git commit -m "feat: add ContentSource interface with StringContent implementation"
```

### Task 2: Add MmapContent implementation

**Files:**
- Modify: `internal/buffer/content.go`
- Test: `internal/buffer/content_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestMmapContent(t *testing.T) {
	// Create a temp file
	f, err := os.CreateTemp("", "ted-mmap-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	content := "hello mmap world\nline two\n"
	f.WriteString(content)
	f.Close()

	mc, err := NewMmapContent(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer mc.Close()

	if mc.Len() != len(content) {
		t.Errorf("expected length %d, got %d", len(content), mc.Len())
	}
	got := mc.Slice(0, 5)
	if string(got) != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
	if mc.ByteAt(6) != 'm' {
		t.Errorf("expected 'm', got %c", mc.ByteAt(6))
	}
}
```

- [ ] **Step 2: Implement MmapContent using syscall**

```go
// MmapContent provides memory-mapped file access for large files.
type MmapContent struct {
	data []byte
	size int
}

func NewMmapContent(path string) (*MmapContent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := int(info.Size())
	if size == 0 {
		return &MmapContent{data: nil, size: 0}, nil
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, size,
		syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		return nil, fmt.Errorf("mmap failed: %w", err)
	}

	return &MmapContent{data: data, size: size}, nil
}

func (mc *MmapContent) Slice(start, end int) []byte {
	if mc.data == nil {
		return nil
	}
	// Return a copy to avoid holding reference to mmap'd page
	out := make([]byte, end-start)
	copy(out, mc.data[start:end])
	return out
}

func (mc *MmapContent) ByteAt(offset int) byte {
	return mc.data[offset]
}

func (mc *MmapContent) Len() int {
	return mc.size
}

func (mc *MmapContent) Close() error {
	if mc.data != nil {
		return syscall.Munmap(mc.data)
	}
	return nil
}
```

- [ ] **Step 3: Run tests, commit**

```bash
go test ./internal/buffer/ -run TestMmapContent -v
git commit -m "feat: add MmapContent for memory-mapped large file access"
```

---

## Chunk 2: Piece Table Migration

### Task 3: Migrate PieceTable from string to ContentSource

**Files:**
- Modify: `internal/buffer/piece.go`
- Modify: `internal/buffer/piece_test.go`

- [ ] **Step 1: Change PieceTable.original from string to ContentSource**

```go
type PieceTable struct {
	original ContentSource // was: string
	add      []byte
	pieces   []piece
}
```

- [ ] **Step 2: Update NewPieceTable to accept ContentSource**

```go
func NewPieceTable(content ContentSource) *PieceTable {
	pt := &PieceTable{
		original: content,
		add:      make([]byte, 0, 1024),
	}
	if content.Len() > 0 {
		pt.pieces = []piece{{source: srcOriginal, start: 0, length: content.Len()}}
	}
	return pt
}
```

Add convenience constructor for backward compatibility:
```go
func NewPieceTableFromString(s string) *PieceTable {
	return NewPieceTable(NewStringContent(s))
}
```

- [ ] **Step 3: Update pieceBytes to use ContentSource**

```go
func (pt *PieceTable) pieceBytes(p piece) []byte {
	if p.source == srcOriginal {
		return pt.original.Slice(p.start, p.start+p.length)
	}
	return pt.add[p.start : p.start+p.length]
}
```

- [ ] **Step 4: Add Close method to PieceTable**

```go
func (pt *PieceTable) Close() error {
	if pt.original != nil {
		return pt.original.Close()
	}
	return nil
}
```

- [ ] **Step 5: Update all callers in buffer.go**

In `NewBuffer`:
```go
func NewBuffer(content string) *Buffer {
	pt := NewPieceTableFromString(content)
	// ...
}
```

- [ ] **Step 6: Fix piece_test.go — update NewPieceTable calls**

All tests that call `NewPieceTable("some string")` should use `NewPieceTableFromString("some string")` instead.

- [ ] **Step 7: Run ALL tests**

```bash
go test ./internal/buffer/ -v
go test ./...
```

- [ ] **Step 8: Commit**

```bash
git commit -m "refactor: migrate PieceTable.original from string to ContentSource interface"
```

---

## Chunk 3: Large File Open Path

### Task 4: Add mmap-based OpenFile for large files

**Files:**
- Modify: `internal/buffer/buffer.go`

- [ ] **Step 1: Update OpenFile with size threshold**

```go
const LargeFileThreshold = 10 * 1024 * 1024 // 10MB

func OpenFile(path string) (*Buffer, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if info.Size() >= LargeFileThreshold {
		return openLargeFile(path, info.Size())
	}

	// Small file: existing behavior
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	b := NewBuffer(string(data))
	b.path = path
	if !utf8.Valid(data) {
		b.ReadOnly = true
	}
	b.undo.MarkSaved()
	return b, nil
}

func openLargeFile(path string, size int64) (*Buffer, error) {
	mc, err := NewMmapContent(path)
	if err != nil {
		return nil, err
	}

	pt := NewPieceTable(mc)
	b := &Buffer{
		pt:   pt,
		undo: NewUndoManager(pt),
		path: path,
	}
	b.rebuildLineIndex()
	b.undo.MarkSaved()
	return b, nil
}
```

- [ ] **Step 2: Fix rebuildLineIndex for large files — avoid pt.Text()**

Replace the current rebuildLineIndex which calls `pt.Text()` (materializes entire content):

```go
func (b *Buffer) rebuildLineIndex() {
	length := b.pt.Length()
	b.lineOffsets = []int{0}

	// Scan through pieces to find newlines without materializing full text
	pos := 0
	for _, p := range b.pt.pieces {
		data := b.pt.pieceBytes(p)
		for i, ch := range data {
			if ch == '\n' {
				b.lineOffsets = append(b.lineOffsets, pos+i+1)
			}
		}
		pos += p.length
	}
	_ = length // sanity check available if needed
}
```

This scans piece-by-piece instead of materializing the entire text. For mmap'd files, only accessed pages are loaded.

- [ ] **Step 3: Add Close method to Buffer**

```go
func (b *Buffer) Close() error {
	return b.pt.Close()
}
```

- [ ] **Step 4: Update editor to call Close on buffer when closing tabs**

In `internal/editor/editor.go`, find where tabs are closed and call `buf.Close()` to release mmap.

- [ ] **Step 5: Write test for large file open**

```go
func TestOpenLargeFile(t *testing.T) {
	// Create a temp file > 10MB
	f, err := os.CreateTemp("", "ted-large-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	line := "This is a test line for large file handling in ted editor.\n"
	// Write ~11MB
	for i := 0; i < 200000; i++ {
		f.WriteString(line)
	}
	f.Close()

	buf, err := OpenFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer buf.Close()

	if buf.LineCount() != 200001 { // 200000 newlines + last line
		t.Errorf("expected 200001 lines, got %d", buf.LineCount())
	}

	// Verify first and last lines
	first := buf.Line(0)
	if first != "This is a test line for large file handling in ted editor." {
		t.Errorf("unexpected first line: %q", first)
	}
	last := buf.Line(199999)
	if last != "This is a test line for large file handling in ted editor." {
		t.Errorf("unexpected last line: %q", last)
	}
}
```

- [ ] **Step 6: Run all tests, commit**

```bash
go test ./internal/buffer/ -v -timeout 30s
go test ./...
git commit -m "feat: mmap-based file loading for large files (>10MB)"
```

---

## Chunk 4: Eliminate buf.Text() from Rendering

### Task 5: Remove buf.Text() calls from EditorView

**Files:**
- Modify: `internal/view/editorview.go`

- [ ] **Step 1: Find all buf.Text() calls in editorview.go**

Search for `buf.Text()` or `e.buf.Text()`. These are typically used for:
- Syntax highlighting (tree-sitter needs full text)
- Search highlighting

- [ ] **Step 2: Replace with line-range based approach**

For syntax highlighting, instead of passing entire text, only highlight visible lines. If tree-sitter needs full text, build it from visible range only or disable tree-sitter for large files:

```go
// For large files, only highlight visible lines
func (e *EditorView) getHighlightText() string {
	if e.buf.IsLarge() {
		// Only get text for visible range + some context
		startLine := e.scrollY
		endLine := e.scrollY + e.Bounds().Height + 10
		if endLine > e.buf.LineCount() {
			endLine = e.buf.LineCount()
		}
		var sb strings.Builder
		for i := startLine; i < endLine; i++ {
			sb.WriteString(e.buf.Line(i))
			sb.WriteByte('\n')
		}
		return sb.String()
	}
	return e.buf.Text()
}
```

- [ ] **Step 3: Add IsLarge() method to Buffer**

```go
func (b *Buffer) IsLarge() bool {
	return b.pt.Length() >= LargeFileThreshold
}
```

- [ ] **Step 4: Run all tests, verify build**

```bash
go build ./cmd/ted/
go test ./...
```

- [ ] **Step 5: Commit**

```bash
git commit -m "perf: avoid buf.Text() in rendering for large files"
```

### Task 6: Benchmark and verify

**Files:**
- Create: `internal/buffer/large_test.go`

- [ ] **Step 1: Write benchmark test**

```go
func BenchmarkOpenLargeFile(b *testing.B) {
	// Create a ~50MB temp file
	f, _ := os.CreateTemp("", "ted-bench-*")
	line := "Benchmark line content for testing large file performance.\n"
	for i := 0; i < 900000; i++ {
		f.WriteString(line)
	}
	f.Close()
	defer os.Remove(f.Name())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf, _ := OpenFile(f.Name())
		buf.Close()
	}
}

func BenchmarkLineAccess(b *testing.B) {
	f, _ := os.CreateTemp("", "ted-bench-*")
	line := "Benchmark line content for testing.\n"
	for i := 0; i < 900000; i++ {
		f.WriteString(line)
	}
	f.Close()
	defer os.Remove(f.Name())

	buf, _ := OpenFile(f.Name())
	defer buf.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buf.Line(i % buf.LineCount())
	}
}
```

- [ ] **Step 2: Run benchmarks**

```bash
go test ./internal/buffer/ -bench=. -benchmem -timeout 60s
```

- [ ] **Step 3: Commit**

```bash
git commit -m "test: add benchmarks for large file open and line access"
```
