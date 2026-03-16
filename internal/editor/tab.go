package editor

import (
	"github.com/JiHyeongSeo/ted/internal/buffer"
	"github.com/JiHyeongSeo/ted/internal/types"
)

// TabKind identifies the type of a tab.
type TabKind int

const (
	TabKindFile  TabKind = iota // file editing tab
	TabKindGraph                // git graph tab
)

// TabInfo holds the state of a single editor tab.
type TabInfo struct {
	Kind     TabKind
	Buffer   *buffer.Buffer
	Cursor   types.Position
	ScrollY  int
	ScrollX  int
	Language string
}

// TabManager manages open file tabs.
type TabManager struct {
	tabs      []*TabInfo
	activeIdx int
}

// NewTabManager creates a new TabManager.
func NewTabManager() *TabManager {
	return &TabManager{}
}

// Open adds a new tab with the given buffer.
func (tm *TabManager) Open(buf *buffer.Buffer, language string) int {
	tab := &TabInfo{
		Buffer:   buf,
		Language: language,
	}
	tm.tabs = append(tm.tabs, tab)
	tm.activeIdx = len(tm.tabs) - 1
	return tm.activeIdx
}

// Close removes the tab at the given index.
// Returns the new active tab index, or -1 if no tabs remain.
func (tm *TabManager) Close(idx int) int {
	if idx < 0 || idx >= len(tm.tabs) {
		return tm.activeIdx
	}
	tm.tabs = append(tm.tabs[:idx], tm.tabs[idx+1:]...)

	if len(tm.tabs) == 0 {
		tm.activeIdx = -1
		return -1
	}

	if tm.activeIdx >= len(tm.tabs) {
		tm.activeIdx = len(tm.tabs) - 1
	}
	return tm.activeIdx
}

// Active returns the currently active tab, or nil.
func (tm *TabManager) Active() *TabInfo {
	if tm.activeIdx < 0 || tm.activeIdx >= len(tm.tabs) {
		return nil
	}
	return tm.tabs[tm.activeIdx]
}

// ActiveIndex returns the active tab index.
func (tm *TabManager) ActiveIndex() int {
	return tm.activeIdx
}

// SetActive sets the active tab index.
func (tm *TabManager) SetActive(idx int) {
	if idx >= 0 && idx < len(tm.tabs) {
		tm.activeIdx = idx
	}
}

// Next switches to the next tab.
func (tm *TabManager) Next() {
	if len(tm.tabs) <= 1 {
		return
	}
	tm.activeIdx = (tm.activeIdx + 1) % len(tm.tabs)
}

// Previous switches to the previous tab.
func (tm *TabManager) Previous() {
	if len(tm.tabs) <= 1 {
		return
	}
	tm.activeIdx = (tm.activeIdx - 1 + len(tm.tabs)) % len(tm.tabs)
}

// Count returns the number of open tabs.
func (tm *TabManager) Count() int {
	return len(tm.tabs)
}

// Tab returns the tab at the given index.
func (tm *TabManager) Tab(idx int) *TabInfo {
	if idx < 0 || idx >= len(tm.tabs) {
		return nil
	}
	return tm.tabs[idx]
}

// All returns all tabs.
func (tm *TabManager) All() []*TabInfo {
	return tm.tabs
}

// FindByPath returns the tab index for a file path, or -1 if not found.
func (tm *TabManager) FindByPath(path string) int {
	for i, tab := range tm.tabs {
		if tab.Buffer.Path() == path {
			return i
		}
	}
	return -1
}
