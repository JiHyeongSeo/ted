package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectSearchGo(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello world\ngoodbye world"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("hello there\nno match here"), 0644)

	ps := NewProjectSearch(dir, nil, false)
	matches, err := ps.Search("hello", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
}

func TestProjectSearchExcludes(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "node_modules"), 0755)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("find me"), 0644)
	os.WriteFile(filepath.Join(dir, "node_modules", "b.txt"), []byte("find me"), 0644)

	ps := NewProjectSearch(dir, []string{"node_modules"}, false)
	matches, err := ps.Search("find me", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match (excludes node_modules), got %d", len(matches))
	}
}

func TestProjectSearchRegex(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("abc 123\ndef 456"), 0644)

	ps := NewProjectSearch(dir, nil, false)
	matches, err := ps.Search(`\d+`, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
}

func TestProjectSearchNoMatches(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello world"), 0644)

	ps := NewProjectSearch(dir, nil, false)
	matches, err := ps.Search("nonexistent", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(matches))
	}
}

func TestParseRipgrepOutput(t *testing.T) {
	output := "/path/file.go:10:5:func main() {\n/path/other.go:20:1:import \"fmt\"\n"
	matches := parseRipgrepOutput(output)
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
	if matches[0].File != "/path/file.go" {
		t.Errorf("expected file '/path/file.go', got %q", matches[0].File)
	}
	if matches[0].Line != 10 {
		t.Errorf("expected line 10, got %d", matches[0].Line)
	}
	if matches[0].Col != 5 {
		t.Errorf("expected col 5, got %d", matches[0].Col)
	}
}

func TestFileMatchString(t *testing.T) {
	fm := FileMatch{File: "test.go", Line: 5, Col: 3, Text: "hello"}
	s := fm.String()
	if s != "test.go:5:3: hello" {
		t.Errorf("unexpected string: %q", s)
	}
}
