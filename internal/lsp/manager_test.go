package lsp

import (
	"testing"
)

func TestServerManagerCreation(t *testing.T) {
	configs := map[string]ServerConfig{
		"go":     {Command: "gopls", Args: []string{"serve"}, RootMarkers: []string{"go.mod"}},
		"python": {Command: "pylsp", Args: []string{}, RootMarkers: []string{"pyproject.toml"}},
	}

	sm := NewServerManager(configs)
	if sm == nil {
		t.Fatal("expected non-nil ServerManager")
	}
	if sm.IsRunning("go") {
		t.Error("go server should not be running")
	}
}

func TestServerManagerNoConfig(t *testing.T) {
	sm := NewServerManager(map[string]ServerConfig{})
	err := sm.StartServer("go", "file:///workspace")
	if err == nil {
		t.Error("expected error for missing config")
	}
}

func TestServerManagerStopNonExistent(t *testing.T) {
	sm := NewServerManager(map[string]ServerConfig{})
	err := sm.StopServer("go")
	if err != nil {
		t.Errorf("unexpected error stopping non-existent server: %v", err)
	}
}

func TestGetClientNil(t *testing.T) {
	sm := NewServerManager(map[string]ServerConfig{})
	client := sm.GetClient("go")
	if client != nil {
		t.Error("expected nil client")
	}
}

func TestDiagnosticNotificationHandler(t *testing.T) {
	h := NewNotificationHandler()

	diags := []Diagnostic{
		{
			Range:    Range{Start: Position{Line: 0, Character: 0}, End: Position{Line: 0, Character: 5}},
			Severity: DiagnosticSeverityError,
			Message:  "undefined: foo",
		},
		{
			Range:    Range{Start: Position{Line: 5, Character: 0}, End: Position{Line: 5, Character: 10}},
			Severity: DiagnosticSeverityWarning,
			Message:  "unused variable",
		},
	}

	changed := false
	h.SetOnChange(func() { changed = true })

	h.HandleDiagnostics("file:///test.go", diags)

	if !changed {
		t.Error("onChange should have been called")
	}
	if h.DiagnosticCount() != 2 {
		t.Errorf("expected 2 diagnostics, got %d", h.DiagnosticCount())
	}
	if h.ErrorCount() != 1 {
		t.Errorf("expected 1 error, got %d", h.ErrorCount())
	}
	if h.WarningCount() != 1 {
		t.Errorf("expected 1 warning, got %d", h.WarningCount())
	}

	// Clear diagnostics
	h.HandleDiagnostics("file:///test.go", nil)
	if h.DiagnosticCount() != 0 {
		t.Errorf("expected 0 diagnostics after clear, got %d", h.DiagnosticCount())
	}
}

func TestFormatDiagnostic(t *testing.T) {
	d := Diagnostic{
		Range:    Range{Start: Position{Line: 9, Character: 4}},
		Severity: DiagnosticSeverityError,
		Message:  "undefined: x",
	}
	s := FormatDiagnostic("file:///test.go", d)
	expected := "/test.go:10:5: error: undefined: x"
	if s != expected {
		t.Errorf("expected %q, got %q", expected, s)
	}
}

func TestFileURIConversion(t *testing.T) {
	uri := FileURIFromPath("/home/user/test.go")
	if uri != "file:///home/user/test.go" {
		t.Errorf("expected file:///home/user/test.go, got %q", uri)
	}

	path := PathFromFileURI(uri)
	if path != "/home/user/test.go" {
		t.Errorf("expected /home/user/test.go, got %q", path)
	}
}
