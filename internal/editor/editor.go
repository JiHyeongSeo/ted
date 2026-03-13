package editor

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/seoji/ted/internal/buffer"
	"github.com/seoji/ted/internal/config"
	"github.com/seoji/ted/internal/input"
	"github.com/seoji/ted/internal/search"
	"github.com/seoji/ted/internal/syntax"
	"github.com/seoji/ted/internal/types"
	"github.com/seoji/ted/internal/view"
)

// Editor is the top-level editor state and event loop orchestrator.
type Editor struct {
	screen     tcell.Screen
	config     *config.Config
	theme      *syntax.Theme
	tabs       *TabManager
	commands   *CommandRegistry
	keymap     *input.Keymap
	layout     *view.Layout
	editorView *view.EditorView
	statusBar  *view.StatusBar
	tabBar     *view.TabBar
	sidebar    *view.Sidebar
	panel      *view.BottomPanel
	palette    *view.CommandPalette
	searchBar    *view.SearchBar
	running      bool
	sidebarFocus bool // true when sidebar has keyboard focus
	quitPending  bool // true when quit requested with unsaved changes
}

// New creates a new Editor instance.
func New(cfg *config.Config, theme *syntax.Theme) *Editor {
	e := &Editor{
		config:   cfg,
		theme:    theme,
		tabs:     NewTabManager(),
		commands: NewCommandRegistry(),
		keymap:   input.NewKeymap(),
		layout:   view.NewLayout(),
	}

	e.layout.SetSidebarWidth(cfg.Sidebar.Width)
	e.layout.SetSidebarVisible(cfg.Sidebar.Visible)
	e.layout.SetPanelHeight(cfg.Panel.Height)
	e.layout.SetPanelVisible(cfg.Panel.Visible)

	e.statusBar = view.NewStatusBar(theme)
	e.tabBar = view.NewTabBar(theme)
	e.sidebar = view.NewSidebar(theme)
	e.panel = view.NewBottomPanel(theme)
	e.palette = view.NewCommandPalette(theme)
	e.searchBar = view.NewSearchBar(theme)

	// Wire palette callbacks
	e.palette.SetOnSelect(func(item view.PaletteItem) {
		e.ExecuteCommand(item.Command)
	})
	e.palette.SetOnDismiss(func() {})

	// Wire sidebar callback
	e.sidebar.SetOnFileOpen(func(path string) {
		e.OpenFile(path)
	})

	// Wire search bar callbacks
	e.searchBar.SetOnSearch(func(query string) {
		e.performSearch(query)
	})
	e.searchBar.SetOnReplace(func(query, replacement string) {
		e.performReplace(query, replacement)
	})
	e.searchBar.SetOnReplaceAll(func(query, replacement string) {
		e.performReplaceAll(query, replacement)
	})
	e.searchBar.SetOnDismiss(func() {
		// Clear search highlights when dismissed
	})

	e.registerCommands()

	return e
}

// OpenFile opens a file in a new tab.
func (e *Editor) OpenFile(path string) error {
	if idx := e.tabs.FindByPath(path); idx >= 0 {
		e.tabs.SetActive(idx)
		e.syncViewToTab()
		return nil
	}

	buf, err := buffer.OpenFile(path)
	if err != nil {
		return err
	}

	lang := detectLanguage(path)
	e.tabs.Open(buf, lang)
	e.syncViewToTab()
	return nil
}

// OpenDirectory opens a directory in the sidebar.
func (e *Editor) OpenDirectory(path string) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	e.sidebar.SetRoot(absPath)
	e.layout.SetSidebarVisible(true)
	e.sidebarFocus = false // start with editor focus
}

// OpenEmpty opens a new empty buffer tab.
func (e *Editor) OpenEmpty() {
	buf := buffer.NewBuffer("")
	e.tabs.Open(buf, "text")
	e.syncViewToTab()
}

// Run starts the editor event loop.
func (e *Editor) Run(screen tcell.Screen) error {
	e.screen = screen
	e.running = true

	// Set cursor style to thin beam (bar)
	screen.SetCursorStyle(tcell.CursorStyleSteadyBar)

	if e.tabs.Count() == 0 {
		e.OpenEmpty()
	}

	// Set sidebar root to current working directory
	if cwd, err := os.Getwd(); err == nil {
		e.sidebar.SetRoot(cwd)
	}

	// Populate palette with all commands
	e.updatePaletteItems()

	e.render()

	for e.running {
		ev := screen.PollEvent()
		if ev == nil {
			break
		}

		switch tev := ev.(type) {
		case *tcell.EventResize:
			screen.Sync()
			e.render()
		case *tcell.EventKey:
			e.handleKeyEvent(tev)
			e.render()
		case *tcell.EventMouse:
			e.handleMouseEvent(tev)
			e.render()
		}
	}

	return nil
}

