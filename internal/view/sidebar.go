package view

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/seoji/ted/internal/syntax"
)

// FileEntry represents a file or directory in the sidebar tree.
type FileEntry struct {
	Name     string
	Path     string
	IsDir    bool
	Depth    int
	Expanded bool
	Children []*FileEntry
}

// Sidebar displays a file tree explorer.
type Sidebar struct {
	BaseComponent
	theme       *syntax.Theme
	root        string
	entries     []*FileEntry
	flatEntries []*FileEntry // flattened visible entries
	selectedIdx int
	scrollY     int
	onFileOpen  func(path string)
}

// NewSidebar creates a new Sidebar.
func NewSidebar(theme *syntax.Theme) *Sidebar {
	return &Sidebar{
		theme: theme,
	}
}

// SetRoot sets the root directory and loads the file tree.
func (s *Sidebar) SetRoot(root string) {
	s.root = root
	s.entries = s.loadDir(root, 0)
	s.rebuildFlat()
}

// SetOnFileOpen sets the callback for when a file is double-clicked or entered.
func (s *Sidebar) SetOnFileOpen(fn func(path string)) {
	s.onFileOpen = fn
}

// Render draws the sidebar.
func (s *Sidebar) Render(screen tcell.Screen) {
	bounds := s.Bounds()
	style := s.theme.UIStyle("sidebar")
	selStyle := s.theme.UIStyle("selection")

	// Clear area
	for y := bounds.Y; y < bounds.Y+bounds.Height; y++ {
		for x := bounds.X; x < bounds.X+bounds.Width; x++ {
			screen.SetContent(x, y, ' ', nil, style)
		}
	}

	// Draw entries
	visibleRows := bounds.Height
	for i := 0; i < visibleRows && i+s.scrollY < len(s.flatEntries); i++ {
		entry := s.flatEntries[i+s.scrollY]
		y := bounds.Y + i
		rowStyle := style
		if i+s.scrollY == s.selectedIdx {
			rowStyle = selStyle
		}

		// Clear row
		for x := bounds.X; x < bounds.X+bounds.Width; x++ {
			screen.SetContent(x, y, ' ', nil, rowStyle)
		}

		// Indent
		x := bounds.X + entry.Depth*2

		// Icon
		icon := ' '
		if entry.IsDir {
			if entry.Expanded {
				icon = '▼'
			} else {
				icon = '▶'
			}
		} else {
			icon = ' '
		}
		if x < bounds.X+bounds.Width {
			screen.SetContent(x, y, icon, nil, rowStyle)
			x++
		}
		if x < bounds.X+bounds.Width {
			screen.SetContent(x, y, ' ', nil, rowStyle)
			x++
		}

		// Name
		for _, ch := range entry.Name {
			if x >= bounds.X+bounds.Width {
				break
			}
			screen.SetContent(x, y, ch, nil, rowStyle)
			x++
		}
	}
}

// HandleEvent processes keyboard events for the sidebar.
func (s *Sidebar) HandleEvent(ev tcell.Event) bool {
	if !s.IsFocused() {
		return false
	}

	keyEv, ok := ev.(*tcell.EventKey)
	if !ok {
		return false
	}

	switch keyEv.Key() {
	case tcell.KeyUp:
		if s.selectedIdx > 0 {
			s.selectedIdx--
			s.ensureVisible()
		}
		return true
	case tcell.KeyDown:
		if s.selectedIdx < len(s.flatEntries)-1 {
			s.selectedIdx++
			s.ensureVisible()
		}
		return true
	case tcell.KeyEnter:
		if s.selectedIdx >= 0 && s.selectedIdx < len(s.flatEntries) {
			entry := s.flatEntries[s.selectedIdx]
			if entry.IsDir {
				entry.Expanded = !entry.Expanded
				s.rebuildFlat()
			} else if s.onFileOpen != nil {
				s.onFileOpen(entry.Path)
			}
		}
		return true
	}

	return false
}

// SelectIndex sets the selected entry by flat index and triggers action.
func (s *Sidebar) SelectIndex(idx int) {
	if idx < 0 || idx >= len(s.flatEntries) {
		return
	}
	s.selectedIdx = idx
	entry := s.flatEntries[idx]
	if entry.IsDir {
		entry.Expanded = !entry.Expanded
		s.rebuildFlat()
	} else if s.onFileOpen != nil {
		s.onFileOpen(entry.Path)
	}
}

// ScrollY returns the current scroll offset.
func (s *Sidebar) ScrollY() int {
	return s.scrollY
}

func (s *Sidebar) ensureVisible() {
	bounds := s.Bounds()
	if s.selectedIdx < s.scrollY {
		s.scrollY = s.selectedIdx
	}
	if s.selectedIdx >= s.scrollY+bounds.Height {
		s.scrollY = s.selectedIdx - bounds.Height + 1
	}
}

func (s *Sidebar) loadDir(dir string, depth int) []*FileEntry {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var entries []*FileEntry

	// Sort: directories first, then alphabetical
	sort.Slice(dirEntries, func(i, j int) bool {
		if dirEntries[i].IsDir() != dirEntries[j].IsDir() {
			return dirEntries[i].IsDir()
		}
		return strings.ToLower(dirEntries[i].Name()) < strings.ToLower(dirEntries[j].Name())
	})

	for _, de := range dirEntries {
		name := de.Name()
		// Skip hidden files and common ignore patterns
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "__pycache__" {
			continue
		}

		entry := &FileEntry{
			Name:  name,
			Path:  filepath.Join(dir, name),
			IsDir: de.IsDir(),
			Depth: depth,
		}

		if de.IsDir() {
			entry.Children = s.loadDir(entry.Path, depth+1)
		}

		entries = append(entries, entry)
	}

	return entries
}

func (s *Sidebar) rebuildFlat() {
	s.flatEntries = nil
	s.flattenEntries(s.entries)
}

func (s *Sidebar) flattenEntries(entries []*FileEntry) {
	for _, e := range entries {
		s.flatEntries = append(s.flatEntries, e)
		if e.IsDir && e.Expanded {
			s.flattenEntries(e.Children)
		}
	}
}
