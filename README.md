# ted

A lightweight terminal-based text editor built in Go with [tcell](https://github.com/gdamore/tcell). Inspired by VSCode, designed for developers who want a fast, keyboard-driven editor with LSP support and git integration.

## Features

### Core Editor
- **Piece Table text buffer** — O(log n) insert/delete, efficient for large files
- **Unlimited undo/redo** with save-point tracking (dirty indicator)
- **UTF-8 support** with CJK/wide character rendering
- **Mouse support** — click to position cursor, scroll wheel, tab clicks
- **Clipboard** — Ctrl+C/X/V (system clipboard via OSC 52)
- **Split editor** — vertical split for side-by-side editing

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
- **Find and replace** (Ctrl+H) with replace all support
- **Project-wide search** (Ctrl+Shift+F) via ripgrep with results panel
- **Tab management** — Ctrl+Tab cycle, Alt+1~9 direct switch

### Layout
- **File explorer sidebar** (Ctrl+B) with Nerd Font icons and file-type colors
- **Sidebar file operations** — Ctrl+N create, r delete, F2 rename
- **Bottom panel** (Ctrl+J) for search results, problems, output
- **Tab bar** with modified indicator
- **Status bar** — filename, language, cursor position, encoding, key hints
- **Focus navigation** — Alt+Left/Right to switch between sidebar and editor

### Git Integration
- **Git graph** — visual commit history with branch graph (like `git log --graph`)
- **Stage/unstage** — per-file staging from graph view (Space/u)
- **Commit, push, pull** — with confirmation dialogs
- **Merge & rebase** — branch picker with confirmation
- **Tag, stash, stash pop** — all from graph view
- **Git blame** — inline display with author, date, summary; click hash to jump to graph
- **Gutter diff markers** — added/modified/deleted line indicators

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
| Ctrl+N | New file/directory (sidebar) |
| F2 | Rename (sidebar) |
| F12 | Go to definition |
| Alt+Left / Alt+Right | Focus sidebar / editor |
| Alt+1~9 | Switch to tab N |

**Git Graph** (open via command palette `> Git Graph`):

| Key | Action |
|-----|--------|
| c | Commit |
| p | Push |
| P | Pull |
| a | Stage all |
| Space | Stage file |
| u | Unstage file |
| t | Tag |
| m | Merge |
| r | Rebase |
| s | Stash |
| S | Stash pop |

## Installation

### From source (requires Go 1.22+ and a C compiler for tree-sitter)

```bash
go install github.com/JiHyeongSeo/ted/cmd/ted@latest
```

### From GitHub Releases

Download pre-built binaries from the [Releases](https://github.com/JiHyeongSeo/ted/releases) page.

```bash
# Example: Linux amd64
curl -Lo ted.tar.gz https://github.com/JiHyeongSeo/ted/releases/latest/download/ted_linux_amd64.tar.gz
tar xzf ted.tar.gz
sudo mv ted /usr/local/bin/
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

# Check version
ted --version
```

## Architecture

```
ted/
├── cmd/ted/main.go           # Entry point
├── internal/
│   ├── buffer/               # Piece Table text buffer + undo/redo
│   ├── config/               # JSON config loading (global + project)
│   ├── editor/               # Editor core, event loop, commands, git ops
│   ├── git/                  # Git operations, blame, graph layout
│   ├── input/                # Keybinding system (JSON-configurable)
│   ├── lsp/                  # LSP client (JSON-RPC 2.0)
│   ├── search/               # In-file + project-wide search
│   ├── syntax/               # Tree-sitter + keyword-based highlighting
│   ├── types/                # Shared types (Position, Rect)
│   └── view/                 # UI components (EditorView, Sidebar, GraphView, etc.)
└── configs/default.json      # Default settings
```

## License

MIT
