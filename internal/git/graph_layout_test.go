package git

import "testing"

func TestLayoutLinear(t *testing.T) {
	commits := []Commit{
		{Hash: "aaa", ShortHash: "aaa", Parents: []string{"bbb"}, Message: "commit A"},
		{Hash: "bbb", ShortHash: "bbb", Parents: []string{"ccc"}, Message: "commit B"},
		{Hash: "ccc", ShortHash: "ccc", Parents: nil, Message: "commit C"},
	}
	rows := LayoutGraph(commits)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	for i, r := range rows {
		if r.Column != 0 {
			t.Errorf("row %d: expected column 0, got %d", i, r.Column)
		}
		if r.Cells[0] != CellCommit {
			t.Errorf("row %d: expected CellCommit at col 0, got %d", i, r.Cells[0])
		}
	}
}

func TestLayoutBranch(t *testing.T) {
	commits := []Commit{
		{Hash: "aaa", ShortHash: "aaa", Parents: []string{"bbb", "ccc"}, Message: "merge"},
		{Hash: "bbb", ShortHash: "bbb", Parents: []string{"ddd"}, Message: "commit B"},
		{Hash: "ccc", ShortHash: "ccc", Parents: []string{"ddd"}, Message: "commit C"},
		{Hash: "ddd", ShortHash: "ddd", Parents: nil, Message: "root"},
	}
	rows := LayoutGraph(commits)
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}
	if rows[0].Column != 0 {
		t.Errorf("merge commit: expected column 0, got %d", rows[0].Column)
	}
	if rows[2].Column != 1 {
		t.Errorf("commit C: expected column 1, got %d", rows[2].Column)
	}
}

func TestLayoutEmpty(t *testing.T) {
	rows := LayoutGraph(nil)
	if len(rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(rows))
	}
}

// TestLayoutDiverge tests the bug case: two branches diverge from a common ancestor
// (no merge commit). Previously the second branch's lane became a "ghost" pipe forever.
//
// Topology (oldest at bottom):
//   E (feature tip)
//   D
//   C (main tip)
//   B  ← both C and D have B as parent
//   A  ← root
//
// git log order (newest first): E, D, C, B, A
func TestLayoutDiverge(t *testing.T) {
	commits := []Commit{
		{Hash: "E", Parents: []string{"D"}, Message: "feature tip"},
		{Hash: "D", Parents: []string{"B"}, Message: "feature"},
		{Hash: "C", Parents: []string{"B"}, Message: "main tip"},
		{Hash: "B", Parents: []string{"A"}, Message: "branch point"},
		{Hash: "A", Parents: nil, Message: "root"},
	}
	rows := LayoutGraph(commits)
	if len(rows) != 5 {
		t.Fatalf("expected 5 rows, got %d", len(rows))
	}

	// B is the branch point: must show convergence (CellMergeRight at col 1)
	rowB := rows[3]
	if rowB.Column != 0 {
		t.Errorf("B: expected col 0, got %d", rowB.Column)
	}
	if len(rowB.Cells) < 2 {
		t.Fatalf("B: expected at least 2 cells, got %d", len(rowB.Cells))
	}
	if rowB.Cells[0] != CellCommit {
		t.Errorf("B: expected CellCommit at col 0, got %d", rowB.Cells[0])
	}
	if rowB.Cells[1] != CellMergeRight {
		t.Errorf("B: expected CellMergeRight at col 1 (convergence), got %d", rowB.Cells[1])
	}

	// A has no parents and should be back to a single lane
	rowA := rows[4]
	if rowA.Column != 0 {
		t.Errorf("A: expected col 0, got %d", rowA.Column)
	}
	if len(rowA.Cells) != 1 || rowA.Cells[0] != CellCommit {
		t.Errorf("A: expected single CellCommit, got cells=%v", rowA.Cells)
	}
}
