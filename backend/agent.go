package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ─── Agent event (for decoupled IO) ──────────────────────────────────────────

type AgentEvent struct {
	Type    string          `json:"type"`              // "turn", "thinking", "tool_call", "tool_result", "text", "done", "error", "usage"
	Turn    int             `json:"turn,omitempty"`
	MaxTurn int             `json:"max_turn,omitempty"`
	Tool    string          `json:"tool,omitempty"`
	Input   json.RawMessage `json:"input,omitempty"`
	Output  string          `json:"output,omitempty"`
	IsError bool            `json:"is_error,omitempty"`
	Text    string          `json:"text,omitempty"`
	Usage   *tokenUsage     `json:"usage,omitempty"`
	Stats   *tokenStats     `json:"stats,omitempty"`
}

// ─── Agent types ─────────────────────────────────────────────────────────────

type contentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

type toolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"input_schema"`
}

type apiResponse struct {
	Content    []contentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type Agent struct {
	apiKey      string
	model       string
	maxTurns    int
	maxTokens   int
	system      string
	temperature float64
	tools       []toolDef
	history     []message
	Stats       tokenStats
	TaskState   *TaskState
	workDir     string // sandbox directory for tool execution
}

// ─── Constructor ─────────────────────────────────────────────────────────────

func newAgent(apiKey string, cfg config) *Agent {
	model := "claude-sonnet-4-5-20250929"
	if cfg.model != "" {
		model = cfg.model
	}
	maxTokens := 4096
	if cfg.maxTokens > 0 {
		maxTokens = cfg.maxTokens
	}
	return &Agent{
		apiKey:      apiKey,
		model:       model,
		maxTurns:    10,
		maxTokens:   maxTokens,
		system:      cfg.system,
		temperature: cfg.temperature,
	}
}

func newAgentWithTools(apiKey string, cfg config) *Agent {
	a := newAgent(apiKey, cfg)
	a.tools = defaultTools()
	a.workDir = createSandbox()
	return a
}

func newAgentWithTaskState(apiKey string, cfg config, ts *TaskState) *Agent {
	a := newAgent(apiKey, cfg)
	a.tools = append(defaultTools(), taskStateTool())
	a.maxTurns = 25
	a.TaskState = ts
	a.workDir = createSandbox()
	return a
}

// filterTools returns only the tools whose Name is in the enabled list.
func filterTools(tools []toolDef, enabled []string) []toolDef {
	set := make(map[string]struct{}, len(enabled))
	for _, name := range enabled {
		set[name] = struct{}{}
	}
	var result []toolDef
	for _, t := range tools {
		if _, ok := set[t.Name]; ok {
			result = append(result, t)
		}
	}
	return result
}

// newAgentWithTaskStateFiltered creates a task-mode agent with optional tool filtering.
// If enabledTools is non-empty, only those default tools are included; update_task_state
// is always added regardless. If enabledTools is empty, all default tools are included.
func newAgentWithTaskStateFiltered(apiKey string, cfg config, ts *TaskState, enabledTools []string) *Agent {
	a := newAgent(apiKey, cfg)
	base := defaultTools()
	if len(enabledTools) > 0 {
		base = filterTools(base, enabledTools)
	}
	a.tools = append(base, taskStateTool())
	a.maxTurns = 25
	a.TaskState = ts
	a.workDir = createSandbox()
	return a
}

const sandboxDir = ".sandbox"

func createSandbox() string {
	dir, err := os.MkdirTemp(sandboxDir, "agent-")
	if err != nil {
		// Fallback: create .sandbox first, retry.
		os.MkdirAll(sandboxDir, 0755)
		dir, err = os.MkdirTemp(sandboxDir, "agent-")
		if err != nil {
			// Last resort: system temp.
			dir, _ = os.MkdirTemp("", "agent-")
		}
	}
	return dir
}

func (a *Agent) Cleanup() {
	if a.workDir != "" {
		os.RemoveAll(a.workDir)
	}
}

func taskStateTool() toolDef {
	return toolDef{
		Name:        "update_task_state",
		Description: "Update the task state machine. Actions: set_plan, start_step, complete_step, fail_step, validate, done, pause, resume.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"description": "The action to perform: set_plan, start_step, complete_step, fail_step, validate, done, pause, resume",
					"enum":        []string{"set_plan", "start_step", "complete_step", "fail_step", "validate", "done", "pause", "resume"},
				},
				"steps": map[string]any{
					"type":        "array",
					"description": "Steps for set_plan action. Each step has a description.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"description": map[string]any{
								"type":        "string",
								"description": "What this step does",
							},
						},
						"required": []string{"description"},
					},
				},
				"step_index": map[string]any{
					"type":        "integer",
					"description": "Index of the step to start/complete/fail (0-based)",
				},
				"result": map[string]any{
					"type":        "string",
					"description": "Result description for complete_step",
				},
				"error": map[string]any{
					"type":        "string",
					"description": "Error description for fail_step",
				},
				"expected_action": map[string]any{
					"type":        "string",
					"description": "What you plan to do next (shown to user)",
				},
			},
			"required": []string{"action"},
		},
	}
}

func defaultTools() []toolDef {
	return []toolDef{
		{
			Name:        "run_shell",
			Description: "Run a shell command and return stdout+stderr. Use this to explore the filesystem, run scripts, count lines, etc.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "The shell command to execute",
					},
				},
				"required": []string{"command"},
			},
		},
		{
			Name:        "read_file",
			Description: "Read the contents of a file and return it as text.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "The file path to read",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

// ─── Tool execution ──────────────────────────────────────────────────────────

func executeTool(name string, rawInput json.RawMessage, workDir string) (string, bool) {
	switch name {
	case "run_shell":
		var input struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return "invalid input: " + err.Error(), true
		}
		cmd := exec.Command("/bin/sh", "-c", input.Command)
		if workDir != "" {
			cmd.Dir = workDir
		}
		out, err := cmd.CombinedOutput()
		result := string(out)
		if err != nil {
			result += "\nexit error: " + err.Error()
		}
		return result, false

	case "read_file":
		var input struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return "invalid input: " + err.Error(), true
		}
		// Resolve relative paths against workDir.
		path := input.Path
		if workDir != "" && !filepath.IsAbs(path) {
			path = filepath.Join(workDir, path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "error reading file: " + err.Error(), true
		}
		return string(data), false

	default:
		return "unknown tool: " + name, true
	}
}

// ─── API call (non-streaming) ────────────────────────────────────────────────

func (a *Agent) buildPayload() map[string]any {
	payload := map[string]any{
		"model":      a.model,
		"max_tokens": a.maxTokens,
		"messages":   a.history,
	}
	if len(a.tools) > 0 {
		payload["tools"] = a.tools
	}
	sys := a.system
	if a.workDir != "" {
		sys += fmt.Sprintf("\n\nYour working directory is: %s\nAll shell commands execute there. Files you create will be in this sandbox.", a.workDir)
	}
	if a.TaskState != nil {
		sys += a.TaskState.SystemPromptSection()
	}
	if sys != "" {
		payload["system"] = sys
	}
	if a.temperature > 0 {
		payload["temperature"] = a.temperature
	}
	return payload
}

func (a *Agent) callAPI() (*apiResponse, error) {
	payload := a.buildPayload()
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, &apiError{StatusCode: resp.StatusCode, Body: string(errBody), RetryAfter: retryAfter(resp, 0)}
	}

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}
	return &result, nil
}

// apiError carries status code and retry info for rate limit handling.
type apiError struct {
	StatusCode int
	Body       string
	RetryAfter time.Duration
}

func (e *apiError) Error() string {
	return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Body)
}

// retryAfter returns how long to wait before retrying a 429 response.
func retryAfter(resp *http.Response, attempt int) time.Duration {
	if s := resp.Header.Get("retry-after"); s != "" {
		if secs, err := strconv.Atoi(s); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	backoff := []time.Duration{5 * time.Second, 15 * time.Second, 30 * time.Second}
	if attempt < len(backoff) {
		return backoff[attempt]
	}
	return 30 * time.Second
}

func (a *Agent) callAPIWithRetry(turn int, emit func(AgentEvent)) (*apiResponse, error) {
	const maxRetries = 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := a.callAPI()
		if err == nil {
			return resp, nil
		}
		var ae *apiError
		if errors.As(err, &ae) && ae.StatusCode == 429 && attempt < maxRetries {
			wait := retryAfter(nil, attempt)
			if ae.RetryAfter > 0 {
				wait = ae.RetryAfter
			}
			emit(AgentEvent{Type: "thinking", Text: fmt.Sprintf("Rate limited, retrying in %s... (%d/%d)", wait, attempt+1, maxRetries)})
			time.Sleep(wait)
			continue
		}
		return nil, err
	}
	return nil, fmt.Errorf("rate limit: max retries exceeded")
}

// ─── Agentic loop ────────────────────────────────────────────────────────────

func (a *Agent) Run(goal string, chatHistory []message, emit func(AgentEvent)) (string, error) {
	if emit == nil {
		emit = func(AgentEvent) {}
	}

	emit(AgentEvent{Type: "text", Text: goal})

	// Initialize history: chat context + goal.
	if len(chatHistory) > 0 {
		a.history = make([]message, len(chatHistory))
		copy(a.history, chatHistory)
		a.history = append(a.history, message{Role: "user", Content: goal})
	} else {
		a.history = []message{{Role: "user", Content: goal}}
	}

	for turn := 1; turn <= a.maxTurns; turn++ {
		emit(AgentEvent{Type: "turn", Turn: turn, MaxTurn: a.maxTurns})

		if turn == 1 {
			payload := a.buildPayload()
			payloadJSON, _ := json.MarshalIndent(payload, "", "  ")
			emit(AgentEvent{Type: "api_request", Text: string(payloadJSON)})
		}

		resp, err := a.callAPIWithRetry(turn, emit)
		if err != nil {
			emit(AgentEvent{Type: "error", Text: fmt.Sprintf("turn %d: %v", turn, err)})
			return "", fmt.Errorf("turn %d: %w", turn, err)
		}

		// Track tokens.
		usage := tokenUsage{InputTokens: resp.Usage.InputTokens, OutputTokens: resp.Usage.OutputTokens}
		a.Stats.Add(usage)
		emit(AgentEvent{Type: "usage", Usage: &usage})

		// Append assistant response to history.
		a.history = append(a.history, message{Role: "assistant", Content: resp.Content})

		// If end_turn, emit text and return.
		if resp.StopReason == "end_turn" {
			var text strings.Builder
			for _, block := range resp.Content {
				if block.Type == "text" {
					emit(AgentEvent{Type: "text_delta", Text: block.Text})
					text.WriteString(block.Text)
				}
			}
			statsCopy := a.Stats
			emit(AgentEvent{Type: "done", Turn: turn, Stats: &statsCopy})
			return text.String(), nil
		}

		// Process tool_use blocks.
		var toolResults []contentBlock
		paused := false
		for _, block := range resp.Content {
			if block.Type == "text" && block.Text != "" {
				emit(AgentEvent{Type: "thinking", Text: block.Text})
			}
			if block.Type != "tool_use" {
				continue
			}

			// Handle task state updates inline.
			if block.Name == "update_task_state" && a.TaskState != nil {
				emit(AgentEvent{Type: "tool_call", Tool: block.Name, Input: block.Input})
				result, err := a.TaskState.applyAction(block.Input)
				isError := err != nil
				output := result
				if isError {
					output = err.Error()
				}
				emit(AgentEvent{Type: "tool_result", Tool: block.Name, Output: output, IsError: isError})

				// Emit task_state event with current state.
				stateJSON, _ := json.Marshal(a.TaskState)
				emit(AgentEvent{Type: "task_state", Text: string(stateJSON)})

				toolResults = append(toolResults, contentBlock{
					Type:      "tool_result",
					ToolUseID: block.ID,
					Content:   output,
					IsError:   isError,
				})

				if a.TaskState.Phase == PhasePaused {
					paused = true
				}
				continue
			}

			emit(AgentEvent{Type: "tool_call", Tool: block.Name, Input: block.Input})

			result, isError := executeTool(block.Name, block.Input, a.workDir)
			emit(AgentEvent{Type: "tool_result", Tool: block.Name, Output: result, IsError: isError})

			toolResults = append(toolResults, contentBlock{
				Type:      "tool_result",
				ToolUseID: block.ID,
				Content:   result,
				IsError:   isError,
			})
		}

		// Append tool results as user message.
		if len(toolResults) > 0 {
			a.history = append(a.history, message{Role: "user", Content: toolResults})
		}

		// If task was paused, return early to save state.
		if paused {
			statsCopy := a.Stats
			emit(AgentEvent{Type: "done", Turn: turn, Stats: &statsCopy})
			return "Task paused. Use /resume to continue.", nil
		}

		// If task is done, return early.
		if a.TaskState != nil && a.TaskState.Phase == PhaseDone {
			var text strings.Builder
			for _, block := range resp.Content {
				if block.Type == "text" {
					text.WriteString(block.Text)
				}
			}
			statsCopy := a.Stats
			emit(AgentEvent{Type: "done", Turn: turn, Stats: &statsCopy})
			result := text.String()
			if result == "" {
				result = "Task completed."
			}
			return result, nil
		}
	}

	statsCopy := a.Stats
	emit(AgentEvent{Type: "done", Stats: &statsCopy})
	return "", fmt.Errorf("agent reached max turns (%d) without completing", a.maxTurns)
}

func truncate(s string, maxLen int) string {
	// Replace newlines for single-line display.
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
