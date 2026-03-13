package editor

// RegisterBuiltinCommands registers all built-in editor commands.
func RegisterBuiltinCommands(reg *CommandRegistry) {
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
		Description: "Go to symbol definition",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("lsp.goToDefinition")
		},
	})

	reg.Register(&Command{
		Name:        "lsp.findReferences",
		Description: "Find all references to symbol",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("lsp.findReferences")
		},
	})

	reg.Register(&Command{
		Name:        "lsp.autocomplete",
		Description: "Trigger autocomplete",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("lsp.autocomplete")
		},
	})

	reg.Register(&Command{
		Name:        "lsp.hover",
		Description: "Show hover information",
		Execute: func(ctx EditorContext) error {
			return ctx.ExecuteCommand("lsp.hover")
		},
	})
}
