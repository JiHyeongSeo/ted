package lsp

import (
	"encoding/json"
	"fmt"
	"strings"
)

// NotificationHandler processes LSP server notifications.
type NotificationHandler struct {
	diagnostics map[string][]Diagnostic // URI -> diagnostics
	onChange    func()                   // callback when diagnostics change
}

// NewNotificationHandler creates a new NotificationHandler.
func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{
		diagnostics: make(map[string][]Diagnostic),
	}
}

// SetOnChange sets a callback invoked when diagnostics change.
func (h *NotificationHandler) SetOnChange(fn func()) {
	h.onChange = fn
}

// HandleDiagnostics processes a publishDiagnostics notification.
func (h *NotificationHandler) HandleDiagnostics(uri string, diags []Diagnostic) {
	if len(diags) == 0 {
		delete(h.diagnostics, uri)
	} else {
		h.diagnostics[uri] = diags
	}
	if h.onChange != nil {
		h.onChange()
	}
}

// GetDiagnostics returns diagnostics for a given URI.
func (h *NotificationHandler) GetDiagnostics(uri string) []Diagnostic {
	return h.diagnostics[uri]
}

// GetAllDiagnostics returns all diagnostics across all files.
func (h *NotificationHandler) GetAllDiagnostics() map[string][]Diagnostic {
	return h.diagnostics
}

// DiagnosticCount returns the total number of diagnostics.
func (h *NotificationHandler) DiagnosticCount() int {
	count := 0
	for _, diags := range h.diagnostics {
		count += len(diags)
	}
	return count
}

// ErrorCount returns the number of error-level diagnostics.
func (h *NotificationHandler) ErrorCount() int {
	count := 0
	for _, diags := range h.diagnostics {
		for _, d := range diags {
			if d.Severity == DiagnosticSeverityError {
				count++
			}
		}
	}
	return count
}

// WarningCount returns the number of warning-level diagnostics.
func (h *NotificationHandler) WarningCount() int {
	count := 0
	for _, diags := range h.diagnostics {
		for _, d := range diags {
			if d.Severity == DiagnosticSeverityWarning {
				count++
			}
		}
	}
	return count
}

// FormatDiagnostic returns a human-readable string for a diagnostic.
func FormatDiagnostic(uri string, d Diagnostic) string {
	severity := "info"
	switch d.Severity {
	case DiagnosticSeverityError:
		severity = "error"
	case DiagnosticSeverityWarning:
		severity = "warning"
	case DiagnosticSeverityHint:
		severity = "hint"
	}

	// Extract filename from URI
	file := uri
	if strings.HasPrefix(uri, "file://") {
		file = uri[7:]
	}

	return fmt.Sprintf("%s:%d:%d: %s: %s",
		file,
		d.Range.Start.Line+1,
		d.Range.Start.Character+1,
		severity,
		d.Message,
	)
}

// FileURIFromPath converts a file path to a file:// URI.
func FileURIFromPath(path string) string {
	return "file://" + path
}

// PathFromFileURI converts a file:// URI to a file path.
func PathFromFileURI(uri string) string {
	if strings.HasPrefix(uri, "file://") {
		return uri[7:]
	}
	return uri
}

// DidOpen sends a textDocument/didOpen notification.
func DidOpen(client *Client, uri, languageID, text string) error {
	return client.Notify("textDocument/didOpen", DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        uri,
			LanguageID: languageID,
			Version:    1,
			Text:       text,
		},
	})
}

// DidChange sends a textDocument/didChange notification with full sync.
func DidChange(client *Client, uri string, version int, text string) error {
	return client.Notify("textDocument/didChange", DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			URI:     uri,
			Version: version,
		},
		ContentChanges: []TextDocumentContentChangeEvent{
			{Text: text},
		},
	})
}

// DidSave sends a textDocument/didSave notification.
func DidSave(client *Client, uri, text string) error {
	return client.Notify("textDocument/didSave", DidSaveTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Text:         text,
	})
}

