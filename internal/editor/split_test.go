package editor

import (
	"testing"

	"github.com/JiHyeongSeo/ted/internal/buffer"
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
	buf := buffer.NewBuffer("")
	sm.Split(buf, "go")
	if !sm.IsSplit() {
		t.Error("should be split after Split()")
	}
	if sm.RightPane() == nil {
		t.Error("right pane should be set")
	}
}

func TestSplitManagerClose(t *testing.T) {
	sm := NewSplitManager()
	buf := buffer.NewBuffer("")
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
	buf := buffer.NewBuffer("")
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
