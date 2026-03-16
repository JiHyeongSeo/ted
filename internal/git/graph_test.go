package git

import (
	"testing"
	"time"
)

func TestParseCommits(t *testing.T) {
	raw := "abc1234\x00abc1234567890abc1234567890abc1234567890\x00def5678901234def5678901234def5678901234\x00Alice\x001710000000\x00feat: add feature\x00HEAD -> main, origin/main\n" +
		"def5678\x00def5678901234def5678901234def5678901234\x00\x00Bob\x001709900000\x00initial commit\x00\n"

	commits, err := ParseCommits(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}

	c := commits[0]
	if c.ShortHash != "abc1234" {
		t.Errorf("expected short hash 'abc1234', got %q", c.ShortHash)
	}
	if c.Hash != "abc1234567890abc1234567890abc1234567890" {
		t.Errorf("unexpected hash: %q", c.Hash)
	}
	if len(c.Parents) != 1 || c.Parents[0] != "def5678901234def5678901234def5678901234" {
		t.Errorf("unexpected parents: %v", c.Parents)
	}
	if c.Author != "Alice" {
		t.Errorf("expected author 'Alice', got %q", c.Author)
	}
	if c.Date.Unix() != 1710000000 {
		t.Errorf("unexpected date: %v", c.Date)
	}
	if c.Message != "feat: add feature" {
		t.Errorf("expected message 'feat: add feature', got %q", c.Message)
	}
	if len(c.Refs) != 2 || c.Refs[0] != "HEAD -> main" || c.Refs[1] != "origin/main" {
		t.Errorf("unexpected refs: %v", c.Refs)
	}

	c2 := commits[1]
	if len(c2.Parents) != 0 {
		t.Errorf("expected no parents for root, got %v", c2.Parents)
	}
	_ = time.Now()
}

func TestParseCommitsEmpty(t *testing.T) {
	commits, err := ParseCommits("")
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 0 {
		t.Errorf("expected 0 commits, got %d", len(commits))
	}
}
