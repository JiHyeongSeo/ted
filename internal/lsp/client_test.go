package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"
)

// readLSPMessage reads a single LSP message from a reader (for test helpers).
func readLSPMessage(r *bufio.Reader) (json.RawMessage, error) {
	contentLength := 0
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if after, ok := strings.CutPrefix(line, "Content-Length:"); ok {
			contentLength, _ = strconv.Atoi(strings.TrimSpace(after))
		}
	}
	if contentLength == 0 {
		return nil, fmt.Errorf("missing Content-Length")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

// writeLSPMessage writes a single LSP message to a writer (for test helpers).
func writeLSPMessage(w io.Writer, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, err := w.Write([]byte(header)); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func TestWriteAndReadMessage(t *testing.T) {
	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()

	client := NewClient(pr, pw)

	go func() {
		client.Notify("test/method", map[string]string{"key": "value"})
	}()

	msg, err := client.readMessage()
	if err != nil {
		t.Fatal(err)
	}

	var notif Notification
	if err := json.Unmarshal(msg, &notif); err != nil {
		t.Fatal(err)
	}
	if notif.Method != "test/method" {
		t.Errorf("expected method 'test/method', got %q", notif.Method)
	}
}

func TestCallAndResponse(t *testing.T) {
	// client reads from serverWriter, writes to clientWriter
	// server reads from clientWriter (via serverReader), writes to serverWriter
	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	client := NewClient(clientReader, clientWriter)
	go client.Start()

	// Mock server goroutine
	go func() {
		reader := bufio.NewReader(serverReader)
		raw, err := readLSPMessage(reader)
		if err != nil {
			return
		}
		var req Request
		json.Unmarshal(raw, &req)

		resp := Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]string{"status": "ok"},
		}
		writeLSPMessage(serverWriter, resp)
	}()

	resp, err := client.Call("test/call", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	clientReader.Close()
	clientWriter.Close()
	serverReader.Close()
	serverWriter.Close()
}

func TestNotificationHandler(t *testing.T) {
	pr, pw := io.Pipe()

	client := NewClient(pr, &bytes.Buffer{})

	received := make(chan string, 1)
	client.SetNotificationHandler(func(method string, params json.RawMessage) {
		received <- method
	})

	go client.Start()

	notif := Notification{
		JSONRPC: "2.0",
		Method:  "textDocument/publishDiagnostics",
		Params:  json.RawMessage(`{"uri":"file:///test.go"}`),
	}
	writeLSPMessage(pw, notif)

	select {
	case method := <-received:
		if method != "textDocument/publishDiagnostics" {
			t.Errorf("expected 'textDocument/publishDiagnostics', got %q", method)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for notification")
	}

	pr.Close()
}

func TestProtocolTypes(t *testing.T) {
	params := InitializeParams{
		ProcessID: 1234,
		RootURI:   "file:///workspace",
		Capabilities: ClientCapabilities{
			TextDocument: TextDocumentClientCapabilities{
				Hover: &HoverClientCapabilities{
					ContentFormat: []string{"plaintext"},
				},
			},
		},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}

	var decoded InitializeParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.ProcessID != 1234 {
		t.Errorf("expected processId 1234, got %d", decoded.ProcessID)
	}
	if decoded.RootURI != "file:///workspace" {
		t.Errorf("expected rootUri 'file:///workspace', got %q", decoded.RootURI)
	}
}

func TestDiagnosticSerialization(t *testing.T) {
	diag := Diagnostic{
		Range: Range{
			Start: Position{Line: 10, Character: 5},
			End:   Position{Line: 10, Character: 15},
		},
		Severity: DiagnosticSeverityError,
		Message:  "undefined variable",
	}

	data, err := json.Marshal(diag)
	if err != nil {
		t.Fatal(err)
	}

	var decoded Diagnostic
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.Range.Start.Line != 10 {
		t.Errorf("expected line 10, got %d", decoded.Range.Start.Line)
	}
	if decoded.Severity != DiagnosticSeverityError {
		t.Errorf("expected severity 1, got %d", decoded.Severity)
	}
}
