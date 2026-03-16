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
