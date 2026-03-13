package editor

import (
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/seoji/ted/internal/buffer"
	"github.com/seoji/ted/internal/config"
	"github.com/seoji/ted/internal/input"
	"github.com/seoji/ted/internal/syntax"
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
	searchBar  *view.SearchBar
	running    bool
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

	if e.tabs.Count() == 0 {
		e.OpenEmpty()
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

	// Try keymap
	cmd, result := e.keymap.Resolve(ev, "editor")
	switch result {
	case input.ResolveMatched:
		e.ExecuteCommand(cmd)
		return
	case input.ResolvePending:
		return
	}

	// Pass to editor view for text input
	if e.editorView != nil && e.editorView.IsFocused() {
		e.editorView.HandleEvent(ev)
		e.syncTabFromView()
	}
}

func (e *Editor) handleMouseEvent(ev *tcell.EventMouse) {
	if e.tabBar != nil {
		oldIdx := e.tabs.ActiveIndex()
		if e.tabBar.HandleEvent(ev) {
			newIdx := e.tabBar.ActiveIndex()
			if newIdx != oldIdx {
				e.tabs.SetActive(newIdx)
				e.syncViewToTab()
			}
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
	e.keymap.Bind("ctrl+f", "search.find", "")
	e.keymap.Bind("ctrl+h", "search.replace", "")
	e.keymap.Bind("ctrl+p", "palette.open", "")
	e.keymap.Bind("ctrl+b", "sidebar.toggle", "")
	e.keymap.Bind("ctrl+j", "panel.toggle", "")
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
	case "tab.next":
		e.tabs.Next()
		e.syncViewToTab()
	case "tab.previous":
		e.tabs.Previous()
		e.syncViewToTab()
	case "sidebar.toggle":
		e.layout.SetSidebarVisible(!e.layout.SidebarVisible())
	case "panel.toggle":
		e.layout.SetPanelVisible(!e.layout.PanelVisible())
	case "palette.open":
		e.palette.Show()
	case "search.find":
		e.searchBar.Show(false)
	case "search.replace":
		e.searchBar.Show(true)
	case "editor.quit":
		e.Stop()
	}
	return nil
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
