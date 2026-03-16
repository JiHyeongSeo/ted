package editor

// RegisterBuiltinCommands registers all built-in editor commands.
func RegisterBuiltinCommands(reg *CommandRegistry) {
	reg.Register(&Command{
		Name:        "file.new",
		Description: "New untitled file",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("file.new")
		},
	})

	reg.Register(&Command{
		Name:        "file.save",
		Description: "Save the current file",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("file.save")
		},
	})

	reg.Register(&Command{
		Name:        "file.open",
		Description: "Open a file",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("file.open")
		},
	})

	reg.Register(&Command{
		Name:        "file.close",
		Description: "Close the current tab",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("file.close")
		},
	})

	reg.Register(&Command{
		Name:        "editor.format",
		Description: "Format/Beautify document (JSON built-in; HTML/CSS/JS via prettier; Go via gofmt; Python via black; SQL via sqlformat)",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("editor.format")
		},
	})

	reg.Register(&Command{
		Name:        "edit.undo",
		Description: "Undo the last edit",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("edit.undo")
		},
	})

	reg.Register(&Command{
		Name:        "edit.redo",
		Description: "Redo the last undone edit",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("edit.redo")
		},
	})

	reg.Register(&Command{
		Name:        "search.find",
		Description: "Find text in the current file",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("search.find")
		},
	})

	reg.Register(&Command{
		Name:        "search.replace",
		Description: "Find and replace text",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("search.replace")
		},
	})

	reg.Register(&Command{
		Name:        "search.findInFiles",
		Description: "Search across project files",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("search.findInFiles")
		},
	})

	reg.Register(&Command{
		Name:        "palette.open",
		Description: "Open the command palette",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("palette.open")
		},
	})

	reg.Register(&Command{
		Name:        "editor.goToLine",
		Description: "Go to a specific line number",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("editor.goToLine")
		},
	})

	reg.Register(&Command{
		Name:        "sidebar.toggle",
		Description: "Toggle the sidebar",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("sidebar.toggle")
		},
	})

	reg.Register(&Command{
		Name:        "panel.toggle",
		Description: "Toggle the bottom panel",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("panel.toggle")
		},
	})

	reg.Register(&Command{
		Name:        "tab.next",
		Description: "Switch to the next tab",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("tab.next")
		},
	})

	reg.Register(&Command{
		Name:        "tab.previous",
		Description: "Switch to the previous tab",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("tab.previous")
		},
	})

	reg.Register(&Command{
		Name:        "lsp.goToDefinition",
		Description: "Jump to where the symbol under cursor is defined",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("lsp.goToDefinition")
		},
	})

	reg.Register(&Command{
		Name:        "lsp.findReferences",
		Description: "List all usages of the symbol under cursor",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("lsp.findReferences")
		},
	})

	reg.Register(&Command{
		Name:        "lsp.autocomplete",
		Description: "Trigger code completion at cursor (also auto-triggers on . and :)",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("lsp.autocomplete")
		},
	})

	reg.Register(&Command{
		Name:        "lsp.hover",
		Description: "Show type info and docs for symbol under cursor",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("lsp.hover")
		},
	})

	// Git commands
	reg.Register(&Command{
		Name:        "git.status",
		Description: "Show git status",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("git.status")
		},
	})
	reg.Register(&Command{
		Name:        "git.stageFile",
		Description: "Stage current file",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("git.stageFile")
		},
	})
	reg.Register(&Command{
		Name:        "git.stageAll",
		Description: "Stage all changes",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("git.stageAll")
		},
	})
	reg.Register(&Command{
		Name:        "git.commit",
		Description: "Commit staged changes",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("git.commit")
		},
	})
	reg.Register(&Command{
		Name:        "git.push",
		Description: "Push to remote",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("git.push")
		},
	})
	reg.Register(&Command{
		Name:        "git.pull",
		Description: "Pull from remote",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("git.pull")
		},
	})
	reg.Register(&Command{
		Name:        "git.blame",
		Description: "Toggle git blame",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("git.blame")
		},
	})
	reg.Register(&Command{
		Name:        "git.graph",
		Description: "Show git commit graph",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("git.graph")
		},
	})

	// Python commands
	reg.Register(&Command{
		Name:        "python.selectEnv",
		Description: "Select Python virtual environment",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("python.selectEnv")
		},
	})

	// Split commands
	reg.Register(&Command{
		Name:        "split.vertical",
		Description: "Split editor vertically",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("split.vertical")
		},
	})
	reg.Register(&Command{
		Name:        "split.close",
		Description: "Close current split pane",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("split.close")
		},
	})
	reg.Register(&Command{
		Name:        "split.focus",
		Description: "Switch focus to other pane",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("split.focus")
		},
	})
}
