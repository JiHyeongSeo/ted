package editor

import (
	"errors"
	"testing"
)

type mockContext struct {
	executed []string
}

func (m *mockContext) ActiveBuffer() interface{ Text() string } { return nil }
func (m *mockContext) ExecuteCommand(name string) error {
	m.executed = append(m.executed, name)
	return nil
}

func TestRegisterAndGet(t *testing.T) {
	reg := NewCommandRegistry()
	reg.Register(&Command{
		Name:        "test.cmd",
		Description: "A test command",
		Execute:     func(ctx EditorContext) error { return nil },
	})

	cmd := reg.Get("test.cmd")
	if cmd == nil {
		t.Fatal("expected command to be registered")
	}
	if cmd.Name != "test.cmd" {
		t.Errorf("expected name 'test.cmd', got %q", cmd.Name)
	}
}

func TestGetUnknown(t *testing.T) {
	reg := NewCommandRegistry()
	if cmd := reg.Get("nonexistent"); cmd != nil {
		t.Error("expected nil for unknown command")
	}
}

func TestExecute(t *testing.T) {
	reg := NewCommandRegistry()
	called := false
	reg.Register(&Command{
		Name:    "test.run",
		Execute: func(ctx EditorContext) error { called = true; return nil },
	})

	ctx := &mockContext{}
	err := reg.Execute("test.run", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("command was not executed")
	}
}

func TestExecuteUnknown(t *testing.T) {
	reg := NewCommandRegistry()
	ctx := &mockContext{}
	err := reg.Execute("nonexistent", ctx)
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestExecuteError(t *testing.T) {
	reg := NewCommandRegistry()
	reg.Register(&Command{
		Name:    "test.fail",
		Execute: func(ctx EditorContext) error { return errors.New("failed") },
	})

	ctx := &mockContext{}
	err := reg.Execute("test.fail", ctx)
	if err == nil || err.Error() != "failed" {
		t.Errorf("expected 'failed' error, got %v", err)
	}
}

func TestList(t *testing.T) {
	reg := NewCommandRegistry()
	reg.Register(&Command{Name: "a", Execute: func(ctx EditorContext) error { return nil }})
	reg.Register(&Command{Name: "b", Execute: func(ctx EditorContext) error { return nil }})

	names := reg.List()
	if len(names) != 2 {
		t.Errorf("expected 2 commands, got %d", len(names))
	}
}
