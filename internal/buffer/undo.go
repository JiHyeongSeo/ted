package buffer

// UndoManager tracks edit history for undo/redo support.
type UndoManager struct {
	pt        *PieceTable
	undoStack []Edit
	redoStack []Edit
	saveIdx   int // index into undoStack at last save
}

// NewUndoManager creates an UndoManager for the given PieceTable.
func NewUndoManager(pt *PieceTable) *UndoManager {
	return &UndoManager{
		pt:      pt,
		saveIdx: 0,
	}
}

// Execute records an edit that was already applied to the PieceTable.
func (um *UndoManager) Execute(edit Edit) {
	um.undoStack = append(um.undoStack, edit)
	um.redoStack = um.redoStack[:0] // clear redo stack
}

// Undo reverses the last edit. No-op if undo stack is empty.
func (um *UndoManager) Undo() {
	if len(um.undoStack) == 0 {
		return
	}
	edit := um.undoStack[len(um.undoStack)-1]
	um.undoStack = um.undoStack[:len(um.undoStack)-1]

	reverse := um.applyReverse(edit)
	um.redoStack = append(um.redoStack, reverse)
}

// Redo re-applies the last undone edit. No-op if redo stack is empty.
func (um *UndoManager) Redo() {
	if len(um.redoStack) == 0 {
		return
	}
	edit := um.redoStack[len(um.redoStack)-1]
	um.redoStack = um.redoStack[:len(um.redoStack)-1]

	reverse := um.applyReverse(edit)
	um.undoStack = append(um.undoStack, reverse)
}

// applyReverse applies the reverse of an edit and returns the edit needed to undo it.
func (um *UndoManager) applyReverse(edit Edit) Edit {
	switch edit.Type {
	case EditInsert:
		// Reverse of insert is delete
		return um.pt.Delete(edit.Offset, len(edit.Text))
	case EditDelete:
		// Reverse of delete is insert
		return um.pt.Insert(edit.Offset, edit.Text)
	}
	return Edit{}
}

// IsDirty returns true if the buffer has been modified since the last save.
func (um *UndoManager) IsDirty() bool {
	return len(um.undoStack) != um.saveIdx
}

// MarkSaved records the current state as the saved state.
func (um *UndoManager) MarkSaved() {
	um.saveIdx = len(um.undoStack)
}

// CanUndo returns whether there are edits to undo.
func (um *UndoManager) CanUndo() bool {
	return len(um.undoStack) > 0
}

// CanRedo returns whether there are edits to redo.
func (um *UndoManager) CanRedo() bool {
	return len(um.redoStack) > 0
}
