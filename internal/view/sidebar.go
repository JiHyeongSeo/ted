package view

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/seoji/ted/internal/syntax"
)

// fileIcon returns a Nerd Font icon string and color for a file/directory entry.
func fileIcon(name string, isDir, expanded bool) (string, tcell.Color) {
	if isDir {
		if expanded {
			return "\uf115", tcell.NewRGBColor(220, 180, 60) //  (folder open)
		}
		return "\uf114", tcell.NewRGBColor(220, 180, 60) //  (folder)
	}

	ext := strings.ToLower(filepath.Ext(name))
	lower := strings.ToLower(name)

	// Check by full name first
	switch lower {
	case "dockerfile":
		return "\uf308", tcell.NewRGBColor(56, 151, 240) //  Docker
	case "makefile":
		return "\uf085", tcell.NewRGBColor(111, 66, 193) //  gears
	case "license":
		return "\uf0e3", tcell.NewRGBColor(180, 180, 180) //  gavel
	case "readme.md", "readme":
		return "\uf48a", tcell.NewRGBColor(55, 160, 80) //  book
	case ".gitignore":
		return "\ue702", tcell.NewRGBColor(244, 76, 48) //  git
	}

	switch ext {
	case ".go":
		return "\ue626", tcell.NewRGBColor(0, 173, 216) //  Go
	case ".py":
		return "\ue73c", tcell.NewRGBColor(55, 160, 80) //  Python
	case ".js":
		return "\ue74e", tcell.NewRGBColor(241, 224, 90) //  JavaScript
	case ".ts":
		return "\ue628", tcell.NewRGBColor(49, 120, 198) //  TypeScript
	case ".jsx", ".tsx":
		return "\ue7ba", tcell.NewRGBColor(97, 218, 251) //  React
	case ".html", ".htm":
		return "\ue736", tcell.NewRGBColor(227, 76, 38) //  HTML5
	case ".css":
		return "\ue749", tcell.NewRGBColor(86, 61, 124) //  CSS3
	case ".scss", ".sass":
		return "\ue74b", tcell.NewRGBColor(205, 103, 153) //  Sass
	case ".json":
		return "\ue60b", tcell.NewRGBColor(241, 224, 90) //  braces
	case ".yaml", ".yml":
		return "\ue6a8", tcell.NewRGBColor(203, 56, 55) //  yaml
	case ".toml":
		return "\ue615", tcell.NewRGBColor(156, 154, 150) //  config
	case ".md":
		return "\ue73e", tcell.NewRGBColor(69, 137, 230) //  Markdown
	case ".txt", ".rst":
		return "\uf15c", tcell.NewRGBColor(180, 180, 180) //  file-text
	case ".sh", ".bash", ".zsh":
		return "\uf489", tcell.NewRGBColor(77, 170, 87) //  terminal
	case ".rs":
		return "\ue7a8", tcell.NewRGBColor(222, 165, 132) //  Rust
	case ".java":
		return "\ue738", tcell.NewRGBColor(204, 62, 68) //  Java
	case ".c", ".h":
		return "\ue61e", tcell.NewRGBColor(85, 85, 255) //  C
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "\ue61d", tcell.NewRGBColor(0, 89, 156) //  C++
	case ".rb":
		return "\ue739", tcell.NewRGBColor(204, 52, 45) //  Ruby
	case ".sql":
		return "\uf1c0", tcell.NewRGBColor(226, 131, 35) //  database
	case ".xml":
		return "\ue619", tcell.NewRGBColor(227, 76, 38) //  code
	case ".svg":
		return "\uf1c5", tcell.NewRGBColor(255, 177, 59) //  image
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".ico", ".webp":
		return "\uf1c5", tcell.NewRGBColor(160, 100, 200) //  image
	case ".zip", ".tar", ".gz", ".bz2", ".rar", ".7z":
		return "\uf1c6", tcell.NewRGBColor(180, 142, 100) //  archive
	case ".lock":
		return "\uf023", tcell.NewRGBColor(120, 120, 120) //  lock
	case ".env":
		return "\uf462", tcell.NewRGBColor(234, 183, 0) //  key
	case ".gitignore":
		return "\ue702", tcell.NewRGBColor(244, 76, 48) //  git
	case ".mod", ".sum":
		return "\ue626", tcell.NewRGBColor(0, 173, 216) //  Go
	default:
		return "\uf15b", tcell.NewRGBColor(180, 180, 180) //  file
	}
}

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

	// Draw header bar — accent color when focused, dim when not
	var headerStyle tcell.Style
	if s.IsFocused() {
		headerStyle = tcell.StyleDefault.Background(tcell.NewRGBColor(0, 122, 204)).Foreground(tcell.ColorWhite) // #007acc
	} else {
		headerStyle = style.Foreground(tcell.NewRGBColor(140, 140, 140))
	}
	for x := bounds.X; x < bounds.X+bounds.Width; x++ {
		screen.SetContent(x, bounds.Y, ' ', nil, headerStyle)
	}
	label := " EXPLORER"
	lx := bounds.X
	for _, ch := range label {
		if lx >= bounds.X+bounds.Width {
			break
		}
		screen.SetContent(lx, bounds.Y, ch, nil, headerStyle)
		lx++
	}

	// Draw entries (starting below header)
	visibleRows := bounds.Height - 1
	if visibleRows < 0 {
		visibleRows = 0
	}
	for i := 0; i < visibleRows && i+s.scrollY < len(s.flatEntries); i++ {
		entry := s.flatEntries[i+s.scrollY]
		y := bounds.Y + 1 + i
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

		// Icon with color
		icon, iconColor := fileIcon(entry.Name, entry.IsDir, entry.Expanded)
		iconStyle := rowStyle.Foreground(iconColor)
		for _, ch := range icon {
			w := runewidth.RuneWidth(ch)
			if x+w > bounds.X+bounds.Width {
				break
			}
			screen.SetContent(x, y, ch, nil, iconStyle)
			x += w
		}
		if x < bounds.X+bounds.Width {
			screen.SetContent(x, y, ' ', nil, rowStyle)
			x++
		}

		// Name with file-type color
		nameStyle := rowStyle.Foreground(iconColor)
		if entry.IsDir {
			nameStyle = rowStyle.Foreground(iconColor)
		}
		for _, ch := range entry.Name {
			w := runewidth.RuneWidth(ch)
			if x+w > bounds.X+bounds.Width {
				break
			}
			screen.SetContent(x, y, ch, nil, nameStyle)
			x += w
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
	case tcell.KeyPgUp:
		pageSize := s.Bounds().Height - 1
		if pageSize < 1 {
			pageSize = 1
		}
		s.selectedIdx -= pageSize
		if s.selectedIdx < 0 {
			s.selectedIdx = 0
		}
		s.ensureVisible()
		return true
	case tcell.KeyPgDn:
		pageSize := s.Bounds().Height - 1
		if pageSize < 1 {
			pageSize = 1
		}
		s.selectedIdx += pageSize
		if s.selectedIdx >= len(s.flatEntries) {
			s.selectedIdx = len(s.flatEntries) - 1
		}
		if s.selectedIdx < 0 {
			s.selectedIdx = 0
		}
		s.ensureVisible()
		return true
	case tcell.KeyHome:
		s.selectedIdx = 0
		s.ensureVisible()
		return true
	case tcell.KeyEnd:
		if len(s.flatEntries) > 0 {
			s.selectedIdx = len(s.flatEntries) - 1
		}
		s.ensureVisible()
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
	visibleRows := bounds.Height - 1 // account for header row
	if visibleRows < 1 {
		visibleRows = 1
	}
	if s.selectedIdx < s.scrollY {
		s.scrollY = s.selectedIdx
	}
	if s.selectedIdx >= s.scrollY+visibleRows {
		s.scrollY = s.selectedIdx - visibleRows + 1
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
