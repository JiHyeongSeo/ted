package view

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JiHyeongSeo/ted/internal/syntax"
)

func TestSidebarSetRoot(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte("package b"), 0644)
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "sub", "c.go"), []byte("package c"), 0644)

	theme := syntax.DefaultTheme()
	sb := NewSidebar(theme)
	sb.SetRoot(dir)

	if len(sb.flatEntries) < 3 {
		t.Errorf("expected at least 3 flat entries, got %d", len(sb.flatEntries))
	}

	// First entry should be directory (sorted first)
	if !sb.flatEntries[0].IsDir {
		t.Error("expected first entry to be directory")
	}
}

func TestSidebarHiddenFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".hidden"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "visible.go"), []byte(""), 0644)

	theme := syntax.DefaultTheme()
	sb := NewSidebar(theme)
	sb.SetRoot(dir)

	for _, entry := range sb.flatEntries {
		if entry.Name == ".hidden" {
			t.Error("hidden files should be excluded")
		}
	}
}

func TestSidebarExpand(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "sub", "file.go"), []byte(""), 0644)

	theme := syntax.DefaultTheme()
	sb := NewSidebar(theme)
	sb.SetRoot(dir)

	initial := len(sb.flatEntries)

	// Expand the directory
	for i, entry := range sb.flatEntries {
		if entry.IsDir {
			sb.selectedIdx = i
			entry.Expanded = true
			sb.rebuildFlat()
			break
		}
	}

	if len(sb.flatEntries) <= initial {
		t.Error("expanding directory should show more entries")
	}
}
