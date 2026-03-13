package buffer

import "testing"

func TestUndoInsert(t *testing.T) {
	pt := NewPieceTable("hello")
	um := NewUndoManager(pt)
	um.Execute(pt.Insert(5, " world"))
	if got := pt.Text(); got != "hello world" {
		t.Fatalf("after insert: expected 'hello world', got %q", got)
	}
	um.Undo()
	if got := pt.Text(); got != "hello" {
		t.Errorf("after undo: expected 'hello', got %q", got)
	}
}

func TestRedoInsert(t *testing.T) {
	pt := NewPieceTable("hello")
	um := NewUndoManager(pt)
	um.Execute(pt.Insert(5, " world"))
	um.Undo()
	um.Redo()
	if got := pt.Text(); got != "hello world" {
		t.Errorf("after redo: expected 'hello world', got %q", got)
	}
}

func TestUndoDelete(t *testing.T) {
	pt := NewPieceTable("hello world")
	um := NewUndoManager(pt)
	um.Execute(pt.Delete(5, 6))
	if got := pt.Text(); got != "hello" {
		t.Fatalf("after delete: expected 'hello', got %q", got)
	}
	um.Undo()
	if got := pt.Text(); got != "hello world" {
		t.Errorf("after undo: expected 'hello world', got %q", got)
	}
}

func TestMultipleUndoRedo(t *testing.T) {
	pt := NewPieceTable("")
	um := NewUndoManager(pt)
	um.Execute(pt.Insert(0, "a"))
	um.Execute(pt.Insert(1, "b"))
	um.Execute(pt.Insert(2, "c"))
	if got := pt.Text(); got != "abc" {
		t.Fatalf("expected 'abc', got %q", got)
	}
	um.Undo() // remove "c"
	um.Undo() // remove "b"
	if got := pt.Text(); got != "a" {
		t.Errorf("after 2 undos: expected 'a', got %q", got)
	}
	um.Redo() // restore "b"
	if got := pt.Text(); got != "ab" {
		t.Errorf("after redo: expected 'ab', got %q", got)
	}
}

func TestRedoClearedOnNewEdit(t *testing.T) {
	pt := NewPieceTable("a")
	um := NewUndoManager(pt)
	um.Execute(pt.Insert(1, "b"))
	um.Undo()
	if got := pt.Text(); got != "a" {
		t.Fatalf("after undo: expected 'a', got %q", got)
	}
	// New edit should clear redo stack
	um.Execute(pt.Insert(1, "c"))
	if got := pt.Text(); got != "ac" {
		t.Fatalf("after new insert: expected 'ac', got %q", got)
	}
	um.Redo() // should be no-op
	if got := pt.Text(); got != "ac" {
		t.Errorf("redo after new edit: expected 'ac', got %q", got)
	}
}

func TestUndoOnEmptyStack(t *testing.T) {
	pt := NewPieceTable("hello")
	um := NewUndoManager(pt)
	um.Undo() // should be no-op
	if got := pt.Text(); got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestDirtyTracking(t *testing.T) {
	pt := NewPieceTable("hello")
	um := NewUndoManager(pt)
	if um.IsDirty() {
		t.Error("should not be dirty initially")
	}
	um.Execute(pt.Insert(5, "!"))
	if !um.IsDirty() {
		t.Error("should be dirty after edit")
	}
	um.MarkSaved()
	if um.IsDirty() {
		t.Error("should not be dirty after save")
	}
	um.Execute(pt.Insert(6, "!"))
	if !um.IsDirty() {
		t.Error("should be dirty after edit post-save")
	}
	um.Undo()
	if um.IsDirty() {
		t.Error("should not be dirty after undoing to save point")
	}
}

func TestCanUndoCanRedo(t *testing.T) {
	pt := NewPieceTable("x")
	um := NewUndoManager(pt)
	if um.CanUndo() {
		t.Error("should not be able to undo initially")
	}
	if um.CanRedo() {
		t.Error("should not be able to redo initially")
	}
	um.Execute(pt.Insert(1, "y"))
	if !um.CanUndo() {
		t.Error("should be able to undo after edit")
	}
	um.Undo()
	if !um.CanRedo() {
		t.Error("should be able to redo after undo")
	}
	if um.CanUndo() {
		t.Error("should not be able to undo with empty undo stack")
	}
}
