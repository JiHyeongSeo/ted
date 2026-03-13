# ted

A lightweight terminal-based text editor built in Go with [tcell](https://github.com/gdamore/tcell). Inspired by VSCode, designed for developers who want a fast, keyboard-driven editor with LSP support.

## Features

### Core Editor
- **Piece Table text buffer** — O(log n) insert/delete, efficient for large files
- **Unlimited undo/redo** with save-point tracking (dirty indicator)
- **UTF-8 support** with CJK/wide character rendering
- **Mouse support** — click to position cursor, scroll wheel, tab clicks
- **Clipboard** — Ctrl+C/X/V (system clipboard via OSC 52)

### Syntax Highlighting
- **Tree-sitter based** AST-accurate highlighting (Go, Python)
- **Keyword-based fallback** for unsupported languages
- **VSCode Dark+ inspired theme** with pure black background

### LSP Integration
- **Go** (gopls) and **Python** (pylsp) language servers
- **Auto-complete** — triggers automatically on `.` and `:`
- **Hover info** — mouse hover shows type/documentation tooltip
- **Go to Definition** — F12
- **Diagnostics** — inline error/warning display

### Navigation
- **Command Palette** (Ctrl+P) — fuzzy file search, `>` for commands, `:` for go-to-line
- **In-file search** (Ctrl+F) with highlight, Enter/Shift+Enter to cycle matches
- **Project-wide search** (Ctrl+Shift+F) via ripgrep with results panel
- **Tab management** — Ctrl+Tab cycle, Alt+1~9 direct switch

### Layout
- **File explorer sidebar** (Ctrl+B) with Nerd Font icons and file-type colors
- **Bottom panel** (Ctrl+J) for search results, problems, output
- **Tab bar** with modified indicator
- **Status bar** — filename, language, cursor position, encoding, Python venv info
- **Focus navigation** — Alt+Left/Right to switch between sidebar and editor

### Python Support
- **Virtual environment detection** — auto-detect .venv, venv, conda envs
- **Environment selector** — Ctrl+Shift+P to switch Python environments
- **Status bar display** — shows active Python version and venv name

### Keybindings

| Key | Action |
|-----|--------|
| Ctrl+S | Save |
| Ctrl+Q | Quit |
| Ctrl+Z / Ctrl+Y | Undo / Redo |
| Ctrl+C / Ctrl+X / Ctrl+V | Copy / Cut / Paste |
| Ctrl+F | Find in file |
| Ctrl+H | Find and replace |
| Ctrl+Shift+F | Find in project |
| Ctrl+P | Command palette |
| Ctrl+G | Go to line |
| Ctrl+B | Toggle sidebar |
| Ctrl+J | Toggle bottom panel |
| Alt+Left / Alt+Right | Focus sidebar / editor |
| Alt+1~9 | Switch to tab N |
| F12 | Go to definition |
| Escape | Dismiss search/panel |
| PageUp / PageDown | Scroll page |

## Installation

Requires Go 1.22+ and a C compiler (for tree-sitter).

```bash
CGO_ENABLED=1 go install ./cmd/ted/
```

### Optional Dependencies
- **gopls** — Go language server (`go install golang.org/x/tools/gopls@latest`)
- **pylsp** — Python language server (`pip install python-lsp-server`)
- **ripgrep** — fast project search (`apt install ripgrep` or `brew install ripgrep`)
- **Nerd Font** — for file icons in sidebar ([nerdfonts.com](https://www.nerdfonts.com/))

## Usage

```bash
# Open a directory
ted .

# Open specific files
ted main.go utils.go

# Open a directory with a file
ted /path/to/project
```

## Architecture

```
ted/
├── cmd/ted/main.go           # Entry point
├── internal/
│   ├── buffer/               # Piece Table text buffer + undo/redo
│   ├── config/               # JSON config loading (global + project)
│   ├── editor/               # Editor core, event loop, commands
│   ├── input/                # Keybinding system (JSON-configurable)
│   ├── lsp/                  # LSP client (JSON-RPC 2.0)
│   ├── search/               # In-file + project-wide search
│   ├── syntax/               # Tree-sitter + keyword-based highlighting
│   ├── types/                # Shared types (Position, Rect)
│   └── view/                 # UI components (EditorView, Sidebar, etc.)
└── configs/default.json      # Default settings
```

## Roadmap

### M1: Core Editor — Done
- [x] Piece Table buffer with undo/redo
- [x] Tree-sitter syntax highlighting (Go, Python)
- [x] LSP integration (autocomplete, hover, definition, diagnostics)
- [x] Command palette, search, file explorer
- [x] VSCode-style keybindings
- [x] Python venv detection and selection

### M2: Git + Claude Integration — Planned
- [ ] Git gutter diff markers (added/modified/deleted)
- [ ] Git commit/push/pull UI panel
- [ ] Git blame (inline + full view)
- [ ] Claude chat panel (via Claude Code CLI)
- [ ] Claude inline code edit with diff preview
- [ ] Ctrl+Click go to definition
- [ ] Find all references (Shift+F12)

### M3: Debugging + Extensions
- [ ] DAP debugging (Go, Python)
- [ ] Plugin system
- [ ] Additional LSP features (rename, formatting)
- [ ] Split editor

## License

Personal project by seoji.
