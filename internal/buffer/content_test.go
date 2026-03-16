package buffer

import (
	"os"
	"testing"
)

func TestStringContent(t *testing.T) {
	cs := NewStringContent("hello world")
	if cs.Len() != 11 {
		t.Errorf("expected length 11, got %d", cs.Len())
	}
	got := cs.Slice(0, 5)
	if string(got) != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
	got = cs.Slice(6, 11)
	if string(got) != "world" {
		t.Errorf("expected 'world', got %q", got)
	}
}

func TestStringContentByteAt(t *testing.T) {
	cs := NewStringContent("abc")
	if cs.ByteAt(0) != 'a' {
		t.Errorf("expected 'a', got %c", cs.ByteAt(0))
	}
	if cs.ByteAt(2) != 'c' {
		t.Errorf("expected 'c', got %c", cs.ByteAt(2))
	}
}

func TestMmapContent(t *testing.T) {
	f, err := os.CreateTemp("", "ted-mmap-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	content := "hello mmap world\nline two\n"
	f.WriteString(content)
	f.Close()

	mc, err := NewMmapContent(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer mc.Close()

	if mc.Len() != len(content) {
		t.Errorf("expected length %d, got %d", len(content), mc.Len())
	}
	got := mc.Slice(0, 5)
	if string(got) != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
	if mc.ByteAt(6) != 'm' {
		t.Errorf("expected 'm', got %c", mc.ByteAt(6))
	}
}