// Stop signals the editor to exit.
func (e *Editor) Stop() {
	e.running = false
}

func (e *Editor) handleKeyEvent(ev *tcell.EventKey) {
	// Palette gets priority when visible
	if e.palette.IsVisible() {
		e.palette.HandleEvent(ev)
		return
	}

	// SearchBar gets priority when visible
	if e.searchBar.IsVisible() {
		e.searchBar.HandleEvent(ev)
		return
	}

	// Try keymap (global shortcuts work regardless of focus)
	cmd, result := e.keymap.Resolve(ev, "editor")
	switch result {
	case input.ResolveMatched:
		e.ExecuteCommand(cmd)
		return
	case input.ResolvePending:
		return
	}

	// Sidebar keyboard navigation when focused
	if e.sidebarFocus && e.layout.SidebarVisible() {
		// Escape returns focus to editor
		if ev.Key() == tcell.KeyEscape {
			e.sidebarFocus = false
			return
		}
		e.sidebar.SetFocused(true)
		e.sidebar.HandleEvent(ev)
		e.sidebar.SetFocused(false)
		return
	}

	// Pass to editor view for text input
	if e.editorView != nil {
		e.editorView.HandleEvent(ev)
		e.syncTabFromView()
	}
}

func (e *Editor) handleMouseEvent(ev *tcell.EventMouse) {
	mx, my := ev.Position()
	btn := ev.Buttons()

	// Tab bar click
	if e.tabBar != nil {
		oldIdx := e.tabs.ActiveIndex()
		if e.tabBar.HandleEvent(ev) {
			newIdx := e.tabBar.ActiveIndex()
			if newIdx != oldIdx {
				e.tabs.SetActive(newIdx)
				e.syncViewToTab()
			}
			return
		}
	}

	// Sidebar click
	if e.layout.SidebarVisible() && btn == tcell.Button1 {
		sb := e.sidebar.Bounds()
		if mx >= sb.X && mx < sb.X+sb.Width && my >= sb.Y && my < sb.Y+sb.Height {
			e.sidebarFocus = true
			// Calculate which entry was clicked
			row := my - sb.Y + e.sidebar.ScrollY()
			e.sidebar.SelectIndex(row)
			return
		}
		// Clicked outside sidebar — return focus to editor
		e.sidebarFocus = false
	}

	// Double-click in sidebar opens file
	if e.layout.SidebarVisible() && btn == tcell.ButtonNone {
		// tcell doesn't expose double-click natively; Enter key handles this
	}
}

func (e *Editor) render() {
	if e.screen == nil {
		return
	}

	w, h := e.screen.Size()
	regions := e.layout.Compute(w, h)

	e.screen.Clear()

	// Render tab bar
	if r, ok := regions["tabbar"]; ok {
		e.tabBar.SetBounds(r)
		tabs := make([]view.Tab, e.tabs.Count())
		for i, t := range e.tabs.All() {
			tabs[i] = view.Tab{
				Title:    filepath.Base(t.Buffer.Path()),
				FilePath: t.Buffer.Path(),
				Dirty:    t.Buffer.IsDirty(),
			}
		}
		e.tabBar.SetTabs(tabs, e.tabs.ActiveIndex())
		e.tabBar.Render(e.screen)
	}

	// Render sidebar
	if r, ok := regions["sidebar"]; ok {
		e.sidebar.SetBounds(r)
		e.sidebar.Render(e.screen)
	}

	// Render sidebar separator
	if r, ok := regions["separator"]; ok {
		sepStyle := e.theme.UIStyle("sidebar").Foreground(tcell.ColorDarkGray)
		for y := r.Y; y < r.Y+r.Height; y++ {
			e.screen.SetContent(r.X, y, '│', nil, sepStyle)
		}
	}

	// Render editor view
	if r, ok := regions["editor"]; ok && e.editorView != nil {
		e.editorView.SetBounds(r)
		e.editorView.SetFocused(true)
		e.editorView.Render(e.screen)
	}

	// Render panel
	if r, ok := regions["panel"]; ok {
		e.panel.SetBounds(r)
		e.panel.Render(e.screen)
	}

	// Render status bar
	if r, ok := regions["statusbar"]; ok {
		e.statusBar.SetBounds(r)
		tab := e.tabs.Active()
		if tab != nil {
			e.statusBar.Update(tab.Buffer.Path(), tab.Language, tab.Cursor.Line, tab.Cursor.Col, tab.Buffer.IsDirty())
		}
		e.statusBar.Render(e.screen)
	}

	// Render overlays (palette, searchbar) on top
	if e.searchBar.IsVisible() {
		if r, ok := regions["editor"]; ok {
			e.searchBar.SetBounds(r)
			e.searchBar.Render(e.screen)
		}
	}
	if e.palette.IsVisible() {
		e.palette.SetBoundsFromScreen(w, h)
		e.palette.Render(e.screen)
	}

	e.screen.Show()
}

