package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

func startServer(apiKey string, port int) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleChat(w, r, apiKey)
	})

	mux.HandleFunc("/api/agent", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleAgent(w, r, apiKey)
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
		case http.MethodDelete:
			handleDeleteSession(w, r, name)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
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

	addr := fmt.Sprintf(":%d", port)
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

	if req.MaxTokens <= 0 {
		req.MaxTokens = 1024
	}
	if req.Temperature == 0 {
		req.Temperature = -1 // API default
	}

	cfg := config{
		maxTokens:   req.MaxTokens,
		temperature: req.Temperature,
		system:      req.System,
	}

	// Load session.
	cw := &contextWindow{}
	if loaded, err := loadSessionCW(req.Session); err == nil && loaded != nil {
		cw = loaded
	}

	var stats tokenStats

	// Compress if needed.
	ci, compErr := maybeCompress(apiKey, cw, &stats)
	if compErr != nil {
		// Non-fatal, continue.
		_ = compErr
	}

	cw.Messages = append(cw.Messages, message{Role: "user", Content: req.Message})
	compressed := buildCompressedMessages(cw)

	sseSetup(w)

	if ci != nil {
		sseWrite(w, map[string]any{
			"type":         "compress",
			"messageCount": ci.MessageCount,
			"summaryLen":   ci.SummaryLen,
			"tokensSaved":  ci.TokensSaved,
		})
	}

	onToken := func(text string) {
		sseWrite(w, map[string]any{
			"type": "text_delta",
			"text": text,
		})
	}

	reply, usage, err := streamChat(apiKey, cfg, compressed, onToken)
	if err != nil {
		sseWrite(w, map[string]any{"type": "error", "message": err.Error()})
		return
	}

	cw.Messages = append(cw.Messages, message{Role: "assistant", Content: reply})
	_ = saveSessionCW(req.Session, cw)

	sseWrite(w, map[string]any{
		"type":   "usage",
		"input":  usage.InputTokens,
		"output": usage.OutputTokens,
	})
	sseWrite(w, map[string]any{"type": "done"})
}

// ─── POST /api/agent ─────────────────────────────────────────────────────────

type agentRequest struct {
	Task    string `json:"task"`
	Session string `json:"session"`
}

func handleAgent(w http.ResponseWriter, r *http.Request, apiKey string) {
	var req agentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Load session for context.
	cw := &contextWindow{}
	if loaded, err := loadSessionCW(req.Session); err == nil && loaded != nil {
		cw = loaded
	}

	sseSetup(w)

	agent := newAgent(apiKey)
	emit := func(ev AgentEvent) {
		sseWrite(w, ev)
	}

	result, err := agent.Run(req.Task, cw.Messages, emit)
	if err != nil {
		sseWrite(w, map[string]any{"type": "error", "message": err.Error()})
		return
	}

	// Save flattened result to session.
	cw.Messages = append(cw.Messages, message{Role: "user", Content: req.Task})
	cw.Messages = append(cw.Messages, message{Role: "assistant", Content: result})
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
	})
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
