package view

import (
	"testing"

	"github.com/JiHyeongSeo/ted/internal/git"
	"github.com/JiHyeongSeo/ted/internal/types"
)

func TestGraphViewSelection(t *testing.T) {
	commits := []git.Commit{
		{Hash: "aaa", ShortHash: "aaa", Message: "commit A"},
		{Hash: "bbb", ShortHash: "bbb", Message: "commit B"},
		{Hash: "ccc", ShortHash: "ccc", Message: "commit C"},
	}
	rows := git.LayoutGraph(commits)
	gv := NewGraphView(nil, rows)
	gv.SetBounds(types.Rect{X: 0, Y: 0, Width: 80, Height: 20})

	if gv.SelectedIndex() != 0 {
		t.Errorf("expected initial selection 0, got %d", gv.SelectedIndex())
	}

	gv.MoveDown()
	if gv.SelectedIndex() != 1 {
		t.Errorf("expected selection 1, got %d", gv.SelectedIndex())
	}

	gv.MoveDown()
	gv.MoveDown() // should clamp at 2
	if gv.SelectedIndex() != 2 {
		t.Errorf("expected selection 2, got %d", gv.SelectedIndex())
	}

	gv.MoveUp()
	if gv.SelectedIndex() != 1 {
		t.Errorf("expected selection 1, got %d", gv.SelectedIndex())
	}
}

func TestGraphViewScroll(t *testing.T) {
	commits := make([]git.Commit, 50)
	for i := range commits {
		commits[i] = git.Commit{Hash: "h", ShortHash: "h", Message: "msg"}
	}
	rows := git.LayoutGraph(commits)
	gv := NewGraphView(nil, rows)
	gv.SetBounds(types.Rect{X: 0, Y: 0, Width: 80, Height: 10})

	for i := 0; i < 15; i++ {
		gv.MoveDown()
	}
	if gv.SelectedIndex() != 15 {
		t.Errorf("expected selection 15, got %d", gv.SelectedIndex())
	}
	if gv.ScrollY() < 6 {
		t.Errorf("expected scrollY >= 6, got %d", gv.ScrollY())
	}
}

func TestGraphViewEmpty(t *testing.T) {
	gv := NewGraphView(nil, nil)
	gv.SetBounds(types.Rect{X: 0, Y: 0, Width: 80, Height: 20})
	if gv.SelectedCommit() != nil {
		t.Errorf("expected nil selected commit for empty graph")
	}
}
