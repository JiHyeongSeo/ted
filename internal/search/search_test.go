package search

import "testing"

func TestFindAllPlainText(t *testing.T) {
	s, err := NewInFileSearch("hello", false, true)
	if err != nil {
		t.Fatal(err)
	}

	text := "hello world\nsay hello\ngoodbye"
	matches := s.FindAll(text)
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
	if matches[0].Line != 0 || matches[0].Col != 0 {
		t.Errorf("match 0: expected (0,0), got (%d,%d)", matches[0].Line, matches[0].Col)
	}
	if matches[1].Line != 1 || matches[1].Col != 4 {
		t.Errorf("match 1: expected (1,4), got (%d,%d)", matches[1].Line, matches[1].Col)
	}
}

func TestFindAllCaseInsensitive(t *testing.T) {
	s, err := NewInFileSearch("hello", false, false)
	if err != nil {
		t.Fatal(err)
	}

	text := "Hello world\nHELLO there"
	matches := s.FindAll(text)
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
}

func TestFindAllRegex(t *testing.T) {
	s, err := NewInFileSearch(`\d+`, true, true)
	if err != nil {
		t.Fatal(err)
	}

	text := "abc 123\ndef 456 789"
	matches := s.FindAll(text)
	if len(matches) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(matches))
	}
}

func TestFindAllNoMatches(t *testing.T) {
	s, err := NewInFileSearch("xyz", false, true)
	if err != nil {
		t.Fatal(err)
	}

	matches := s.FindAll("hello world")
	if len(matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(matches))
	}
}

func TestFindNext(t *testing.T) {
	s, err := NewInFileSearch("find", false, true)
	if err != nil {
		t.Fatal(err)
	}

	text := "find me\ndon't find here\nfind again"
	m := s.FindNext(text, 0, 1) // start after first "find"
	if m == nil {
		t.Fatal("expected a match")
	}
	if m.Line != 1 || m.Col != 6 {
		t.Errorf("expected (1,6), got (%d,%d)", m.Line, m.Col)
	}
}

func TestFindNextWraparound(t *testing.T) {
	s, err := NewInFileSearch("first", false, true)
	if err != nil {
		t.Fatal(err)
	}

	text := "first line\nsecond line"
	m := s.FindNext(text, 1, 0) // start at line 1
	if m == nil {
		t.Fatal("expected a match with wraparound")
	}
	if m.Line != 0 || m.Col != 0 {
		t.Errorf("expected (0,0), got (%d,%d)", m.Line, m.Col)
	}
}

func TestInvalidRegex(t *testing.T) {
	_, err := NewInFileSearch("[invalid", true, true)
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestFindAllMultipleInSameLine(t *testing.T) {
	s, err := NewInFileSearch("ab", false, true)
	if err != nil {
		t.Fatal(err)
	}

	matches := s.FindAll("ababab")
	if len(matches) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(matches))
	}
}

func TestMatchLength(t *testing.T) {
	s, err := NewInFileSearch("world", false, true)
	if err != nil {
		t.Fatal(err)
	}

	matches := s.FindAll("hello world")
	if len(matches) != 1 {
		t.Fatal("expected 1 match")
	}
	if matches[0].Length != 5 {
		t.Errorf("expected length 5, got %d", matches[0].Length)
	}
}
