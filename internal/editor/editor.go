package editor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/seoji/ted/internal/buffer"
	"github.com/seoji/ted/internal/config"
	"github.com/seoji/ted/internal/git"
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
	autocomplete *view.AutocompletePopup
	tooltip      *view.Tooltip
	running      bool
	sidebarFocus bool // true when sidebar has keyboard focus
	panelFocus   bool // true when bottom panel has keyboard focus
	quitPending  bool // true when quit requested with unsaved changes
	lspManager   *lsp.ServerManager
	lspHandler   *lsp.NotificationHandler
	projectRoot  string // root directory for project search and LSP
	projectSearchResults []search.FileMatch // cached project search results
	projectSearchQuery   string              // the query used for project search
	recentFiles  *RecentFiles
	lastHoverLine int // track last hover position to avoid duplicate requests
	lastHoverCol  int
	pythonEnv     *PythonEnv // current Python environment
	splitManager    *SplitManager
	rightEditorView *view.EditorView // nil when not split
	diffTracker      *git.DiffTracker
	graphView        *view.GraphView
	commitDetailView *view.CommitDetailView
	pasteActive      bool
}

// New creates a new Editor instance.
func New(cfg *config.Config, theme *syntax.Theme) *Editor {
	// Configure LSP servers
	lspConfigs := map[string]lsp.ServerConfig{
		"go":     {Command: "gopls", Args: []string{"serve"}},
		"python": {Command: "pylsp", Args: []string{}},
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

	e.splitManager = NewSplitManager()
	e.layout.SetSidebarWidth(cfg.Sidebar.Width)
	e.layout.SetSidebarVisible(true) // sidebar always visible
	e.layout.SetPanelHeight(cfg.Panel.Height)
	e.layout.SetPanelVisible(cfg.Panel.Visible)

	e.statusBar = view.NewStatusBar(theme)
	e.tabBar = view.NewTabBar(theme)
	e.sidebar = view.NewSidebar(theme)
	e.panel = view.NewBottomPanel(theme)
	e.palette = view.NewCommandPalette(theme)
	e.searchBar = view.NewSearchBar(theme)
	e.inputBar = view.NewInputBar(theme)
	e.autocomplete = view.NewAutocompletePopup(theme)
	e.tooltip = view.NewTooltip(theme)
	e.recentFiles = LoadRecentFiles()

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
	e.palette.SetOnFileOpen(func(path string) {
		e.OpenFile(path)
	})
	e.palette.SetOnBufferOpen(func(path string) {
		// Find the tab with this path and switch to it
		idx := e.tabs.FindByPath(path)
		if idx >= 0 {
			e.tabs.SetActive(idx)
			e.syncViewToTab()
		}
	})
	e.palette.SetOnGoToLine(func(line int) {
		if e.editorView != nil {
			e.editorView.SetCursorPosition(types.Position{Line: line - 1, Col: 0})
			e.syncTabFromView()
		}
	})
	e.palette.SetOnDismiss(func() {})

	// Wire autocomplete callbacks
	e.autocomplete.SetOnSelect(func(item view.CompletionItem) {
		if e.editorView != nil {
			text := item.InsertText
			if text == "" {
				text = item.Label
			}
			for _, ch := range text {
				e.editorView.InsertChar(ch)
			}
			e.syncTabFromView()
		}
	})
	e.autocomplete.SetOnDismiss(func() {})

	// Wire sidebar callback
	e.sidebar.SetOnFileOpen(func(path string) {
		e.OpenFile(path)
		e.sidebarFocus = false
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
	e.searchBar.SetOnNavigate(func(m search.Match) {
		if e.editorView == nil {
			return
		}
		tab := e.tabs.Active()
		if tab == nil {
			return
		}
		runeCol := byteColToRuneCol(tab.Buffer.Line(m.Line), m.Col)
		e.editorView.SetCursorPosition(types.Position{Line: m.Line, Col: runeCol})
		e.syncTabFromView()
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

	// Detect Python environment
	if cwd, err := os.Getwd(); err == nil {
		e.pythonEnv = DetectPythonEnv(cwd)
	}

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

	// Track recent files
	e.recentFiles.Add(path)

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
	e.diffTracker, _ = git.NewDiffTracker(e.projectRoot)
	e.layout.SetSidebarVisible(true)
	e.sidebarFocus = true // start with sidebar focus
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

	// Enable bracketed paste so pasted text arrives as a single block
	screen.EnablePaste()

	// Only open empty buffer if no directory was opened via sidebar
	if e.tabs.Count() == 0 && !e.layout.SidebarVisible() {
		e.OpenEmpty()
	}

	// Set sidebar root and project root to current working directory
	if cwd, err := os.Getwd(); err == nil {
		e.sidebar.SetRoot(cwd)
		if e.projectRoot == "" {
			e.projectRoot = cwd
		}
		if e.diffTracker == nil {
			e.diffTracker, _ = git.NewDiffTracker(e.projectRoot)
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
		case *tcell.EventPaste:
			if tev.Start() {
				e.pasteActive = true
			} else if tev.End() {
				e.pasteActive = false
				// WT sends garbled encoding via bracketed paste — ignore it.
				// Read clipboard directly with proper UTF-8 encoding.
				if text := e.readSystemClipboard(); text != "" && e.editorView != nil {
					e.editorView.InsertText(text)
					e.syncTabFromView()
				}
				e.render()
			}
		case *tcell.EventKey:
			if e.pasteActive {
				continue // discard garbled paste key events
			}
			e.handleKeyEvent(tev)
			e.render()
		case *tcell.EventMouse:
			e.handleMouseEvent(tev)
			e.render()
		case *tcell.EventInterrupt:
			// Triggered by async LSP operations (autocomplete, hover)
			_ = tev
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
	// Tooltip gets dismissed on any key
	if e.tooltip.IsVisible() {
		e.tooltip.Hide()
	}

	// Autocomplete popup gets priority when visible
	if e.autocomplete.IsVisible() {
		if e.autocomplete.HandleEvent(ev) {
			return
		}
	}

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
		// Escape or Alt+Right returns focus to editor
		if ev.Key() == tcell.KeyEscape {
			e.sidebarFocus = false
			return
		}
		if ev.Key() == tcell.KeyRight && ev.Modifiers()&tcell.ModAlt != 0 {
			e.sidebarFocus = false
			return
		}
		e.sidebar.SetFocused(true)
		e.sidebar.HandleEvent(ev)
		e.sidebar.SetFocused(false)
		return
	}

	// Panel keyboard navigation when focused
	if e.panelFocus && e.layout.PanelVisible() {
		if ev.Key() == tcell.KeyEscape {
			e.panelFocus = false
			e.projectSearchQuery = ""
			e.projectSearchResults = nil
			if e.editorView != nil {
				e.editorView.ClearSearchHighlights()
			}
			if e.rightEditorView != nil {
				e.rightEditorView.ClearSearchHighlights()
			}
			e.layout.SetPanelVisible(false)
			return
		}
		e.panel.HandleEvent(ev)
		return
	}

	// Escape in editor: clear search highlights and close panel
	if ev.Key() == tcell.KeyEscape {
		if e.projectSearchQuery != "" || e.layout.PanelVisible() {
			e.projectSearchQuery = ""
			e.projectSearchResults = nil
			if e.editorView != nil {
				e.editorView.ClearSearchHighlights()
			}
			if e.rightEditorView != nil {
				e.rightEditorView.ClearSearchHighlights()
			}
			e.layout.SetPanelVisible(false)
			return
		}
	}

	// Graph tab event handling
	tab := e.tabs.Active()
	if tab != nil && tab.Kind == TabKindGraph && e.graphView != nil {
		if e.graphView.HandleEvent(ev) {
			return
		}
	}

	// Pass to active editor view for text input
	av := e.activeEditorView()
	if av != nil {
		av.HandleEvent(ev)
		if e.splitManager.IsSplit() && e.splitManager.ActivePane() == PaneRight {
			e.syncRightTab()
		} else {
			e.syncTabFromView()
		}
		// Clear project search highlights on any edit action
		if e.projectSearchQuery != "" {
			k := ev.Key()
			if k == tcell.KeyRune || k == tcell.KeyBackspace || k == tcell.KeyBackspace2 || k == tcell.KeyDelete || k == tcell.KeyEnter {
				e.projectSearchQuery = ""
				av.ClearSearchHighlights()
			}
		}
		// Auto-trigger autocomplete after typing '.' or '::'
		if ev.Key() == tcell.KeyRune {
			ch := ev.Rune()
			if ch == '.' || ch == ':' {
				go e.lspAutocompleteAsync()
			}
		}
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
			// Skip header row click
			if my == sb.Y {
				return
			}
			// Calculate which entry was clicked (subtract 1 for header)
			row := my - sb.Y - 1 + e.sidebar.ScrollY()
			e.sidebar.SelectIndex(row)
			return
		}
		// Clicked outside sidebar — return focus to editor
		e.sidebarFocus = false
	}

	// Panel click — navigate to result and set panel focus
	if e.layout.PanelVisible() {
		pb := e.panel.Bounds()
		if mx >= pb.X && mx < pb.X+pb.Width && my >= pb.Y && my < pb.Y+pb.Height {
			e.panelFocus = true
			e.sidebarFocus = false
			e.panel.HandleEvent(ev)
			return
		}
	}

	// Graph view mouse events
	tab := e.tabs.Active()
	if tab != nil && tab.Kind == TabKindGraph && e.graphView != nil {
		if e.graphView.HandleEvent(ev) {
			return
		}
		if e.commitDetailView != nil {
			if e.commitDetailView.HandleEvent(ev) {
				return
			}
		}
	}

	// Editor area — click, scroll wheel, or hover
	if e.editorView != nil {
		eb := e.editorView.Bounds()
		if mx >= eb.X && mx < eb.X+eb.Width && my >= eb.Y && my < eb.Y+eb.Height {
			if btn == tcell.Button1 {
				e.sidebarFocus = false
				e.panelFocus = false
				e.tooltip.Hide()
				e.editorView.HandleMouseClick(mx, my)
				e.syncTabFromView()
			} else if btn == tcell.WheelUp {
				e.tooltip.Hide()
				e.editorView.ScrollUp(3)
			} else if btn == tcell.WheelDown {
				e.tooltip.Hide()
				e.editorView.ScrollDown(3)
			} else if btn == tcell.ButtonNone {
				// Mouse hover — trigger LSP hover
				e.handleMouseHover(mx, my)
			}
		} else {
			// Mouse moved outside editor area
			e.tooltip.Hide()
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
			title := ""
			if t.Kind == TabKindGraph {
				title = "⎇ Git Graph"
			} else if t.Buffer.Path() != "" {
				title = filepath.Base(t.Buffer.Path())
			}
			tabs[i] = view.Tab{
				Title:    title,
				FilePath: t.Buffer.Path(),
				Dirty:    t.Kind == TabKindFile && t.Buffer.IsDirty(),
			}
		}
		e.tabBar.SetTabs(tabs, e.tabs.ActiveIndex())
		e.tabBar.Render(e.screen)
	}

	// Render sidebar
	if r, ok := regions["sidebar"]; ok {
		e.sidebar.SetBounds(r)
		e.sidebar.SetFocused(e.sidebarFocus)
		e.sidebar.Render(e.screen)
	}

	// Render sidebar separator
	if r, ok := regions["separator"]; ok {
		sepStyle := e.theme.UIStyle("sidebar").Foreground(tcell.ColorDarkGray)
		for y := r.Y; y < r.Y+r.Height; y++ {
			e.screen.SetContent(r.X, y, '│', nil, sepStyle)
		}
	}

	// Render editor view(s)
	if e.splitManager.IsSplit() {
		if r, ok := regions["editor.left"]; ok && e.editorView != nil {
			e.editorView.SetBounds(r)
			e.editorView.SetFocused(e.splitManager.ActivePane() == PaneLeft)
			e.editorView.Render(e.screen)
		}
		if r, ok := regions["editor.separator"]; ok {
			// Use a brighter color for the separator when split is active
			color := tcell.ColorGray
			if e.splitManager.IsSplit() {
				color = tcell.ColorBlue
			}
			sepStyle := e.theme.UIStyle("panel").Foreground(color)
			for y := r.Y; y < r.Y+r.Height; y++ {
				e.screen.SetContent(r.X, y, '│', nil, sepStyle)
			}
		}
		if r, ok := regions["editor.right"]; ok && e.rightEditorView != nil {
			e.rightEditorView.SetBounds(r)
			e.rightEditorView.SetFocused(e.splitManager.ActivePane() == PaneRight)
			e.rightEditorView.Render(e.screen)
		}
	} else {
		if r, ok := regions["editor"]; ok {
			tab := e.tabs.Active()
			if tab != nil && tab.Kind == TabKindGraph && e.graphView != nil {
				graphHeight := r.Height * 7 / 10
				detailHeight := r.Height - graphHeight

				graphRect := types.Rect{X: r.X, Y: r.Y, Width: r.Width, Height: graphHeight}
				detailRect := types.Rect{X: r.X, Y: r.Y + graphHeight, Width: r.Width, Height: detailHeight}

				e.graphView.SetBounds(graphRect)
				e.graphView.SetFocused(true)
				e.graphView.Render(e.screen)

				e.commitDetailView.SetBounds(detailRect)
				e.commitDetailView.Render(e.screen)
			} else if e.editorView != nil {
				e.editorView.SetBounds(r)
				e.editorView.SetFocused(true)
				e.editorView.Render(e.screen)
			}
		}
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
			// Show Python info for Python files
			if tab.Language == "python" && e.pythonEnv != nil {
				info := "Python " + e.pythonEnv.Version
				if e.pythonEnv.VenvName != "" {
					info += " (" + e.pythonEnv.VenvName + ")"
				}
				e.statusBar.SetPythonInfo(info)
			} else {
				e.statusBar.SetPythonInfo("")
			}
		}
		e.statusBar.Render(e.screen)
	}

	// Helper to get active editor region
	activeEditorRegion := func() (types.Rect, bool) {
		if e.splitManager.IsSplit() {
			if e.splitManager.ActivePane() == PaneRight {
				r, ok := regions["editor.right"]
				return r, ok
			}
			r, ok := regions["editor.left"]
			return r, ok
		}
		r, ok := regions["editor"]
		return r, ok
	}

	// Render overlays (search bar, input bar, palette) on top
	if e.searchBar.IsVisible() {
		if r, ok := activeEditorRegion(); ok {
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
		if r, ok := activeEditorRegion(); ok {
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
	if e.autocomplete.IsVisible() {
		e.autocomplete.Render(e.screen)
	}
	if e.tooltip.IsVisible() {
		e.tooltip.Render(e.screen)
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

	if tab.Kind == TabKindGraph {
		e.editorView = nil
		return
	}

	e.editorView = view.NewEditorView(tab.Buffer, e.theme)
	e.editorView.SetLanguage(tab.Language)
	e.editorView.SetCursorPosition(tab.Cursor)
	e.editorView.SetScrollY(tab.ScrollY)
	e.updateGutterMarkers()
}

func (e *Editor) syncTabFromView() {
	tab := e.tabs.Active()
	if tab == nil || e.editorView == nil {
		return
	}
	tab.Cursor = e.editorView.CursorPosition()
	tab.ScrollY, tab.ScrollX = e.editorView.ScrollPosition()
}

func (e *Editor) activeEditorView() *view.EditorView {
	if e.splitManager.IsSplit() && e.splitManager.ActivePane() == PaneRight {
		return e.rightEditorView
	}
	return e.editorView
}

func (e *Editor) syncRightView() {
	ps := e.splitManager.RightPane()
	if ps == nil {
		e.rightEditorView = nil
		return
	}
	e.rightEditorView = view.NewEditorView(ps.Buffer, e.theme)
	e.rightEditorView.SetLanguage(ps.Language)
	e.rightEditorView.SetCursorPosition(ps.Cursor)
	e.rightEditorView.SetScrollY(ps.ScrollY)
}

func (e *Editor) syncRightTab() {
	ps := e.splitManager.RightPane()
	if ps == nil || e.rightEditorView == nil {
		return
	}
	ps.Cursor = e.rightEditorView.CursorPosition()
	ps.ScrollY, ps.ScrollX = e.rightEditorView.ScrollPosition()
}

func (e *Editor) copyToSystemClipboard(text string) {
	// WSL: base64-encode UTF-8 text to avoid CP949 encoding issues with PowerShell stdin
	b64 := base64.StdEncoding.EncodeToString([]byte(text))
	psCmd := "[System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String('" + b64 + "')) | Set-Clipboard"
	if err := exec.Command("powershell.exe", "-NoProfile", "-Command", psCmd).Run(); err == nil {
		return
	}
	// Fallback: xclip / xsel (native UTF-8)
	for _, c := range []struct {
		name string
		args []string
	}{
		{"xclip", []string{"-selection", "clipboard"}},
		{"xsel", []string{"--clipboard", "--input"}},
	} {
		cmd := exec.Command(c.name, c.args...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return
		}
	}
}

func (e *Editor) readSystemClipboard() string {
	// WSL: read clipboard via PowerShell with explicit UTF-8 output encoding
	psCmd := "[Console]::OutputEncoding=[System.Text.Encoding]::UTF8; Get-Clipboard"
	if out, err := exec.Command("powershell.exe", "-NoProfile", "-Command", psCmd).Output(); err == nil {
		return strings.TrimRight(string(out), "\r\n")
	}
	// Fallback: xclip / xsel
	for _, c := range []struct {
		name string
		args []string
	}{
		{"xclip", []string{"-selection", "clipboard", "-o"}},
		{"xsel", []string{"--clipboard", "--output"}},
	} {
		if out, err := exec.Command(c.name, c.args...).Output(); err == nil {
			return strings.TrimRight(string(out), "\r\n")
		}
	}
	return ""
}

func (e *Editor) updateGutterMarkers() {
	if e.diffTracker == nil || e.editorView == nil {
		return
	}
	tab := e.tabs.Active()
	if tab == nil || tab.Buffer.Path() == "" {
		return
	}
	markers, err := e.diffTracker.ComputeMarkers(tab.Buffer.Path())
	if err != nil {
		return
	}
	e.editorView.SetGutterMarkers(markers)
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
	// Command items
	cmds := e.commands.Commands()
	cmdItems := make([]view.PaletteItem, len(cmds))
	for i, cmd := range cmds {
		// Get keybinding for this command
		keybinding := ""
		bindings := e.keymap.BindingsForCommand(cmd.Name)
		if len(bindings) > 0 {
			keybinding = bindings[0] // use first binding
		}

		cmdItems[i] = view.PaletteItem{
			Label:       cmd.Name,
			Description: cmd.Description,
			Command:     cmd.Name,
			Keybinding:  keybinding,
		}
	}
	e.palette.SetItems(cmdItems)

	// File items - scan project root
	e.updatePaletteFileItems()
}

func (e *Editor) updatePaletteFileItems() {
	if e.projectRoot == "" {
		return
	}

	var fileItems []view.PaletteItem

	// Add recent files first (marked)
	seen := map[string]bool{}
	for _, f := range e.recentFiles.Files {
		rel, err := filepath.Rel(e.projectRoot, f)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		// Check file still exists
		if _, err := os.Stat(f); err != nil {
			continue
		}
		fileItems = append(fileItems, view.PaletteItem{
			Label:       rel,
			Description: "(recent)",
			FilePath:    f,
		})
		seen[f] = true
	}

	// Walk project directory for files
	filepath.Walk(e.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		name := info.Name()
		// Skip hidden dirs and common ignores
		if info.IsDir() {
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "__pycache__" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") {
			return nil
		}
		if seen[path] {
			return nil
		}
		rel, _ := filepath.Rel(e.projectRoot, path)
		fileItems = append(fileItems, view.PaletteItem{
			Label:    rel,
			FilePath: path,
		})
		return nil
	})

	e.palette.SetFileItems(fileItems)
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
	e.keymap.Bind("ctrl+r", "search.replace", "")
	e.keymap.Bind("ctrl+p", "palette.open", "")
	e.keymap.Bind("ctrl+b", "sidebar.focusToggle", "")
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
	e.keymap.Bind("ctrl+shift+p", "python.selectEnv", "")
	e.keymap.Bind("ctrl+shift+g", "git.graph", "")
	// Load additional keybindings from JSON config file
	configDir := config.DefaultUserConfigDir()
	// Try project-local config first, then user config
	for _, dir := range []string{".", configDir} {
		path := filepath.Join(dir, "configs", "keybindings.json")
		if _, err := os.Stat(path); err == nil {
			e.keymap.LoadFromFile(path)
		}
	}
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
			e.updateGutterMarkers()
		}
	case "file.close":
		e.closeCurrentTab()
	case "edit.undo":
		if tab := e.tabs.Active(); tab != nil {
			tab.Buffer.Undo()
			if e.editorView != nil {
				e.editorView.ReparseHighlighting()
			}
		}
	case "edit.redo":
		if tab := e.tabs.Active(); tab != nil {
			tab.Buffer.Redo()
			if e.editorView != nil {
				e.editorView.ReparseHighlighting()
			}
		}
	case "edit.copy":
		if e.editorView != nil {
			e.editorView.Copy()
			if cb := e.editorView.Clipboard(); cb != "" {
				e.copyToSystemClipboard(cb)
			}
		}
	case "edit.cut":
		if e.editorView != nil {
			e.editorView.Cut()
			if cb := e.editorView.Clipboard(); cb != "" {
				e.copyToSystemClipboard(cb)
			}
			e.syncTabFromView()
		}
	case "edit.paste":
		if e.editorView != nil {
			if text := e.readSystemClipboard(); text != "" {
				e.editorView.InsertText(text)
			} else {
				e.editorView.Paste() // fallback to internal clipboard
			}
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
	case "sidebar.focusToggle":
		e.panelFocus = false
		if e.sidebarFocus {
			e.sidebarFocus = false
		} else {
			e.sidebarFocus = true
		}
	case "panel.toggle":
		e.layout.SetPanelVisible(!e.layout.PanelVisible())
		if !e.layout.PanelVisible() && e.projectSearchQuery != "" {
			e.projectSearchQuery = ""
			if e.editorView != nil {
				e.editorView.ClearSearchHighlights()
			}
		}
	case "palette.open":
		// Populate buffer items from open tabs
		var bufferItems []view.PaletteItem
		for _, tab := range e.tabs.All() {
			if tab.Buffer != nil {
				path := tab.Buffer.Path()
				label := filepath.Base(path)
				if label == "" {
					label = "[Untitled]"
				}
				bufferItems = append(bufferItems, view.PaletteItem{
					Label:    label,
					FilePath: path,
				})
			}
		}
		e.palette.SetBufferItems(bufferItems)
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
	case "git.status":
		e.gitShowStatus()
	case "git.stageFile":
		e.gitStageCurrentFile()
	case "git.stageAll":
		e.gitStageAll()
	case "git.commit":
		e.gitCommitPrompt()
	case "git.push":
		e.gitPush()
	case "git.pull":
		e.gitPull()
	case "git.blame":
		e.gitToggleBlame()
	case "git.graph":
		e.gitOpenGraph()
	case "python.selectEnv":
		e.showPythonEnvSelector()
	case "editor.quit":
		e.tryQuit()
	case "split.vertical":
		if !e.splitManager.IsSplit() {
			tab := e.tabs.Active()
			if tab != nil {
				e.syncTabFromView()
				e.splitManager.Split(tab.Buffer, tab.Language)
				e.layout.SetSplitMode(true)
				e.syncRightView()
			}
		}
	case "split.close":
		if e.splitManager.IsSplit() {
			e.syncRightTab()
			e.splitManager.CloseSplit()
			e.layout.SetSplitMode(false)
			e.rightEditorView = nil
		}
	case "split.focus":
		if e.splitManager.IsSplit() {
			if e.splitManager.ActivePane() == PaneLeft {
				e.syncTabFromView()
			} else {
				e.syncRightTab()
			}
			e.splitManager.FocusOther()
		}
	}
	return nil
}

func (e *Editor) showPythonEnvSelector() {
	envs := ListAvailableVenvs(e.projectRoot)
	if len(envs) == 0 {
		e.statusBar.SetMessage("No Python environments found")
		return
	}

	items := make([]view.PaletteItem, len(envs))
	for i, env := range envs {
		label := env.Path
		if env.VenvName != "" {
			label = env.VenvName
		}
		desc := "Python " + env.Version
		items[i] = view.PaletteItem{
			Label:       label,
			Description: desc,
			Command:     fmt.Sprintf("__pyenv:%d", i),
		}
	}

	e.palette.SetItems(items)
	savedOnSelect := e.palette.OnSelect()
	e.palette.SetOnSelect(func(item view.PaletteItem) {
		if strings.HasPrefix(item.Command, "__pyenv:") {
			idxStr := strings.TrimPrefix(item.Command, "__pyenv:")
			if idx, err := strconv.Atoi(idxStr); err == nil && idx < len(envs) {
				e.pythonEnv = &envs[idx]
				e.statusBar.SetMessage("Python: " + envs[idx].Path)
				// Restart Python LSP with new environment
				if e.lspManager.IsRunning("python") {
					e.lspManager.StopServer("python")
				}
			}
		}
		// Restore palette
		e.updatePaletteItems()
		e.palette.SetOnSelect(savedOnSelect)
	})
	e.palette.Show()
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

	if tab.Kind == TabKindGraph {
		e.graphView = nil
		e.commitDetailView = nil
		idx := e.tabs.ActiveIndex()
		e.tabs.Close(idx)
		if e.tabs.Count() == 0 && !e.layout.SidebarVisible() {
			e.OpenEmpty()
		}
		e.syncViewToTab()
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
	tab.Buffer.Close()
	idx := e.tabs.ActiveIndex()
	e.tabs.Close(idx)
	if e.tabs.Count() == 0 && !e.layout.SidebarVisible() {
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
		e.panelFocus = true
		e.sidebarFocus = false
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
			// If file is already open, just switch tab without recreating EditorView
			if idx := e.tabs.FindByPath(r.File); idx >= 0 {
				if e.editorView != nil {
					e.syncTabFromView()
				}
				e.tabs.SetActive(idx)
				e.syncViewToTab()
			} else {
				e.OpenFile(r.File)
			}
			if e.editorView != nil {
				// Set bounds before positioning so ensureCursorVisible can scroll
				if e.editorView.Bounds().Width == 0 {
					w, h := e.screen.Size()
					regions := e.layout.Compute(w, h)
					if r, ok := regions["editor"]; ok {
						e.editorView.SetBounds(r)
					}
				}
				e.editorView.SetCursorPosition(types.Position{Line: r.Line - 1, Col: r.Col - 1})
				e.syncTabFromView()
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

	// Convert to CompletionItem and show in autocomplete popup
	completionItems := make([]view.CompletionItem, len(items))
	for i, item := range items {
		text := item.InsertText
		if text == "" {
			text = item.Label
		}
		completionItems[i] = view.CompletionItem{
			Label:      item.Label,
			Detail:     item.Detail,
			InsertText: text,
		}
	}

	// Get cursor screen position for anchoring
	scrollY, _ := e.editorView.ScrollPosition()
	bounds := e.editorView.Bounds()
	anchorY := bounds.Y + (cursor.Line - scrollY)
	anchorX := bounds.X + e.editorView.CursorScreenX()
	e.autocomplete.Show(completionItems, anchorX, anchorY)
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
	// Show hover as tooltip popup near cursor
	scrollY, _ := e.editorView.ScrollPosition()
	bounds := e.editorView.Bounds()
	anchorY := bounds.Y + (cursor.Line - scrollY)
	anchorX := bounds.X + e.editorView.CursorScreenX()
	e.tooltip.Show(hoverText, anchorX, anchorY)
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

// lspAutocompleteAsync runs autocomplete without blocking the UI.
func (e *Editor) lspAutocompleteAsync() {
	tab := e.tabs.Active()
	if tab == nil || e.editorView == nil || tab.Buffer.Path() == "" {
		return
	}
	client := e.lspManager.GetClient(tab.Language)
	if client == nil {
		return
	}

	// Notify didChange before requesting completion
	lsp.DidChange(client, lsp.FileURIFromPath(tab.Buffer.Path()), 0, tab.Buffer.Text())

	cursor := e.editorView.CursorPosition()
	resp, err := lsp.RequestCompletion(client, lsp.FileURIFromPath(tab.Buffer.Path()), cursor.Line, cursor.Col)
	if err != nil {
		return
	}
	items, err := lsp.ParseCompletionResponse(resp)
	if err != nil || len(items) == 0 {
		return
	}

	completionItems := make([]view.CompletionItem, len(items))
	for i, item := range items {
		text := item.InsertText
		if text == "" {
			text = item.Label
		}
		completionItems[i] = view.CompletionItem{
			Label:      item.Label,
			Detail:     item.Detail,
			InsertText: text,
		}
	}

	scrollY, _ := e.editorView.ScrollPosition()
	bounds := e.editorView.Bounds()
	anchorY := bounds.Y + (cursor.Line - scrollY)
	anchorX := bounds.X + e.editorView.CursorScreenX()
	e.autocomplete.Show(completionItems, anchorX, anchorY)

	// Force a re-render
	if e.screen != nil {
		e.screen.PostEvent(tcell.NewEventInterrupt(nil))
	}
}

// handleMouseHover triggers LSP hover for the word under the mouse cursor.
func (e *Editor) handleMouseHover(mx, my int) {
	if e.editorView == nil {
		return
	}
	tab := e.tabs.Active()
	if tab == nil || tab.Buffer.Path() == "" {
		return
	}
	client := e.lspManager.GetClient(tab.Language)
	if client == nil {
		return
	}

	// Convert screen position to buffer position
	eb := e.editorView.Bounds()
	line := e.editorView.ScreenYToLine(my)
	col := e.editorView.ScreenXToColForLine(mx, line)

	if line < 0 || line >= tab.Buffer.LineCount() {
		return
	}

	// Skip if same position as last hover
	if line == e.lastHoverLine && col == e.lastHoverCol {
		return
	}
	e.lastHoverLine = line
	e.lastHoverCol = col

	// Don't hover on empty space
	lineText := tab.Buffer.Line(line)
	lineRunes := []rune(lineText)
	if col >= len(lineRunes) {
		e.tooltip.Hide()
		return
	}

	// Only hover on word characters
	ch := lineRunes[col]
	if !isWordChar(ch) {
		e.tooltip.Hide()
		return
	}

	// Request hover in background
	go func() {
		resp, err := lsp.RequestHover(client, lsp.FileURIFromPath(tab.Buffer.Path()), line, col)
		if err != nil || resp.Error != nil || resp.Result == nil {
			return
		}
		data, _ := json.Marshal(resp.Result)
		var hover lsp.Hover
		json.Unmarshal(data, &hover)

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
			return
		}

		_ = eb // use editor bounds for positioning
		e.tooltip.Show(hoverText, mx, my)

		if e.screen != nil {
			e.screen.PostEvent(tcell.NewEventInterrupt(nil))
		}
	}()
}

func isWordChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}
