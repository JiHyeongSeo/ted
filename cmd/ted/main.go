package main

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/seoji/ted/internal/config"
	"github.com/seoji/ted/internal/editor"
	"github.com/seoji/ted/internal/syntax"
)

func main() {
	// Load configuration
	cfg, err := config.Load(config.DefaultUserConfigDir(), ".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ted: config error: %v\n", err)
		os.Exit(1)
	}

	// Load theme
	theme := syntax.DefaultTheme()

	// Initialize screen
	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ted: screen error: %v\n", err)
		os.Exit(1)
	}
	if err := screen.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "ted: screen init error: %v\n", err)
		os.Exit(1)
	}
	defer screen.Fini()

	screen.EnableMouse()
	screen.SetStyle(tcell.StyleDefault.
		Background(theme.ResolveColor(theme.UI["background"])).
		Foreground(theme.ResolveColor(theme.UI["foreground"])))

	// Create editor
	ed := editor.New(cfg, theme)

	// Load keybindings
	ed.LoadKeybindings()

	// Open files from command line
	if len(os.Args) > 1 {
		for _, path := range os.Args[1:] {
			if err := ed.OpenFile(path); err != nil {
				fmt.Fprintf(os.Stderr, "ted: cannot open %s: %v\n", path, err)
			}
		}
	}

	// Run editor event loop
	if err := ed.Run(screen); err != nil {
		fmt.Fprintf(os.Stderr, "ted: %v\n", err)
		os.Exit(1)
	}
}
