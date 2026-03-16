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
	rightPane  *PaneState
}

func NewSplitManager() *SplitManager {
	return &SplitManager{
		activePane: PaneMain,
	}
}

func (sm *SplitManager) IsSplit() bool     { return sm.split }
func (sm *SplitManager) ActivePane() Pane  { return sm.activePane }
func (sm *SplitManager) SetActivePane(p Pane) { sm.activePane = p }
func (sm *SplitManager) RightPane() *PaneState { return sm.rightPane }

func (sm *SplitManager) Split(buf *buffer.Buffer, language string) {
	sm.split = true
	sm.activePane = PaneLeft
	sm.rightPane = &PaneState{
		Buffer:   buf,
		Language: language,
	}
}

func (sm *SplitManager) CloseSplit() {
	sm.split = false
	sm.activePane = PaneMain
	sm.rightPane = nil
}

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
