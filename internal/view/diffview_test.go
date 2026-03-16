package view

import "testing"

func TestComputeSideBySide_NoChanges(t *testing.T) {
	lines := computeSideBySide("a\nb\nc\n", "a\nb\nc\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	for _, l := range lines {
		if l.Kind != DiffEqual {
			t.Errorf("expected DiffEqual, got %d", l.Kind)
		}
	}
}

func TestComputeSideBySide_Addition(t *testing.T) {
	lines := computeSideBySide("a\nc\n", "a\nb\nc\n")
	// a=equal, b=added, c=equal
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0].Kind != DiffEqual || lines[0].LeftText != "a" {
		t.Errorf("line 0: expected equal 'a', got %v", lines[0])
	}
	if lines[1].Kind != DiffAdded || lines[1].RightText != "b" {
		t.Errorf("line 1: expected added 'b', got %v", lines[1])
	}
	if lines[2].Kind != DiffEqual || lines[2].LeftText != "c" {
		t.Errorf("line 2: expected equal 'c', got %v", lines[2])
	}
}

func TestComputeSideBySide_Removal(t *testing.T) {
	lines := computeSideBySide("a\nb\nc\n", "a\nc\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[1].Kind != DiffRemoved || lines[1].LeftText != "b" {
		t.Errorf("line 1: expected removed 'b', got %v", lines[1])
	}
}

func TestComputeSideBySide_Empty(t *testing.T) {
	lines := computeSideBySide("", "a\nb\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	for _, l := range lines {
		if l.Kind != DiffAdded {
			t.Errorf("expected DiffAdded, got %d", l.Kind)
		}
	}
}
