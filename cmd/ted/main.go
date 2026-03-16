package main

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/JiHyeongSeo/ted/internal/config"
	"github.com/JiHyeongSeo/ted/internal/editor"
	"github.com/JiHyeongSeo/ted/internal/syntax"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("ted " + version)
		return
	}

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

	// Check minimum terminal size
	w, h := screen.Size()
	if w < 80 || h < 24 {
		screen.Fini()
		fmt.Fprintf(os.Stderr, "ted: terminal too small (%dx%d). Resize to at least 80x24.\n", w, h)
		os.Exit(1)
	}

	// Create editor
	ed := editor.New(cfg, theme)

	// Load keybindings
	ed.LoadKeybindings()

	// Open files/directories from command line
	if len(os.Args) > 1 {
		for _, path := range os.Args[1:] {
			info, err := os.Stat(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ted: cannot open %s: %v\n", path, err)
				continue
			}
			if info.IsDir() {
				// Open directory: set sidebar root and show sidebar
				ed.OpenDirectory(path)
			} else {
				if err := ed.OpenFile(path); err != nil {
					fmt.Fprintf(os.Stderr, "ted: cannot open %s: %v\n", path, err)
				}
			}
		}
	}

	// Run editor event loop
	if err := ed.Run(screen); err != nil {
		fmt.Fprintf(os.Stderr, "ted: %v\n", err)
		os.Exit(1)
	}
}
