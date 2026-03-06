package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
)

// ─── Global provider settings ─────────────────────────────────────────────────

type providerSettings struct {
	mu         sync.RWMutex
	Provider   string `json:"provider"`   // "claude" | "local"
	LocalURL   string `json:"localURL"`   // e.g. "http://localhost:1234"
	LocalModel string `json:"localModel"` // e.g. "qwen2.5-0.5b-instruct-mlx"
}

func (ps *providerSettings) get() (provider, localURL, localModel string) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.Provider, ps.LocalURL, ps.LocalModel
}

func (ps *providerSettings) set(provider, localURL, localModel string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.Provider = provider
	ps.LocalURL = localURL
	ps.LocalModel = localModel
}

var globalProvider providerSettings

func startServer(apiKey string, cfg config) {
	// Initialize global provider from CLI flags.
	if cfg.baseURL != "" {
		globalProvider.set("local", cfg.baseURL, cfg.model)
	} else {
		globalProvider.set("claude", "http://localhost:1234", "qwen2.5-0.5b-instruct-mlx")
	}

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

	// ─── Memory endpoints ────────────────────────────────────────────────────
	mux.HandleFunc("/api/memory/profiles", func(w http.ResponseWriter, r *http.Request) {
		handleMemoryList(w, r, memoryProfilesDir(), saveProfile)
	})
	mux.HandleFunc("/api/memory/profiles/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/api/memory/profiles/")
		handleMemoryItem(w, r, name, memoryProfilesDir())
	})
	mux.HandleFunc("/api/memory/projects", func(w http.ResponseWriter, r *http.Request) {
		handleMemoryList(w, r, memoryProjectsDir(), saveProject)
	})
	mux.HandleFunc("/api/memory/projects/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/api/memory/projects/")
		handleMemoryItem(w, r, name, memoryProjectsDir())
	})

	mux.HandleFunc("/api/memory/operators", func(w http.ResponseWriter, r *http.Request) {
		handleMemoryList(w, r, memoryOperatorsDir(), saveOperator)
	})
	mux.HandleFunc("/api/memory/operators/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/api/memory/operators/")
		handleMemoryItem(w, r, name, memoryOperatorsDir())
	})

	// ─── Provider settings endpoints ─────────────────────────────────────────
	mux.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetSettings(w, r)
		case http.MethodPost:
			handlePostSettings(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		model := DefaultModel
		if cfg.model != "" {
			model = cfg.model
		}
		provider, localURL, localModel := globalProvider.get()
		p := PricingFor(model)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"model":       model,
			"maxTokens":   cfg.maxTokens,
			"temperature": cfg.temperature,
			"system":      cfg.system,
			"provider":    provider,
			"localURL":    localURL,
			"localModel":  localModel,
			"costIn":      p.CostIn,
			"costOut":     p.CostOut,
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

// ─── Task phase orchestrator ─────────────────────────────────────────────────

func runTaskPhase(apiKey string, cfg config, cw *contextWindow, enabledTools []string, emit func(AgentEvent), localURL string, localModel string) error {
	ts := cw.TaskState

	if localURL != "" {
		return runTaskPhaseLocal(localURL, localModel, cfg, cw, emit)
	}

	// Ensure a single sandbox directory is shared across all phases of this task.
	sandbox := ts.EnsureSandbox()

	switch ts.Phase {
	case PhasePlanning:
		agent := newPlanningAgent(apiKey, cfg, ts, sandbox)
		_, err := agent.Run(ts.Goal, nil, emit)
		if err != nil {
			ts.Error = err.Error()
			return err
		}
		if agent.PhaseResult != nil && agent.PhaseResult.Tool == "submit_plan" {
			var plan struct {
				Steps []struct {
					Description string `json:"description"`
				} `json:"steps"`
				Summary string `json:"summary"`
			}
			json.Unmarshal(agent.PhaseResult.Input, &plan)
			ts.Steps = make([]TaskStep, len(plan.Steps))
			for i, s := range plan.Steps {
				ts.Steps[i] = TaskStep{Index: i, Description: s.Description, Status: StepPending}
			}
			ts.Artifacts["plan_summary"] = plan.Summary
			ts.Phase = PhaseExecuting
			ts.Paused = true
			ts.Feedback = ""
		} else {
			ts.Error = "Planning agent did not submit a plan"
		}

	case PhaseExecuting:
		agent := newExecutingAgent(apiKey, cfg, ts, enabledTools, sandbox)
		_, err := agent.Run("Execute the plan", nil, emit)
		// Even on error (e.g. max turns), save what was done.
		if err != nil {
			ts.Error = err.Error()
		}
		// Copy step results from agent.
		if len(agent.StepResults) > 0 {
			ts.StepResults = append(ts.StepResults, agent.StepResults...)
			// Update step statuses based on results.
			for _, r := range agent.StepResults {
				for i, s := range ts.Steps {
					if s.Index == r.Index {
						ts.Steps[i].Status = r.Status
						break
					}
				}
			}
		}
		ts.Phase = PhaseValidating
		ts.Paused = true

	case PhaseValidating:
		agent := newValidatingAgent(apiKey, cfg, ts, sandbox)
		_, err := agent.Run("Validate the results", nil, emit)
		if err != nil {
			ts.Error = err.Error()
			return err
		}
		if agent.PhaseResult != nil && agent.PhaseResult.Tool == "submit_validation" {
			var val struct {
				Passed    bool   `json:"passed"`
				Feedback  string `json:"feedback"`
				NextPhase string `json:"next_phase"`
			}
			json.Unmarshal(agent.PhaseResult.Input, &val)
			ts.ValidationCount++
			ts.Artifacts["validation"] = val.Feedback
			if val.Passed {
				ts.Phase = PhaseDone
				ts.Paused = false
				// Task is done — release the shared sandbox.
				ts.CleanupSandbox()
			} else {
				ts.Feedback = val.Feedback
				if val.NextPhase == PhasePlanning {
					ts.Phase = PhasePlanning
				} else {
					ts.Phase = PhaseExecuting
				}
				ts.Paused = true
			}
		} else {
			ts.Error = "Validation agent did not submit validation"
			ts.Paused = true
		}

	case PhaseDone:
		// Nothing to do.
		return nil
	}

	// Emit updated task state.
	stateJSON, _ := json.Marshal(ts)
	emit(AgentEvent{Type: "task_state", Text: string(stateJSON)})
	return nil
}

// runTaskPhaseLocal handles task phases using a local LLM (text-only, no tools).
func runTaskPhaseLocal(localURL, localModel string, cfg config, cw *contextWindow, emit func(AgentEvent)) error {
	ts := cw.TaskState
	model := localModel
	if model == "" {
		model = cfg.model
	}

	switch ts.Phase {
	case PhasePlanning:
		system := buildPlanningPromptLocal(ts)
		localCfg := cfg
		localCfg.system = system
		msgs := []message{{Role: "user", Content: ts.Goal}}

		text, _, err := streamChatOpenAI(localURL, model, localCfg, msgs, func(token string) {
			emit(AgentEvent{Type: "text_delta", Text: token})
		})
		if err != nil {
			ts.Error = err.Error()
			return err
		}

		steps, summary := parsePlanText(text)
		ts.Steps = steps
		ts.Artifacts["plan_summary"] = summary
		ts.Phase = PhaseExecuting
		ts.Paused = true
		ts.Feedback = ""

	case PhaseExecuting:
		system := buildExecutingPromptLocal(ts)
		localCfg := cfg
		localCfg.system = system
		msgs := []message{{Role: "user", Content: "Execute the plan for: " + ts.Goal}}

		text, _, err := streamChatOpenAI(localURL, model, localCfg, msgs, func(token string) {
			emit(AgentEvent{Type: "text_delta", Text: token})
		})
		if err != nil {
			ts.Error = err.Error()
			return err
		}

		results := parseExecutionText(text, ts.Steps)
		ts.StepResults = append(ts.StepResults, results...)
		for _, r := range results {
			for i, s := range ts.Steps {
				if s.Index == r.Index {
					ts.Steps[i].Status = r.Status
					break
				}
			}
			srJSON, _ := json.Marshal(r)
			emit(AgentEvent{Type: "step_result", Text: string(srJSON)})
		}
		ts.Artifacts["exec_log"] = text
		ts.Phase = PhaseValidating
		ts.Paused = true

	case PhaseValidating:
		system := buildValidatingPromptLocal(ts)
		localCfg := cfg
		localCfg.system = system
		msgs := []message{{Role: "user", Content: "Validate the results for: " + ts.Goal}}

		text, _, err := streamChatOpenAI(localURL, model, localCfg, msgs, func(token string) {
			emit(AgentEvent{Type: "text_delta", Text: token})
		})
		if err != nil {
			ts.Error = err.Error()
			return err
		}

		passed, feedback, nextPhase := parseValidationText(text)
		ts.ValidationCount++
		ts.Artifacts["validation"] = feedback
		if passed {
			ts.Phase = PhaseDone
			ts.Paused = false
		} else {
			ts.Feedback = feedback
			if nextPhase == PhasePlanning {
				ts.Phase = PhasePlanning
			} else {
				ts.Phase = PhaseExecuting
			}
			ts.Paused = true
		}

	case PhaseDone:
		return nil
	}

	// Emit done for streaming.
	emit(AgentEvent{Type: "done"})

	// Emit updated task state.
	stateJSON, _ := json.Marshal(ts)
	emit(AgentEvent{Type: "task_state", Text: string(stateJSON)})
	return nil
}

// ─── POST /api/chat ──────────────────────────────────────────────────────────

type chatRequest struct {
	Message      string   `json:"message"`
	Session      string   `json:"session"`
	Model        string   `json:"model"`
	System       string   `json:"system"`
	MaxTokens    int      `json:"maxTokens"`
	Temperature  float64  `json:"temperature"`
	Strategy     string   `json:"strategy"`
	WindowSize   int      `json:"windowSize"`
	Profile      string   `json:"profile"`
	Project      string   `json:"project"`
	Operator     string   `json:"operator"`
	TaskMode     bool     `json:"taskMode"`
	EnabledTools []string `json:"enabledTools"`
	Invariants   []string `json:"invariants"`
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

	var apiRequestText string
	emit := func(ev AgentEvent) {
		if ev.Type == "api_request" {
			apiRequestText = ev.Text
		}
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
	if req.Profile != "" && cw.Settings.Profile == "" {
		cw.Settings.Profile = req.Profile
	}
	if req.Project != "" && cw.Settings.Project == "" {
		cw.Settings.Project = req.Project
	}
	if req.Operator != "" && cw.Settings.Operator == "" {
		cw.Settings.Operator = req.Operator
	}

	// Inject memory layers into system prompt.
	cfg.system = buildFullSystemPrompt(cfg, cw.Settings)

	// Restore cumulative stats from session (or start fresh).
	stats := cw.Stats.toTokenStats()
	stats.Model = cfg.model
	if stats.Model == "" {
		stats.Model = DefaultModel
	}

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

	// Read current provider settings.
	provider, localURL, localModel := globalProvider.get()

	var result string
	var chatErr error

	// Task mode is available for both providers.
	isTaskMode := req.TaskMode || (cw.TaskState != nil && cw.TaskState.Phase != PhaseDone)

	if isTaskMode {
		if cw.TaskState == nil {
			cw.TaskState = &TaskState{
				Goal:       req.Message,
				Phase:      PhasePlanning,
				Artifacts:  make(map[string]string),
				Invariants: req.Invariants,
			}
		}

		// If paused and user sent a message, treat as continue with optional feedback.
		if cw.TaskState.Paused {
			cw.TaskState.Paused = false
			if req.Message != "" && req.Message != cw.TaskState.Goal {
				cw.TaskState.Feedback = req.Message
			}
		}

		// Pass localURL/localModel for local provider, empty strings for Claude.
		taskLocalURL := ""
		taskLocalModel := ""
		if provider == "local" {
			taskLocalURL = localURL
			taskLocalModel = localModel
		}

		chatErr = runTaskPhase(apiKey, cfg, cw, req.EnabledTools, emit, taskLocalURL, taskLocalModel)
		if chatErr != nil {
			sseWrite(w, map[string]any{"type": "error", "message": chatErr.Error()})
		}

		result = cw.TaskState.Phase // marker for session save
	} else if provider == "local" {
		// Local LLM path: stream directly via OpenAI-compatible API.
		model := localModel
		if model == "" {
			model = cfg.model
		}
		localCfg := cfg
		localCfg.model = model

		result, _, chatErr = streamChatOpenAI(localURL, model, localCfg, compressed, func(token string) {
			sseWrite(w, AgentEvent{Type: "text_delta", Text: token})
		})
		if chatErr != nil {
			sseWrite(w, map[string]any{"type": "error", "message": chatErr.Error()})
		}
		// Emit done event.
		sseWrite(w, AgentEvent{Type: "done"})
	} else {
		// Claude API path.
		agent := newAgent(apiKey, cfg)

		result, chatErr = agent.Run(req.Message, compressed, emit)
		agent.Cleanup()
		if chatErr != nil {
			sseWrite(w, map[string]any{"type": "error", "message": chatErr.Error()})
		}

		// Accumulate agent stats into session lifetime stats.
		stats.TotalInput += agent.Stats.TotalInput
		stats.TotalOutput += agent.Stats.TotalOutput
		stats.Exchanges += agent.Stats.Exchanges

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

		// Auto-update profile/project memory.
		if cw.Settings.Profile != "" || cw.Settings.Project != "" {
			if err := maybeUpdateMemory(apiKey, cw, &stats); err != nil {
				fmt.Fprintf(os.Stderr, "[memory_update] error: %v\n", err)
			} else {
				sseWrite(w, map[string]any{"type": "memory_updated"})
			}
		}
	}

	// Save to session (append to raw Messages, not compressed).
	if req.Message != "" {
		appendMessage(cw, message{Role: "user", Content: req.Message, ApiRequest: apiRequestText})
	}
	appendMessage(cw, message{Role: "assistant", Content: result})

	cw.Settings.Model = cfg.model
	cw.Settings.Temperature = cfg.temperature
	cw.Settings.MaxTokens = cfg.maxTokens
	cw.Settings.System = cfg.system
	cw.Stats = statsFromToken(stats)
	_ = saveSessionCW(req.Session, cw)
}

// ─── GET /api/sessions ───────────────────────────────────────────────────────

func handleListSessions(w http.ResponseWriter, r *http.Request) {
	list, err := listSessions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
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

	resp := map[string]any{
		"messages":     activeMessages(cw),
		"summary":      cw.Summary,
		"settings":     cw.Settings,
		"stats":        cw.Stats,
		"facts":        cw.Facts,
		"branches":     branchInfos,
		"activeBranch": cw.ActiveBranch,
	}
	if cw.TaskState != nil {
		resp["taskState"] = cw.TaskState
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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

// ─── Memory endpoints ───────────────────────────────────────────────────────

func handleMemoryList(w http.ResponseWriter, r *http.Request, dir string, saveFn func(string, string) error) {
	switch r.Method {
	case http.MethodGet:
		names, err := listMemoryFiles(dir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(names)

	case http.MethodPost:
		var req struct {
			Name    string `json:"name"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := saveFn(req.Name, req.Content); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "created", "name": req.Name})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleMemoryItem(w http.ResponseWriter, r *http.Request, name, dir string) {
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		content, err := getMemoryFile(dir, name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"name": name, "content": content})

	case http.MethodPut:
		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := saveMemoryFile(dir, name, req.Content); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "updated", "name": name})

	case http.MethodDelete:
		if err := deleteMemoryFile(dir, name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// ─── Settings endpoints ──────────────────────────────────────────────────────

func handleGetSettings(w http.ResponseWriter, r *http.Request) {
	provider, localURL, localModel := globalProvider.get()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"provider":   provider,
		"localURL":   localURL,
		"localModel": localModel,
	})
}

func handlePostSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider   string `json:"provider"`
		LocalURL   string `json:"localURL"`
		LocalModel string `json:"localModel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate provider value.
	if req.Provider != "claude" && req.Provider != "local" {
		http.Error(w, `provider must be "claude" or "local"`, http.StatusBadRequest)
		return
	}

	// When provider is local, localURL is required.
	if req.Provider == "local" && req.LocalURL == "" {
		http.Error(w, "localURL is required when provider is local", http.StatusBadRequest)
		return
	}

	globalProvider.set(req.Provider, req.LocalURL, req.LocalModel)

	provider, localURL, localModel := globalProvider.get()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":     "updated",
		"provider":   provider,
		"localURL":   localURL,
		"localModel": localModel,
	})
}
