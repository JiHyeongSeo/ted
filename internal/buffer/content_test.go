package buffer

import "testing"

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