func (e *Editor) syncViewToTab() {
	tab := e.tabs.Active()
	if tab == nil {
		e.editorView = nil
		return
	}

	e.editorView = view.NewEditorView(tab.Buffer, e.theme)
	e.editorView.SetLanguage(tab.Language)
	e.editorView.SetCursorPosition(tab.Cursor)
	e.editorView.SetScrollY(tab.ScrollY)
}

func (e *Editor) syncTabFromView() {
	tab := e.tabs.Active()
	if tab == nil || e.editorView == nil {
		return
	}
	tab.Cursor = e.editorView.CursorPosition()
	tab.ScrollY, tab.ScrollX = e.editorView.ScrollPosition()
}

func (e *Editor) registerCommands() {
	RegisterBuiltinCommands(e.commands)

	e.commands.Register(&Command{
		Name:        "editor.quit",
		Description: "Quit the editor",
		Execute: func(ctx EditorContext) error {
			if ed, ok := ctx.(*Editor); ok {
				ed.Stop()
			}
			return nil
		},
	})
}

func (e *Editor) updatePaletteItems() {
	cmds := e.commands.Commands()
	items := make([]view.PaletteItem, len(cmds))
	for i, cmd := range cmds {
		items[i] = view.PaletteItem{
			Label:       cmd.Name,
			Description: cmd.Description,
			Command:     cmd.Name,
		}
	}
	e.palette.SetItems(items)
}

// LoadKeybindings loads key bindings from the default keybindings config.
func (e *Editor) LoadKeybindings() {
	e.keymap.Bind("ctrl+s", "file.save", "")
	e.keymap.Bind("ctrl+o", "file.open", "")
	e.keymap.Bind("ctrl+w", "file.close", "")
	e.keymap.Bind("ctrl+q", "editor.quit", "")
	e.keymap.Bind("ctrl+z", "edit.undo", "")
	e.keymap.Bind("ctrl+y", "edit.redo", "")
	e.keymap.Bind("ctrl+c", "edit.copy", "")
	e.keymap.Bind("ctrl+x", "edit.cut", "")
	e.keymap.Bind("ctrl+v", "edit.paste", "")
	e.keymap.Bind("ctrl+f", "search.find", "")
	e.keymap.Bind("ctrl+h", "search.replace", "")
	e.keymap.Bind("ctrl+p", "palette.open", "")
	e.keymap.Bind("ctrl+b", "sidebar.toggle", "")
	e.keymap.Bind("ctrl+j", "panel.toggle", "")
	e.keymap.Bind("alt+1", "tab.goto.1", "")
	e.keymap.Bind("alt+2", "tab.goto.2", "")
	e.keymap.Bind("alt+3", "tab.goto.3", "")
	e.keymap.Bind("alt+4", "tab.goto.4", "")
	e.keymap.Bind("alt+5", "tab.goto.5", "")
	e.keymap.Bind("alt+6", "tab.goto.6", "")
	e.keymap.Bind("alt+7", "tab.goto.7", "")
	e.keymap.Bind("alt+8", "tab.goto.8", "")
	e.keymap.Bind("alt+9", "tab.goto.9", "")
	e.keymap.Bind("f12", "lsp.goToDefinition", "")
	e.keymap.Bind("shift+f12", "lsp.findReferences", "")
	e.keymap.Bind("ctrl+space", "lsp.autocomplete", "")
	e.keymap.Bind("ctrl+k ctrl+i", "lsp.hover", "")
}

// --- EditorContext interface implementation ---

// ActiveBuffer returns the active buffer.
func (e *Editor) ActiveBuffer() interface{ Text() string } {
	tab := e.tabs.Active()
	if tab == nil {
		return nil
	}
	return tab.Buffer
}

