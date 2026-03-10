package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// ─── Config types ─────────────────────────────────────────────────────────────

// MCPServerConfig holds the configuration for a single MCP server.
type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// MCPServersFile is the top-level structure of .mcp_servers.json.
type MCPServersFile struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// ─── Connection and manager types ────────────────────────────────────────────

// MCPConnection represents a live connection to one MCP server.
type MCPConnection struct {
	Name      string
	Config    MCPServerConfig
	Client    mcpclient.MCPClient
	Tools     []mcp.Tool
	Connected bool
	Error     string
}

// MCPManager manages all MCP server connections.
type MCPManager struct {
	mu          sync.RWMutex
	connections map[string]*MCPConnection
	configPath  string
}

// ─── API response types ───────────────────────────────────────────────────────

// MCPServerStatus is the per-server status returned by the servers endpoint.
type MCPServerStatus struct {
	Name       string `json:"name"`
	Connected  bool   `json:"connected"`
	ToolsCount int    `json:"toolsCount"`
	Error      string `json:"error,omitempty"`
}

// MCPToolInfo is the per-tool info returned by the tools endpoint.
type MCPToolInfo struct {
	Server      string `json:"server"`
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

// ─── MCPManager constructor and lifecycle ─────────────────────────────────────

// NewMCPManager creates an MCPManager that will read config from configPath.
func NewMCPManager(configPath string) *MCPManager {
	return &MCPManager{
		connections: make(map[string]*MCPConnection),
		configPath:  configPath,
	}
}

// LoadConfig reads the JSON config file and populates the connections map with
// disconnected stubs. Missing config file is not an error.
func (m *MCPManager) LoadConfig() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read config: %w", err)
	}

	var file MCPServersFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for name, cfg := range file.MCPServers {
		m.connections[name] = &MCPConnection{
			Name:   name,
			Config: cfg,
		}
	}
	return nil
}

// ConnectAll attempts to connect to every configured server. Errors per server
// are stored in the connection and logged to stderr — not returned.
func (m *MCPManager) ConnectAll(ctx context.Context) {
	m.mu.RLock()
	names := make([]string, 0, len(m.connections))
	for name := range m.connections {
		names = append(names, name)
	}
	m.mu.RUnlock()

	for _, name := range names {
		if err := m.Connect(ctx, name); err != nil {
			fmt.Fprintf(os.Stderr, "[mcp] connect %q: %v\n", name, err)
		}
	}
}

// Connect establishes a connection to a single named server, performing the
// MCP handshake and caching the tool list.
func (m *MCPManager) Connect(ctx context.Context, name string) error {
	m.mu.RLock()
	conn, ok := m.connections[name]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("server %q not found in config", name)
	}

	// Disconnect any existing client before reconnecting.
	if conn.Connected && conn.Client != nil {
		_ = conn.Client.Close()
	}

	// Build env slice in KEY=value format.
	var envSlice []string
	for k, v := range conn.Config.Env {
		envSlice = append(envSlice, k+"="+v)
	}

	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	c, err := mcpclient.NewStdioMCPClient(conn.Config.Command, envSlice, conn.Config.Args...)
	if err != nil {
		m.mu.Lock()
		conn.Connected = false
		conn.Error = err.Error()
		m.mu.Unlock()
		return fmt.Errorf("create stdio client: %w", err)
	}

	// MCP handshake.
	_, err = c.Initialize(connectCtx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: "2024-11-05",
			ClientInfo: mcp.Implementation{
				Name:    "challenge-mcp-client",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		_ = c.Close()
		m.mu.Lock()
		conn.Connected = false
		conn.Error = err.Error()
		m.mu.Unlock()
		return fmt.Errorf("initialize: %w", err)
	}

	// Cache the tool list.
	listResult, err := c.ListTools(connectCtx, mcp.ListToolsRequest{})
	if err != nil {
		_ = c.Close()
		m.mu.Lock()
		conn.Connected = false
		conn.Error = err.Error()
		m.mu.Unlock()
		return fmt.Errorf("list tools: %w", err)
	}

	m.mu.Lock()
	conn.Client = c
	conn.Tools = listResult.Tools
	conn.Connected = true
	conn.Error = ""
	m.mu.Unlock()

	fmt.Printf("[mcp] connected to %q (%d tools)\n", name, len(listResult.Tools))
	return nil
}

// Disconnect closes the connection to a single named server.
func (m *MCPManager) Disconnect(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, ok := m.connections[name]
	if !ok {
		return fmt.Errorf("server %q not found", name)
	}
	if conn.Client != nil {
		if err := conn.Client.Close(); err != nil {
			return fmt.Errorf("close: %w", err)
		}
	}
	conn.Client = nil
	conn.Tools = nil
	conn.Connected = false
	return nil
}

// DisconnectAll closes every active connection.
func (m *MCPManager) DisconnectAll() {
	m.mu.RLock()
	names := make([]string, 0, len(m.connections))
	for name := range m.connections {
		names = append(names, name)
	}
	m.mu.RUnlock()

	for _, name := range names {
		if err := m.Disconnect(name); err != nil {
			fmt.Fprintf(os.Stderr, "[mcp] disconnect %q: %v\n", name, err)
		}
	}
}

// ListServers returns the status of every configured server.
func (m *MCPManager) ListServers() []MCPServerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]MCPServerStatus, 0, len(m.connections))
	for _, conn := range m.connections {
		result = append(result, MCPServerStatus{
			Name:       conn.Name,
			Connected:  conn.Connected,
			ToolsCount: len(conn.Tools),
			Error:      conn.Error,
		})
	}
	return result
}

