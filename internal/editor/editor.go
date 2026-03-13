package editor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/seoji/ted/internal/buffer"
	"github.com/seoji/ted/internal/config"
	"github.com/seoji/ted/internal/input"
	"github.com/seoji/ted/internal/lsp"
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
	inputBar     *view.InputBar
	running      bool
	sidebarFocus bool // true when sidebar has keyboard focus
	quitPending  bool // true when quit requested with unsaved changes
	lspManager   *lsp.ServerManager
	lspHandler   *lsp.NotificationHandler
	projectRoot  string // root directory for project search and LSP
	projectSearchResults []search.FileMatch // cached project search results
	projectSearchQuery   string              // the query used for project search
}

// New creates a new Editor instance.
func New(cfg *config.Config, theme *syntax.Theme) *Editor {
	// Configure LSP servers
	lspConfigs := map[string]lsp.ServerConfig{
		"go": {Command: "gopls", Args: []string{"serve"}},
	}

	lspHandler := lsp.NewNotificationHandler()

	e := &Editor{
		config:     cfg,
		theme:      theme,
		tabs:       NewTabManager(),
		commands:   NewCommandRegistry(),
		keymap:     input.NewKeymap(),
		layout:     view.NewLayout(),
		lspManager: lsp.NewServerManager(lspConfigs),
		lspHandler: lspHandler,
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
	e.inputBar = view.NewInputBar(theme)

	// Wire LSP diagnostic handler
	e.lspManager.SetDiagnosticHandler(func(uri string, diags []lsp.Diagnostic) {
		e.lspHandler.HandleDiagnostics(uri, diags)
		// Update problems panel
		var lines []string
		for u, ds := range e.lspHandler.GetAllDiagnostics() {
			for _, d := range ds {
				lines = append(lines, lsp.FormatDiagnostic(u, d))
			}
		}
		e.panel.SetContent(0, lines) // "Problems" tab
	})

	// Wire palette callbacks
	e.palette.SetOnSelect(func(item view.PaletteItem) {
		e.ExecuteCommand(item.Command)
	})
	e.palette.SetOnDismiss(func() {})

	// Wire sidebar callback
	e.sidebar.SetOnFileOpen(func(path string) {
		e.OpenFile(path)
	})

	// Wire panel click callback — navigate to search result / diagnostic
	e.panel.SetOnLineClick(func(tabIdx int, lineIdx int) {
		e.handlePanelLineClick(tabIdx, lineIdx)
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
		if e.editorView != nil {
			e.editorView.ClearSearchHighlights()
		}
	})

	// Wire input bar callbacks (for Ctrl+G go to line)
	e.inputBar.SetOnSubmit(func(value string) {
		lineNum, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || lineNum < 1 {
			e.statusBar.SetMessage("Invalid line number")
			return
		}
		if e.editorView != nil {
			e.editorView.SetCursorPosition(types.Position{Line: lineNum - 1, Col: 0})
			e.syncTabFromView()
		}
	})
	e.inputBar.SetOnCancel(func() {})

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

	// Start LSP if needed and notify didOpen
	e.ensureLSP()
	tab := e.tabs.Active()
	if tab != nil {
		e.lspNotifyOpen(tab)
	}

	return nil
}

// OpenDirectory opens a directory in the sidebar.
func (e *Editor) OpenDirectory(path string) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	e.sidebar.SetRoot(absPath)
	e.projectRoot = absPath
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

	// Set sidebar root and project root to current working directory
	if cwd, err := os.Getwd(); err == nil {
		e.sidebar.SetRoot(cwd)
		if e.projectRoot == "" {
			e.projectRoot = cwd
		}
	}

	// Populate palette with all commands
	e.updatePaletteItems()

	// Start LSP for active tab's language
	e.ensureLSP()

	defer e.lspManager.StopAll()

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

	// InputBar gets priority when visible (Ctrl+G)
	if e.inputBar.IsVisible() {
		e.inputBar.HandleEvent(ev)
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

	if btn == tcell.Button1 {
		// Dismiss input bar on click outside it
		if e.inputBar.IsVisible() {
			ib := e.inputBar.Bounds()
			if !(mx >= ib.X && mx < ib.X+ib.Width && my == ib.Y) {
				e.inputBar.Hide()
			}
		}
		// Search bar stays open on click — user can click to move cursor
		// while keeping search results visible (VS Code behavior)
	}

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

	// Panel click — navigate to result
	if e.layout.PanelVisible() {
		pb := e.panel.Bounds()
		if mx >= pb.X && mx < pb.X+pb.Width && my >= pb.Y && my < pb.Y+pb.Height {
			e.panel.HandleEvent(ev)
			return
		}
	}

	// Editor area click — move cursor
	if btn == tcell.Button1 && e.editorView != nil {
		eb := e.editorView.Bounds()
		if mx >= eb.X && mx < eb.X+eb.Width && my >= eb.Y && my < eb.Y+eb.Height {
			e.sidebarFocus = false
			e.editorView.HandleMouseClick(mx, my)
			e.syncTabFromView()
		}
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

	// Render overlays (search bar, input bar, palette) on top
	if e.searchBar.IsVisible() {
		if r, ok := regions["editor"]; ok {
			// VS Code style: right-aligned small overlay at top of editor
			barWidth := 40
			if barWidth > r.Width {
				barWidth = r.Width
			}
			barHeight := 1
			if e.searchBar.ReplaceMode() {
				barHeight = 2
			}
			if barHeight > r.Height {
				barHeight = r.Height
			}
			barX := r.X + r.Width - barWidth
			e.searchBar.SetBounds(types.Rect{X: barX, Y: r.Y, Width: barWidth, Height: barHeight})
			e.searchBar.Render(e.screen)
		}
	}
	if e.inputBar.IsVisible() {
		if r, ok := regions["editor"]; ok {
			// Right-aligned small overlay at top of editor
			barWidth := 30
			if barWidth > r.Width {
				barWidth = r.Width
			}
			barX := r.X + r.Width - barWidth
			e.inputBar.SetBounds(types.Rect{X: barX, Y: r.Y, Width: barWidth, Height: 1})
			e.inputBar.Render(e.screen)
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
	e.keymap.Bind("ctrl+g", "editor.goToLine", "")
	e.keymap.Bind("ctrl+shift+f", "search.findInFiles", "")
	e.keymap.Bind("alt+f", "search.findInFiles", "") // fallback for terminals that can't send Ctrl+Shift+F
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
			if err := tab.Buffer.Save(); err != nil {
				return err
			}
			// Notify LSP of save
			if client := e.lspManager.GetClient(tab.Language); client != nil && tab.Buffer.Path() != "" {
				lsp.DidSave(client, lsp.FileURIFromPath(tab.Buffer.Path()), tab.Buffer.Text())
			}
		}
	case "file.close":
		e.closeCurrentTab()
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
	case "search.findInFiles":
		e.showProjectSearch()
	case "editor.goToLine":
		e.inputBar.Show("Go to line: ")
	case "lsp.goToDefinition":
		e.lspGoToDefinition()
	case "lsp.findReferences":
		e.lspFindReferences()
	case "lsp.autocomplete":
		e.lspAutocomplete()
	case "lsp.hover":
		e.lspHover()
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
		e.editorView.ClearSearchHighlights()
		return
	}
	tab := e.tabs.Active()
	if tab == nil {
		return
	}
	s, err := search.NewInFileSearch(query, false, false)
	if err != nil {
		e.searchBar.SetMatches(nil)
		e.editorView.ClearSearchHighlights()
		return
	}
	matches := s.FindAll(tab.Buffer.Text())
	e.searchBar.SetMatches(matches)

	// Convert matches to rune-based search highlights
	var highlights []view.SearchHighlight
	for _, m := range matches {
		runeCol := byteColToRuneCol(tab.Buffer.Line(m.Line), m.Col)
		runeLen := byteColToRuneCol(tab.Buffer.Line(m.Line)[m.Col:], m.Length)
		highlights = append(highlights, view.SearchHighlight{
			Line:   m.Line,
			Col:    runeCol,
			Length: runeLen,
		})
	}
	e.editorView.SetSearchHighlights(highlights)

	// Jump to first match after current cursor
	if len(matches) > 0 {
		cursor := e.editorView.CursorPosition()
		for _, m := range matches {
			runeCol := byteColToRuneCol(tab.Buffer.Line(m.Line), m.Col)
			if m.Line > cursor.Line || (m.Line == cursor.Line && runeCol > cursor.Col) {
				e.editorView.SetCursorPosition(types.Position{Line: m.Line, Col: runeCol})
				e.syncTabFromView()
				return
			}
		}
		// Wrap: jump to first match
		runeCol := byteColToRuneCol(tab.Buffer.Line(matches[0].Line), matches[0].Col)
		e.editorView.SetCursorPosition(types.Position{Line: matches[0].Line, Col: runeCol})
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

// byteColToRuneCol converts a byte offset to a rune count within a string.
func byteColToRuneCol(s string, byteCol int) int {
	if byteCol >= len(s) {
		byteCol = len(s)
	}
	return len([]rune(s[:byteCol]))
}

// closeCurrentTab closes the active tab with unsaved changes check.
func (e *Editor) closeCurrentTab() {
	tab := e.tabs.Active()
	if tab == nil {
		return
	}
	if tab.Buffer.IsDirty() {
		e.statusBar.SetMessage("Unsaved changes! Save first or press Ctrl+Q to quit.")
		return
	}
	// Notify LSP of file close
	if e.lspManager.IsRunning(tab.Language) {
		client := e.lspManager.GetClient(tab.Language)
		if client != nil && tab.Buffer.Path() != "" {
			lsp.DidClose(client, lsp.FileURIFromPath(tab.Buffer.Path()))
		}
	}
	idx := e.tabs.ActiveIndex()
	e.tabs.Close(idx)
	if e.tabs.Count() == 0 {
		e.OpenEmpty()
	}
	e.syncViewToTab()
}

// showProjectSearch opens the project-wide search input.
func (e *Editor) showProjectSearch() {
	e.inputBar.SetOnSubmit(func(query string) {
		query = strings.TrimSpace(query)
		if query == "" {
			return
		}
		ps := search.NewProjectSearch(e.projectRoot, []string{".git", "node_modules", "vendor"}, true)
		results, err := ps.Search(query, false)
		if err != nil {
			e.statusBar.SetMessage(fmt.Sprintf("Search error: %v", err))
			return
		}
		e.projectSearchResults = results
		e.projectSearchQuery = query
		// Show results in the bottom panel "Output" tab
		var lines []string
		for _, r := range results {
			rel, _ := filepath.Rel(e.projectRoot, r.File)
			if rel == "" {
				rel = r.File
			}
			lines = append(lines, fmt.Sprintf("%s:%d:%d  %s", rel, r.Line, r.Col, r.Text))
		}
		if len(lines) == 0 {
			lines = append(lines, fmt.Sprintf("No results for '%s'", query))
		} else {
			lines = append([]string{fmt.Sprintf("Found %d results for '%s':", len(results), query)}, lines...)
		}
		e.panel.SetContent(1, lines) // "Output" tab
		e.panel.SetActiveTab(1)
		e.layout.SetPanelVisible(true)
		e.statusBar.SetMessage(fmt.Sprintf("Found %d results", len(results)))

		// If there are results, jump to first one
		if len(results) > 0 {
			first := results[0]
			e.OpenFile(first.File)
			if e.editorView != nil {
				e.editorView.SetCursorPosition(types.Position{Line: first.Line - 1, Col: first.Col - 1})
				e.syncTabFromView()
				e.highlightProjectSearchInFile()
			}
		}
	})
	e.inputBar.SetOnCancel(func() {
		// Restore go-to-line behavior after project search
		e.inputBar.SetOnSubmit(func(value string) {
			lineNum, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || lineNum < 1 {
				e.statusBar.SetMessage("Invalid line number")
				return
			}
			if e.editorView != nil {
				e.editorView.SetCursorPosition(types.Position{Line: lineNum - 1, Col: 0})
				e.syncTabFromView()
			}
		})
	})
	e.inputBar.Show("Search in files: ")
}

// handlePanelLineClick handles clicking a line in the bottom panel.
func (e *Editor) handlePanelLineClick(tabIdx int, lineIdx int) {
	switch tabIdx {
	case 0: // Problems tab — format: "file:line:col: severity: message"
		e.navigatePanelLine(tabIdx, lineIdx)
	case 1: // Output tab — format: "relpath:line:col  text" (project search results)
		e.navigatePanelLine(tabIdx, lineIdx)
	}
}

// navigatePanelLine parses a panel content line and navigates to the file/position.
func (e *Editor) navigatePanelLine(tabIdx int, lineIdx int) {
	// Get panel content
	if tabIdx < 0 || tabIdx > 2 {
		return
	}
	// Panel content lines have format: "path:line:col ..."
	// Try to parse from projectSearchResults if available (Output tab)
	if tabIdx == 1 && len(e.projectSearchResults) > 0 {
		// lineIdx 0 is the header "Found N results for '...'"
		resultIdx := lineIdx - 1
		if resultIdx >= 0 && resultIdx < len(e.projectSearchResults) {
			r := e.projectSearchResults[resultIdx]
			e.OpenFile(r.File)
			if e.editorView != nil {
				e.editorView.SetCursorPosition(types.Position{Line: r.Line - 1, Col: r.Col - 1})
				e.syncTabFromView()
				// Highlight all matches of the search query in this file
				e.highlightProjectSearchInFile()
			}
			return
		}
	}

	// Generic fallback: parse "path:line:col" from the line text
	// This handles problems tab and other formats
	if tabIdx == 0 {
		allDiags := e.lspHandler.GetAllDiagnostics()
		idx := 0
		for uri, diags := range allDiags {
			for _, d := range diags {
				if idx == lineIdx {
					path := lsp.PathFromFileURI(uri)
					e.OpenFile(path)
					if e.editorView != nil {
						e.editorView.SetCursorPosition(types.Position{Line: d.Range.Start.Line, Col: d.Range.Start.Character})
						e.syncTabFromView()
					}
					return
				}
				idx++
			}
		}
	}
}

// highlightProjectSearchInFile highlights all occurrences of the project search
// query in the currently open file.
func (e *Editor) highlightProjectSearchInFile() {
	if e.projectSearchQuery == "" || e.editorView == nil {
		return
	}
	tab := e.tabs.Active()
	if tab == nil {
		return
	}
	s, err := search.NewInFileSearch(e.projectSearchQuery, false, false)
	if err != nil {
		return
	}
	matches := s.FindAll(tab.Buffer.Text())
	var highlights []view.SearchHighlight
	for _, m := range matches {
		runeCol := byteColToRuneCol(tab.Buffer.Line(m.Line), m.Col)
		runeLen := byteColToRuneCol(tab.Buffer.Line(m.Line)[m.Col:], m.Length)
		highlights = append(highlights, view.SearchHighlight{
			Line:   m.Line,
			Col:    runeCol,
			Length: runeLen,
		})
	}
	e.editorView.SetSearchHighlights(highlights)
}

// --- LSP integration ---

// ensureLSP starts the LSP server for the active tab's language if needed.
func (e *Editor) ensureLSP() {
	tab := e.tabs.Active()
	if tab == nil {
		return
	}
	lang := tab.Language
	if lang == "" || lang == "text" {
		return
	}
	if e.lspManager.IsRunning(lang) {
		return
	}
	rootURI := lsp.FileURIFromPath(e.projectRoot)
	if err := e.lspManager.StartServer(lang, rootURI); err != nil {
		// LSP not available — silently ignore
		return
	}
	// Notify didOpen for all open tabs of this language
	for _, t := range e.tabs.All() {
		if t.Language == lang && t.Buffer.Path() != "" {
			client := e.lspManager.GetClient(lang)
			if client != nil {
				lsp.DidOpen(client, lsp.FileURIFromPath(t.Buffer.Path()), lang, t.Buffer.Text())
			}
		}
	}
}

// lspNotifyOpen notifies LSP that a file was opened.
func (e *Editor) lspNotifyOpen(tab *TabInfo) {
	if tab.Buffer.Path() == "" {
		return
	}
	client := e.lspManager.GetClient(tab.Language)
	if client == nil {
		return
	}
	lsp.DidOpen(client, lsp.FileURIFromPath(tab.Buffer.Path()), tab.Language, tab.Buffer.Text())
}

func (e *Editor) lspGoToDefinition() {
	tab := e.tabs.Active()
	if tab == nil || e.editorView == nil || tab.Buffer.Path() == "" {
		return
	}
	client := e.lspManager.GetClient(tab.Language)
	if client == nil {
		e.statusBar.SetMessage("LSP not available for " + tab.Language)
		return
	}
	cursor := e.editorView.CursorPosition()
	resp, err := lsp.RequestDefinition(client, lsp.FileURIFromPath(tab.Buffer.Path()), cursor.Line, cursor.Col)
	if err != nil {
		e.statusBar.SetMessage(fmt.Sprintf("LSP error: %v", err))
		return
	}
	if resp.Error != nil {
		e.statusBar.SetMessage(fmt.Sprintf("LSP: %s", resp.Error.Message))
		return
	}
	// Parse response as Location or []Location
	locations := parseLSPLocations(resp)
	if len(locations) == 0 {
		e.statusBar.SetMessage("No definition found")
		return
	}
	loc := locations[0]
	path := lsp.PathFromFileURI(loc.URI)
	e.OpenFile(path)
	if e.editorView != nil {
		e.editorView.SetCursorPosition(types.Position{Line: loc.Range.Start.Line, Col: loc.Range.Start.Character})
		e.syncTabFromView()
	}
}

func (e *Editor) lspFindReferences() {
	tab := e.tabs.Active()
	if tab == nil || e.editorView == nil || tab.Buffer.Path() == "" {
		return
	}
	client := e.lspManager.GetClient(tab.Language)
	if client == nil {
		e.statusBar.SetMessage("LSP not available for " + tab.Language)
		return
	}
	cursor := e.editorView.CursorPosition()
	resp, err := lsp.RequestReferences(client, lsp.FileURIFromPath(tab.Buffer.Path()), cursor.Line, cursor.Col)
	if err != nil {
		e.statusBar.SetMessage(fmt.Sprintf("LSP error: %v", err))
		return
	}
	if resp.Error != nil {
		e.statusBar.SetMessage(fmt.Sprintf("LSP: %s", resp.Error.Message))
		return
	}
	locations := parseLSPLocations(resp)
	if len(locations) == 0 {
		e.statusBar.SetMessage("No references found")
		return
	}
	// Show references in bottom panel
	var lines []string
	lines = append(lines, fmt.Sprintf("Found %d references:", len(locations)))
	for _, loc := range locations {
		path := lsp.PathFromFileURI(loc.URI)
		rel, _ := filepath.Rel(e.projectRoot, path)
		if rel == "" {
			rel = path
		}
		lines = append(lines, fmt.Sprintf("  %s:%d:%d", rel, loc.Range.Start.Line+1, loc.Range.Start.Character+1))
	}
	e.panel.SetContent(1, lines)
	e.panel.SetActiveTab(1)
	e.layout.SetPanelVisible(true)
	e.statusBar.SetMessage(fmt.Sprintf("Found %d references", len(locations)))
}

func (e *Editor) lspAutocomplete() {
	tab := e.tabs.Active()
	if tab == nil || e.editorView == nil || tab.Buffer.Path() == "" {
		return
	}
	client := e.lspManager.GetClient(tab.Language)
	if client == nil {
		return
	}
	cursor := e.editorView.CursorPosition()
	resp, err := lsp.RequestCompletion(client, lsp.FileURIFromPath(tab.Buffer.Path()), cursor.Line, cursor.Col)
	if err != nil {
		return
	}
	items, err := lsp.ParseCompletionResponse(resp)
	if err != nil || len(items) == 0 {
		e.statusBar.SetMessage("No completions")
		return
	}
	// Show completions in palette for selection
	paletteItems := make([]view.PaletteItem, len(items))
	for i, item := range items {
		text := item.InsertText
		if text == "" {
			text = item.Label
		}
		paletteItems[i] = view.PaletteItem{
			Label:       item.Label,
			Description: item.Detail,
			Command:     "__insert:" + text,
		}
	}
	e.palette.SetItems(paletteItems)
	e.palette.SetOnSelect(func(item view.PaletteItem) {
		if text, ok := strings.CutPrefix(item.Command, "__insert:"); ok {
			if e.editorView != nil {
				for _, ch := range text {
					e.editorView.InsertChar(ch)
				}
				e.syncTabFromView()
			}
		} else {
			e.ExecuteCommand(item.Command)
		}
		// Restore palette to command list
		e.updatePaletteItems()
		e.palette.SetOnSelect(func(item view.PaletteItem) {
			e.ExecuteCommand(item.Command)
		})
	})
	e.palette.Show()
}

func (e *Editor) lspHover() {
	tab := e.tabs.Active()
	if tab == nil || e.editorView == nil || tab.Buffer.Path() == "" {
		return
	}
	client := e.lspManager.GetClient(tab.Language)
	if client == nil {
		e.statusBar.SetMessage("LSP not available for " + tab.Language)
		return
	}
	cursor := e.editorView.CursorPosition()
	resp, err := lsp.RequestHover(client, lsp.FileURIFromPath(tab.Buffer.Path()), cursor.Line, cursor.Col)
	if err != nil {
		e.statusBar.SetMessage(fmt.Sprintf("LSP error: %v", err))
		return
	}
	if resp.Error != nil || resp.Result == nil {
		e.statusBar.SetMessage("No hover info")
		return
	}
	// Parse hover result
	data, _ := json.Marshal(resp.Result)
	var hover lsp.Hover
	json.Unmarshal(data, &hover)

	// Extract hover text
	hoverText := ""
	switch v := hover.Contents.(type) {
	case string:
		hoverText = v
	case map[string]interface{}:
		if val, ok := v["value"]; ok {
			hoverText = fmt.Sprintf("%v", val)
		}
	}
	if hoverText == "" {
		e.statusBar.SetMessage("No hover info")
		return
	}
	// Show hover in status bar (first line only)
	lines := strings.SplitN(hoverText, "\n", 2)
	e.statusBar.SetMessage(lines[0])
}

// parseLSPLocations parses a definition/references response into locations.
func parseLSPLocations(resp *lsp.Response) []lsp.Location {
	if resp.Result == nil {
		return nil
	}
	data, _ := json.Marshal(resp.Result)

	// Try as single Location
	var loc lsp.Location
	if err := json.Unmarshal(data, &loc); err == nil && loc.URI != "" {
		return []lsp.Location{loc}
	}

	// Try as []Location
	var locs []lsp.Location
	if err := json.Unmarshal(data, &locs); err == nil {
		return locs
	}

	return nil
}