// DidClose sends a textDocument/didClose notification.
func DidClose(client *Client, uri string) error {
	return client.Notify("textDocument/didClose", DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	})
}

// RequestCompletion sends a textDocument/completion request.
func RequestCompletion(client *Client, uri string, line, character int) (*Response, error) {
	return client.Call("textDocument/completion", CompletionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     Position{Line: line, Character: character},
	})
}

// RequestHover sends a textDocument/hover request.
func RequestHover(client *Client, uri string, line, character int) (*Response, error) {
	return client.Call("textDocument/hover", TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     Position{Line: line, Character: character},
	})
}

// RequestDefinition sends a textDocument/definition request.
func RequestDefinition(client *Client, uri string, line, character int) (*Response, error) {
	return client.Call("textDocument/definition", TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     Position{Line: line, Character: character},
	})
}

// RequestReferences sends a textDocument/references request.
func RequestReferences(client *Client, uri string, line, character int) (*Response, error) {
	params := map[string]interface{}{
		"textDocument": TextDocumentIdentifier{URI: uri},
		"position":     Position{Line: line, Character: character},
		"context":      map[string]bool{"includeDeclaration": true},
	}
	return client.Call("textDocument/references", params)
}

// RequestDocumentSymbols requests document symbols for the given URI.
func RequestDocumentSymbols(client *Client, uri string) (*Response, error) {
	params := DocumentSymbolParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}
	return client.Call("textDocument/documentSymbol", params)
}

// ParseDocumentSymbols parses a documentSymbol response.
// Returns a flat list of (name, kind, line) tuples from both nested DocumentSymbol
// and flat SymbolInformation formats.
func ParseDocumentSymbols(resp *Response) ([]DocumentSymbol, error) {
	if resp.Error != nil {
		return nil, fmt.Errorf("documentSymbol error: %s", resp.Error.Message)
	}

	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, err
	}

	// Try nested DocumentSymbol[] first
	var docSyms []DocumentSymbol
	if err := json.Unmarshal(data, &docSyms); err == nil && len(docSyms) > 0 {
		return flattenDocumentSymbols(docSyms, ""), nil
	}

	// Try flat SymbolInformation[]
	var symInfos []SymbolInformation
	if err := json.Unmarshal(data, &symInfos); err == nil {
		result := make([]DocumentSymbol, 0, len(symInfos))
		for _, si := range symInfos {
			sym := DocumentSymbol{
				Name: si.Name,
				Kind: si.Kind,
				Range: Range{
					Start: si.Location.Range.Start,
					End:   si.Location.Range.End,
				},
				SelectionRange: si.Location.Range,
			}
			if si.ContainerName != "" {
				sym.Detail = si.ContainerName
			}
			result = append(result, sym)
		}
		return result, nil
	}

	return nil, nil
}

// flattenDocumentSymbols flattens a nested symbol tree into a flat list.
func flattenDocumentSymbols(syms []DocumentSymbol, prefix string) []DocumentSymbol {
	var result []DocumentSymbol
	for _, s := range syms {
		flat := s
		if prefix != "" {
			flat.Detail = prefix
		}
		flat.Children = nil
		result = append(result, flat)
		if len(s.Children) > 0 {
			childPrefix := s.Name
			if prefix != "" {
				childPrefix = prefix + "." + s.Name
			}
			result = append(result, flattenDocumentSymbols(s.Children, childPrefix)...)
		}
	}
	return result
}

// ParseCompletionResponse parses a completion response into items.
func ParseCompletionResponse(resp *Response) ([]CompletionItem, error) {
	if resp.Error != nil {
		return nil, fmt.Errorf("completion error: %s", resp.Error.Message)
	}

	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, err
	}

	// Try as CompletionList first
	var list CompletionList
	if err := json.Unmarshal(data, &list); err == nil && len(list.Items) > 0 {
		return list.Items, nil
	}

	// Try as []CompletionItem
	var items []CompletionItem
	if err := json.Unmarshal(data, &items); err == nil {
		return items, nil
	}

	return nil, nil
}
