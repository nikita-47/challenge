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
		name := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
		if name == "" {
			http.Error(w, "session name required", http.StatusBadRequest)
			return
		}
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
	System      string  `json:"system"`
	MaxTokens   int     `json:"maxTokens"`
	Temperature float64 `json:"temperature"`
}

func handleChat(w http.ResponseWriter, r *http.Request, apiKey string) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	cfg := config{
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

	// Compress history BEFORE adding the new message.
	var stats tokenStats
	ci, compErr := maybeCompress(apiKey, cw, &stats)
	if compErr != nil {
		// Non-fatal: continue with full history.
	} else if ci != nil {
		cw.Messages = append(cw.Messages, message{
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

	compressed := buildCompressedMessages(cw)

	agent := newAgent(apiKey, cfg)
	result, err := agent.Run(req.Message, compressed, emit)
	if err != nil {
		sseWrite(w, map[string]any{"type": "error", "message": err.Error()})
		return
	}

	// Save to session (append to raw Messages, not compressed).
	cw.Messages = append(cw.Messages, message{Role: "user", Content: req.Message})
	cw.Messages = append(cw.Messages, message{Role: "assistant", Content: result})
	cw.Settings = &sessionSettings{
		Temperature: cfg.temperature,
		MaxTokens:   cfg.maxTokens,
		System:      cfg.system,
	}
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"messages": cw.Messages,
		"summary":  cw.Summary,
		"settings": cw.Settings,
	})
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
