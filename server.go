package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

func startServer(apiKey string, cfg config) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleChat(w, r, apiKey)
	})

	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleListSessions(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/sessions/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
		if path == "" {
			http.Error(w, "session name required", http.StatusBadRequest)
			return
		}

		// Check for branch sub-routes: /api/sessions/:name/branches or /api/sessions/:name/branch
		if i := strings.Index(path, "/"); i >= 0 {
			name := path[:i]
			sub := path[i+1:]
			switch {
			case sub == "branches":
				handleBranches(w, r, apiKey, name)
			case sub == "branch":
				handleSwitchBranch(w, r, name)
			case sub == "raw" && r.Method == http.MethodGet:
				handleGetSessionRaw(w, name)
			default:
				http.Error(w, "not found", http.StatusNotFound)
			}
			return
		}

		name := path
		switch r.Method {
		case http.MethodGet:
			handleGetSession(w, r, name)
		case http.MethodPut:
			handleRenameSession(w, r, name)
		case http.MethodDelete:
			handleDeleteSession(w, r, name)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		model := "claude-sonnet-4-5-20250929"
		if cfg.model != "" {
			model = cfg.model
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"model":       model,
			"maxTokens":   cfg.maxTokens,
			"temperature": cfg.temperature,
			"system":      cfg.system,
		})
	})

	// Static files — serve frontend/dist/ if it exists.
	distDir := "frontend/dist"
	if info, err := os.Stat(distDir); err == nil && info.IsDir() {
		fs := http.FileServer(http.Dir(distDir))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// For SPA: if file doesn't exist, serve index.html.
			path := distDir + r.URL.Path
			if _, err := os.Stat(path); os.IsNotExist(err) && !strings.Contains(r.URL.Path, ".") {
				http.ServeFile(w, r, distDir+"/index.html")
				return
			}
			fs.ServeHTTP(w, r)
		})
	}

	addr := fmt.Sprintf(":%d", cfg.port)
	fmt.Printf("Server listening on http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

// ─── SSE helpers ─────────────────────────────────────────────────────────────

func sseSetup(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func sseWrite(w http.ResponseWriter, data any) {
	b, _ := json.Marshal(data)
	fmt.Fprintf(w, "data: %s\n\n", b)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// ─── POST /api/chat ──────────────────────────────────────────────────────────

type chatRequest struct {
	Message     string  `json:"message"`
	Session     string  `json:"session"`
	Model       string  `json:"model"`
	System      string  `json:"system"`
	MaxTokens   int     `json:"maxTokens"`
	Temperature float64 `json:"temperature"`
	Strategy    string  `json:"strategy"`
	WindowSize  int     `json:"windowSize"`
}

func handleChat(w http.ResponseWriter, r *http.Request, apiKey string) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	cfg := config{
		model:       req.Model,
		maxTokens:   req.MaxTokens,
		temperature: req.Temperature,
		system:      req.System,
	}

	// Load session for context.
	cw := &contextWindow{}
	if loaded, err := loadSessionCW(req.Session); err == nil && loaded != nil {
		cw = loaded
	}

	sseSetup(w)

	emit := func(ev AgentEvent) {
		sseWrite(w, ev)
	}

	// Apply strategy from request if session has no strategy set.
	if cw.Settings == nil {
		cw.Settings = &sessionSettings{}
	}
	if req.Strategy != "" && cw.Settings.Strategy == "" {
		cw.Settings.Strategy = req.Strategy
	}
	if req.WindowSize > 0 && cw.Settings.WindowSize == 0 {
		cw.Settings.WindowSize = req.WindowSize
	}

	// Restore cumulative stats from session (or start fresh).
	stats := cw.Stats.toTokenStats()

	// Pre-process context (compress for summary strategy, noop for others).
	ci, compErr := maybeProcess(apiKey, cw, &stats)
	if compErr != nil {
		// Non-fatal: continue with full history.
	} else if ci != nil {
		appendMessage(cw, message{
			Role: "system",
			Event: &messageEvent{
				Type:         "compress",
				MessageCount: ci.MessageCount,
				SummaryLen:   ci.SummaryLen,
				TokensSaved:  ci.TokensSaved,
			},
		})
		sseWrite(w, map[string]any{
			"type":         "compress",
			"messageCount": ci.MessageCount,
			"summaryLen":   ci.SummaryLen,
			"tokensSaved":  ci.TokensSaved,
		})
	}

	compressed := buildAPIMessages(cw)

	agent := newAgent(apiKey, cfg)
	result, err := agent.Run(req.Message, compressed, emit)
	if err != nil {
		sseWrite(w, map[string]any{"type": "error", "message": err.Error()})
		return
	}

	// Accumulate agent stats into session lifetime stats.
	stats.TotalInput += agent.Stats.TotalInput
	stats.TotalOutput += agent.Stats.TotalOutput
	stats.Exchanges += agent.Stats.Exchanges

	// Save to session (append to raw Messages, not compressed).
	appendMessage(cw, message{Role: "user", Content: req.Message})
	appendMessage(cw, message{Role: "assistant", Content: result})

	// Post-process: extract facts for facts strategy.
	if getStrategy(cw) == strategyFacts {
		if err := maybeExtractFacts(apiKey, cw, &stats); err != nil {
			// Non-fatal: log but continue.
		} else if len(cw.Facts) > 0 {
			sseWrite(w, map[string]any{
				"type":  "facts_updated",
				"facts": cw.Facts,
			})
		}
	}

	cw.Settings.Model = cfg.model
	cw.Settings.Temperature = cfg.temperature
	cw.Settings.MaxTokens = cfg.maxTokens
	cw.Settings.System = cfg.system
	cw.Stats = statsFromToken(stats)
	_ = saveSessionCW(req.Session, cw)
}

// ─── GET /api/sessions ───────────────────────────────────────────────────────

func handleListSessions(w http.ResponseWriter, r *http.Request) {
	names, err := listSessions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(names)
}

// ─── GET /api/sessions/:name ─────────────────────────────────────────────────

func handleGetSession(w http.ResponseWriter, r *http.Request, name string) {
	cw, err := loadSessionCW(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if cw == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	// Build branch info for response.
	var branchInfos []map[string]any
	for _, b := range cw.Branches {
		branchInfos = append(branchInfos, map[string]any{
			"name":         b.Name,
			"forkIndex":    b.ForkIndex,
			"messageCount": len(b.Messages),
			"createdAt":    b.CreatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"messages":     activeMessages(cw),
		"summary":      cw.Summary,
		"settings":     cw.Settings,
		"stats":        cw.Stats,
		"facts":        cw.Facts,
		"branches":     branchInfos,
		"activeBranch": cw.ActiveBranch,
	})
}

// ─── GET /api/sessions/:name/raw ─────────────────────────────────────────────

func handleGetSessionRaw(w http.ResponseWriter, name string) {
	data, err := os.ReadFile(sessionPath(name))
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// ─── DELETE /api/sessions/:name ──────────────────────────────────────────────

// ─── PUT /api/sessions/:name ─────────────────────────────────────────────────

func handleRenameSession(w http.ResponseWriter, r *http.Request, oldName string) {
	var req struct {
		NewName string `json:"newName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.NewName == "" {
		http.Error(w, "newName is required", http.StatusBadRequest)
		return
	}
	if err := renameSession(oldName, req.NewName); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "renamed", "name": req.NewName})
}

// ─── DELETE /api/sessions/:name ──────────────────────────────────────────────

func handleDeleteSession(w http.ResponseWriter, r *http.Request, name string) {
	if err := deleteSession(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// ─── Branch endpoints ───────────────────────────────────────────────────────

func handleBranches(w http.ResponseWriter, r *http.Request, apiKey string, sessionName string) {
	switch r.Method {
	case http.MethodGet:
		cw, err := loadSessionCW(sessionName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if cw == nil {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		var infos []map[string]any
		for _, b := range cw.Branches {
			infos = append(infos, map[string]any{
				"name":         b.Name,
				"forkIndex":    b.ForkIndex,
				"messageCount": len(b.Messages),
				"createdAt":    b.CreatedAt,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"branches":     infos,
			"activeBranch": cw.ActiveBranch,
		})

	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}
		cw, err := loadSessionCW(sessionName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if cw == nil {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		if err := createBranch(cw, req.Name); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		_ = saveSessionCW(sessionName, cw)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "created", "branch": req.Name})

	case http.MethodDelete:
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}
		cw, err := loadSessionCW(sessionName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if cw == nil {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		if err := deleteBranch(cw, req.Name); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		_ = saveSessionCW(sessionName, cw)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleSwitchBranch(w http.ResponseWriter, r *http.Request, sessionName string) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	cw, err := loadSessionCW(sessionName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if cw == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if err := switchBranch(cw, req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = saveSessionCW(sessionName, cw)

	// Return messages for the new branch.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":   "switched",
		"branch":   req.Name,
		"messages": activeMessages(cw),
	})
}