// GetTools returns tool info for the given server name. If server is empty,
// tools from all connected servers are returned.
func (m *MCPManager) GetTools(server string) []MCPToolInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []MCPToolInfo
	for _, conn := range m.connections {
		if server != "" && conn.Name != server {
			continue
		}
		for _, t := range conn.Tools {
			result = append(result, MCPToolInfo{
				Server:      conn.Name,
				Name:        t.Name,
				Description: t.Description,
				InputSchema: t.InputSchema,
			})
		}
	}
	return result
}

// CallTool invokes a named tool on the named server with the provided arguments.
func (m *MCPManager) CallTool(ctx context.Context, server, tool string, args map[string]any) (*mcp.CallToolResult, error) {
	m.mu.RLock()
	conn, ok := m.connections[server]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("server %q not found", server)
	}
	if !conn.Connected || conn.Client == nil {
		return nil, fmt.Errorf("server %q is not connected", server)
	}

	return conn.Client.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      tool,
			Arguments: args,
		},
	})
}

// ─── HTTP handlers ────────────────────────────────────────────────────────────

// parseMCPToolName splits a namespaced tool name of the form "server__toolname"
// into its server and tool components. Returns ok=false if the name does not
// contain the double-underscore separator.
func parseMCPToolName(name string) (server, tool string, ok bool) {
	idx := strings.Index(name, "__")
	if idx < 0 {
		return "", "", false
	}
	return name[:idx], name[idx+2:], true
}

// GetToolDefs returns toolDef entries for MCP tools. If toolNames is non-empty,
// only tools whose namespaced name ("server__toolname") appears in the list are
// returned. An empty toolNames returns every tool from every connected server.
func (m *MCPManager) GetToolDefs(toolNames []string) []toolDef {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Build a lookup set from the requested names for O(1) access.
	var filterSet map[string]struct{}
	if len(toolNames) > 0 {
		filterSet = make(map[string]struct{}, len(toolNames))
		for _, n := range toolNames {
			filterSet[n] = struct{}{}
		}
	}

	var defs []toolDef
	for _, conn := range m.connections {
		if !conn.Connected {
			continue
		}
		for _, t := range conn.Tools {
			namespacedName := conn.Name + "__" + t.Name

			if filterSet != nil {
				if _, ok := filterSet[namespacedName]; !ok {
					continue
				}
			}

			// Convert ToolInputSchema to map[string]any via JSON round-trip so
			// it is compatible with the toolDef.InputSchema field (typed as any).
			schemaBytes, err := json.Marshal(t.InputSchema)
			if err != nil {
				continue
			}
			var inputSchema map[string]any
			if err := json.Unmarshal(schemaBytes, &inputSchema); err != nil {
				continue
			}

			defs = append(defs, toolDef{
				Name:        namespacedName,
				Description: t.Description,
				InputSchema: inputSchema,
			})
		}
	}
	return defs
}

// handleMCPServers handles GET /api/mcp/servers.
func handleMCPServers(w http.ResponseWriter, r *http.Request, mgr *MCPManager) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mgr.ListServers())
}

// handleMCPServerAction handles POST /api/mcp/servers/{name}/connect and
// POST /api/mcp/servers/{name}/disconnect.
func handleMCPServerAction(w http.ResponseWriter, r *http.Request, mgr *MCPManager) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Path: /api/mcp/servers/{name}/{action}
	path := strings.TrimPrefix(r.URL.Path, "/api/mcp/servers/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		http.Error(w, "invalid path — expected /api/mcp/servers/{name}/{action}", http.StatusBadRequest)
		return
	}
	name, action := parts[0], parts[1]

	switch action {
	case "connect":
		if err := mgr.Connect(r.Context(), name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "connected", "server": name})

	case "disconnect":
		if err := mgr.Disconnect(name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "disconnected", "server": name})

	default:
		http.Error(w, fmt.Sprintf("unknown action %q", action), http.StatusBadRequest)
	}
}

// handleMCPTools handles GET /api/mcp/tools (optional ?server=name query param).
func handleMCPTools(w http.ResponseWriter, r *http.Request, mgr *MCPManager) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	server := r.URL.Query().Get("server")
	tools := mgr.GetTools(server)
	if tools == nil {
		tools = []MCPToolInfo{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tools)
}

// handleMCPToolCall handles POST /api/mcp/tools/call.
// Request body: { "server": "...", "tool": "...", "arguments": {...} }
func handleMCPToolCall(w http.ResponseWriter, r *http.Request, mgr *MCPManager) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Server    string         `json:"server"`
		Tool      string         `json:"tool"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Server == "" || req.Tool == "" {
		http.Error(w, "server and tool are required", http.StatusBadRequest)
		return
	}

	result, err := mgr.CallTool(r.Context(), req.Server, req.Tool, req.Arguments)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleMCPReload handles POST /api/mcp/reload — re-reads config and reconnects.
func handleMCPReload(w http.ResponseWriter, r *http.Request, mgr *MCPManager) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mgr.DisconnectAll()

	// Reset connections map so removed servers are purged.
	mgr.mu.Lock()
	mgr.connections = make(map[string]*MCPConnection)
	mgr.mu.Unlock()

	if err := mgr.LoadConfig(); err != nil {
		http.Error(w, "config load: "+err.Error(), http.StatusInternalServerError)
		return
	}
	mgr.ConnectAll(r.Context())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "reloaded",
		"servers": mgr.ListServers(),
	})
}
