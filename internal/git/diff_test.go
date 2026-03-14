package git

import (
	"testing"

	"github.com/seoji/ted/internal/types"
)

func TestParseHunks_PureAddition(t *testing.T) {
	// Added 3 lines starting at line 5
	diff := `diff --git a/foo.go b/foo.go
index abc..def 100644
--- a/foo.go
+++ b/foo.go
@@ -4,0 +5,3 @@ some context
+line1
+line2
+line3
`
	markers := ParseHunksForTest(diff)

	for _, line := range []int{4, 5, 6} {
		if markers[line] != types.MarkAdded {
			t.Errorf("line %d: expected MarkAdded, got %v", line, markers[line])
		}
	}
	if _, ok := markers[3]; ok {
		t.Errorf("line 3 should not have a marker")
	}
	if _, ok := markers[7]; ok {
		t.Errorf("line 7 should not have a marker")
	}
}

func TestParseHunks_PureAdditionSingleLine(t *testing.T) {
	// Added 1 line at line 10
	diff := `@@ -9,0 +10,1 @@
+new line
`
	markers := ParseHunksForTest(diff)

	if markers[9] != types.MarkAdded {
		t.Errorf("line 9: expected MarkAdded, got %v", markers[9])
	}
	if len(markers) != 1 {
		t.Errorf("expected 1 marker, got %d", len(markers))
	}
}

func TestParseHunks_PureDeletion(t *testing.T) {
	// Deleted 2 lines at old line 5, new position is line 5
	diff := `@@ -5,2 +4,0 @@ context
-deleted1
-deleted2
`
	markers := ParseHunksForTest(diff)

	if markers[3] != types.MarkDeleted {
		t.Errorf("line 3: expected MarkDeleted, got %v", markers[3])
	}
	if len(markers) != 1 {
		t.Errorf("expected 1 marker, got %d", len(markers))
	}
}

func TestParseHunks_DeletionAtStart(t *testing.T) {
	// Deleted lines at the very start of file
	diff := `@@ -1,2 +0,0 @@
-line1
-line2
`
	markers := ParseHunksForTest(diff)

	// newStart=0, so max(0, 0-1) = max(0, -1) = 0
	if markers[0] != types.MarkDeleted {
		t.Errorf("line 0: expected MarkDeleted, got %v", markers[0])
	}
	if len(markers) != 1 {
		t.Errorf("expected 1 marker, got %d", len(markers))
	}
}

func TestParseHunks_Modification(t *testing.T) {
	// Changed 2 lines starting at line 10
	diff := `@@ -10,2 +10,2 @@ context
-old1
-old2
+new1
+new2
`
	markers := ParseHunksForTest(diff)

	if markers[9] != types.MarkModified {
		t.Errorf("line 9: expected MarkModified, got %v", markers[9])
	}
	if markers[10] != types.MarkModified {
		t.Errorf("line 10: expected MarkModified, got %v", markers[10])
	}
	if len(markers) != 2 {
		t.Errorf("expected 2 markers, got %d", len(markers))
	}
}

func TestParseHunks_ModificationDifferentCounts(t *testing.T) {
	// Replaced 1 line with 3 lines at line 5
	diff := `@@ -5,1 +5,3 @@ context
-old
+new1
+new2
+new3
`
	markers := ParseHunksForTest(diff)

	for _, line := range []int{4, 5, 6} {
		if markers[line] != types.MarkModified {
			t.Errorf("line %d: expected MarkModified, got %v", line, markers[line])
		}
	}
	if len(markers) != 3 {
		t.Errorf("expected 3 markers, got %d", len(markers))
	}
}

func TestParseHunks_CountOmitted(t *testing.T) {
	// When count is omitted, it means 1
	diff := `@@ -10 +10 @@
-old
+new
`
	markers := ParseHunksForTest(diff)

	if markers[9] != types.MarkModified {
		t.Errorf("line 9: expected MarkModified, got %v", markers[9])
	}
	if len(markers) != 1 {
		t.Errorf("expected 1 marker, got %d", len(markers))
	}
}

func TestParseHunks_MultipleHunks(t *testing.T) {
	diff := `@@ -3,0 +4,2 @@ context1
+added1
+added2
@@ -10,1 +12,1 @@ context2
-old
+new
@@ -20,3 +22,0 @@ context3
-del1
-del2
-del3
`
	markers := ParseHunksForTest(diff)

	// First hunk: added at lines 3,4 (0-based)
	if markers[3] != types.MarkAdded {
		t.Errorf("line 3: expected MarkAdded, got %v", markers[3])
	}
	if markers[4] != types.MarkAdded {
		t.Errorf("line 4: expected MarkAdded, got %v", markers[4])
	}

	// Second hunk: modified at line 11 (0-based)
	if markers[11] != types.MarkModified {
		t.Errorf("line 11: expected MarkModified, got %v", markers[11])
	}

	// Third hunk: deleted marker at line 21 (0-based)
	if markers[21] != types.MarkDeleted {
		t.Errorf("line 21: expected MarkDeleted, got %v", markers[21])
	}

	if len(markers) != 4 {
		t.Errorf("expected 4 markers, got %d", len(markers))
	}
}

func TestParseHunks_EmptyDiff(t *testing.T) {
	markers := ParseHunksForTest("")
	if len(markers) != 0 {
		t.Errorf("expected 0 markers for empty diff, got %d", len(markers))
	}
}

func TestParseHunks_BinaryFile(t *testing.T) {
	// Binary files produce no @@ hunks, so parseHunks gets empty or no hunk lines
	diff := `diff --git a/image.png b/image.png
Binary files /dev/null and b/image.png differ
`
	// The ComputeMarkers function handles binary check separately,
	// but parseHunks should return empty for input without @@ headers
	markers := ParseHunksForTest(diff)
	if len(markers) != 0 {
		t.Errorf("expected 0 markers for binary diff, got %d", len(markers))
	}
}