// ExecuteCommand dispatches a command by name.
func (e *Editor) ExecuteCommand(name string) error {
	// Clear quit warning on any other action
	if name != "editor.quit" && e.quitPending {
		e.quitPending = false
		e.statusBar.ClearMessage()
	}

	switch name {
	case "file.save":
		if tab := e.tabs.Active(); tab != nil {
			return tab.Buffer.Save()
		}
	case "file.close":
		idx := e.tabs.ActiveIndex()
		e.tabs.Close(idx)
		e.syncViewToTab()
	case "edit.undo":
		if tab := e.tabs.Active(); tab != nil {
			tab.Buffer.Undo()
		}
	case "edit.redo":
		if tab := e.tabs.Active(); tab != nil {
			tab.Buffer.Redo()
		}
	case "edit.copy":
		if e.editorView != nil {
			e.editorView.Copy()
		}
	case "edit.cut":
		if e.editorView != nil {
			e.editorView.Cut()
			e.syncTabFromView()
		}
	case "edit.paste":
		if e.editorView != nil {
			e.editorView.Paste()
			e.syncTabFromView()
		}
	case "tab.next":
		e.tabs.Next()
		e.syncViewToTab()
	case "tab.previous":
		e.tabs.Previous()
		e.syncViewToTab()
	case "tab.goto.1", "tab.goto.2", "tab.goto.3", "tab.goto.4", "tab.goto.5",
		"tab.goto.6", "tab.goto.7", "tab.goto.8", "tab.goto.9":
		idx := int(name[len(name)-1]-'1') // "tab.goto.1" → 0
		if idx >= 0 && idx < e.tabs.Count() {
			e.tabs.SetActive(idx)
			e.syncViewToTab()
		}
	case "sidebar.toggle":
		if !e.layout.SidebarVisible() {
			// Hidden → show + focus sidebar
			e.layout.SetSidebarVisible(true)
			e.sidebarFocus = true
		} else if e.sidebarFocus {
			// Sidebar focused → hide sidebar + return to editor
			e.layout.SetSidebarVisible(false)
			e.sidebarFocus = false
		} else {
			// Sidebar visible, editor focused → focus sidebar
			e.sidebarFocus = true
		}
	case "panel.toggle":
		e.layout.SetPanelVisible(!e.layout.PanelVisible())
	case "palette.open":
		e.palette.Show()
	case "search.find":
		e.searchBar.Show(false)
	case "search.replace":
		e.searchBar.Show(true)
	case "editor.quit":
		e.tryQuit()
	}
	return nil
}

// tryQuit handles quit with unsaved changes warning.
func (e *Editor) tryQuit() {
	if e.quitPending {
		// Second Ctrl+Q — force quit
		e.Stop()
		return
	}
	// Check if any tab has unsaved changes
	for _, tab := range e.tabs.All() {
		if tab.Buffer.IsDirty() {
			e.quitPending = true
			e.statusBar.SetMessage("Unsaved changes! Press Ctrl+Q again to quit without saving.")
			return
		}
	}
	e.Stop()
}

func (e *Editor) performSearch(query string) {
	if query == "" || e.editorView == nil {
		e.searchBar.SetMatches(nil)
		return
	}
	tab := e.tabs.Active()
	if tab == nil {
		return
	}
	s, err := search.NewInFileSearch(query, false, false)
	if err != nil {
		e.searchBar.SetMatches(nil)
		return
	}
	matches := s.FindAll(tab.Buffer.Text())
	e.searchBar.SetMatches(matches)

	// Jump to first match after current cursor
	if len(matches) > 0 {
		cursor := e.editorView.CursorPosition()
		for _, m := range matches {
			if m.Line > cursor.Line || (m.Line == cursor.Line && m.Col > cursor.Col) {
				e.editorView.SetCursorPosition(types.Position{Line: m.Line, Col: m.Col})
				e.syncTabFromView()
				return
			}
		}
		// Wrap: jump to first match
		e.editorView.SetCursorPosition(types.Position{Line: matches[0].Line, Col: matches[0].Col})
		e.syncTabFromView()
	}
}

func (e *Editor) performReplace(query, replacement string) {
	tab := e.tabs.Active()
	if tab == nil || e.editorView == nil {
		return
	}
	// Find the match at cursor position
	cursor := e.editorView.CursorPosition()
	s, err := search.NewInFileSearch(query, false, false)
	if err != nil {
		return
	}
	matches := s.FindAll(tab.Buffer.Text())
	for _, m := range matches {
		if m.Line == cursor.Line && m.Col == cursor.Col {
			tab.Buffer.Delete(m.Line, m.Col, m.Length)
			tab.Buffer.Insert(m.Line, m.Col, replacement)
			break
		}
	}
	// Re-search to update matches
	e.performSearch(query)
}

func (e *Editor) performReplaceAll(query, replacement string) {
	tab := e.tabs.Active()
	if tab == nil {
		return
	}
	s, err := search.NewInFileSearch(query, false, false)
	if err != nil {
		return
	}
	matches := s.FindAll(tab.Buffer.Text())
	// Replace from bottom to top so offsets don't shift
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]
		tab.Buffer.Delete(m.Line, m.Col, m.Length)
		tab.Buffer.Insert(m.Line, m.Col, replacement)
	}
	e.performSearch(query)
}

func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".rs":
		return "rust"
	case ".c", ".h":
		return "c"
	case ".cpp", ".hpp", ".cc":
		return "cpp"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	case ".md":
		return "markdown"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".sh", ".bash":
		return "bash"
	case ".html", ".htm":
		return "html"
	case ".css":
		return "css"
	default:
		return "text"
	}
}
