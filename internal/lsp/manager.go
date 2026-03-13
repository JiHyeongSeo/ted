package lsp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
)

// ServerConfig holds configuration for an LSP server.
type ServerConfig struct {
	Command     string   `json:"command"`
	Args        []string `json:"args"`
	RootMarkers []string `json:"rootMarkers"`
}

// ServerState represents the lifecycle state of an LSP server.
type ServerState int

const (
	ServerStopped ServerState = iota
	ServerStarting
	ServerRunning
	ServerShuttingDown
)

// Server represents a running LSP server process.
type Server struct {
	config       ServerConfig
	cmd          *exec.Cmd
	client       *Client
	state        ServerState
	capabilities ServerCapabilities
	rootURI      string
}

// ServerManager manages LSP server instances per language.
type ServerManager struct {
	mu      sync.Mutex
	servers map[string]*Server // keyed by language ID
	configs map[string]ServerConfig
	onDiag  func(uri string, diags []Diagnostic)
}

// NewServerManager creates a new ServerManager.
func NewServerManager(configs map[string]ServerConfig) *ServerManager {
	return &ServerManager{
		servers: make(map[string]*Server),
		configs: configs,
	}
}

// SetDiagnosticHandler sets the callback for diagnostics.
func (sm *ServerManager) SetDiagnosticHandler(handler func(uri string, diags []Diagnostic)) {
	sm.onDiag = handler
}

// StartServer starts an LSP server for the given language.
func (sm *ServerManager) StartServer(language string, rootURI string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, ok := sm.servers[language]; ok {
		return nil // already running
	}

	cfg, ok := sm.configs[language]
	if !ok {
		return fmt.Errorf("no LSP config for language: %s", language)
	}

	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", cfg.Command, err)
	}

	client := NewClient(stdout, stdin)

	server := &Server{
		config:  cfg,
		cmd:     cmd,
		client:  client,
		state:   ServerStarting,
		rootURI: rootURI,
	}

	// Handle notifications
	client.SetNotificationHandler(func(method string, params json.RawMessage) {
		sm.handleNotification(language, method, params)
	})

	go client.Start()

	// Send initialize request
	initParams := InitializeParams{
		ProcessID: os.Getpid(),
		RootURI:   rootURI,
		Capabilities: ClientCapabilities{
			TextDocument: TextDocumentClientCapabilities{
				Completion: &CompletionClientCapabilities{},
				Hover: &HoverClientCapabilities{
					ContentFormat: []string{"plaintext", "markdown"},
				},
			},
		},
	}

	resp, err := client.Call("initialize", initParams)
	if err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("initialize: %w", err)
	}

	if resp.Error != nil {
		cmd.Process.Kill()
		return fmt.Errorf("initialize error: %s", resp.Error.Message)
	}

	// Parse capabilities
	if resp.Result != nil {
		resultJSON, _ := json.Marshal(resp.Result)
		var initResult InitializeResult
		json.Unmarshal(resultJSON, &initResult)
		server.capabilities = initResult.Capabilities
	}

	// Send initialized notification
	client.Notify("initialized", struct{}{})

	server.state = ServerRunning
	sm.servers[language] = server

	return nil
}

// StopServer stops the LSP server for the given language.
func (sm *ServerManager) StopServer(language string) error {
	sm.mu.Lock()
	server, ok := sm.servers[language]
	if !ok {
		sm.mu.Unlock()
		return nil
	}
	server.state = ServerShuttingDown
	sm.mu.Unlock()

	// Send shutdown request
	server.client.Call("shutdown", nil)
	server.client.Notify("exit", nil)
	server.client.Close()

	if server.cmd.Process != nil {
		server.cmd.Process.Kill()
		server.cmd.Wait()
	}

	sm.mu.Lock()
	delete(sm.servers, language)
	sm.mu.Unlock()

	return nil
}

// StopAll stops all running LSP servers.
func (sm *ServerManager) StopAll() {
	sm.mu.Lock()
	languages := make([]string, 0, len(sm.servers))
	for lang := range sm.servers {
		languages = append(languages, lang)
	}
	sm.mu.Unlock()

	for _, lang := range languages {
		sm.StopServer(lang)
	}
}

// GetClient returns the LSP client for a language, or nil if not running.
func (sm *ServerManager) GetClient(language string) *Client {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if server, ok := sm.servers[language]; ok && server.state == ServerRunning {
		return server.client
	}
	return nil
}

// GetCapabilities returns the server capabilities for a language.
func (sm *ServerManager) GetCapabilities(language string) *ServerCapabilities {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if server, ok := sm.servers[language]; ok {
		return &server.capabilities
	}
	return nil
}

// IsRunning returns whether an LSP server is running for the given language.
func (sm *ServerManager) IsRunning(language string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	server, ok := sm.servers[language]
	return ok && server.state == ServerRunning
}

func (sm *ServerManager) handleNotification(language, method string, params json.RawMessage) {
	switch method {
	case "textDocument/publishDiagnostics":
		var diagParams PublishDiagnosticsParams
		if err := json.Unmarshal(params, &diagParams); err != nil {
			return
		}
		if sm.onDiag != nil {
			sm.onDiag(diagParams.URI, diagParams.Diagnostics)
		}
	}
}
