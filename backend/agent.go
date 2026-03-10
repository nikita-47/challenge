package main

import (
	"bytes"
	"context"
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

	"github.com/mark3labs/mcp-go/mcp"
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

// ─── PhaseResult — filled when a phase-ending tool is called ─────────────────

type PhaseResult struct {
	Tool  string
	Input json.RawMessage
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
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	InputSchema  any            `json:"input_schema"`
	CacheControl map[string]any `json:"cache_control,omitempty"`
}

type apiResponse struct {
	Content    []contentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Usage      struct {
		InputTokens        int `json:"input_tokens"`
		OutputTokens       int `json:"output_tokens"`
		CacheCreationInput int `json:"cache_creation_input_tokens"`
		CacheReadInput     int `json:"cache_read_input_tokens"`
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
	workDir     string       // sandbox directory for tool execution
	mcpMgr      *MCPManager  // optional MCP manager for routing MCP tool calls
	PhaseResult *PhaseResult // filled when submit_plan or submit_validation is called
	StepResults []StepResult // accumulated from report_step calls
}

// ─── Constructors ─────────────────────────────────────────────────────────────

func newAgent(apiKey string, cfg config) *Agent {
	model := DefaultModel
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
		Stats:       tokenStats{Model: model},
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

// ─── Phase-specific agent constructors ───────────────────────────────────────

func newProposingAgent(apiKey string, cfg config, ts *TaskState, sandboxDir string) *Agent {
	a := newAgent(apiKey, cfg)
	a.system = buildProposingPrompt(ts)
	a.tools = []toolDef{submitPhasesTool()}
	a.maxTurns = 3
	a.workDir = sandboxDir
	return a
}

func newPlanningAgent(apiKey string, cfg config, ts *TaskState, sandboxDir string) *Agent {
	a := newAgent(apiKey, cfg)
	a.system = buildPlanningPrompt(ts)
	a.tools = []toolDef{submitPlanTool()}
	a.maxTurns = 5
	a.workDir = sandboxDir
	return a
}

func newExecutingAgent(apiKey string, cfg config, ts *TaskState, enabledTools []string, sandboxDir string) *Agent {
	a := newAgent(apiKey, cfg)
	a.system = buildExecutingPrompt(ts)
	// Base tools for execution: run_shell, read_file, report_step.
	base := defaultTools()
	if len(enabledTools) > 0 {
		base = filterTools(base, enabledTools)
	}
	// Always include report_step.
	a.tools = append(base, reportStepTool())
	a.maxTurns = 12
	a.workDir = sandboxDir
	return a
}

func newValidatingAgent(apiKey string, cfg config, ts *TaskState, sandboxDir string) *Agent {
	a := newAgent(apiKey, cfg)
	a.system = buildValidatingPrompt(ts)
	a.tools = append(defaultTools(), submitValidationTool())
	a.maxTurns = 6
	a.workDir = sandboxDir
	return a
}

// ─── Phase-specific tools ─────────────────────────────────────────────────────

func submitPlanTool() toolDef {
	return toolDef{
		Name:        "submit_plan",
		Description: "Submit the structured plan for the task. Call this when you have analyzed the goal and created a concrete plan.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"steps": map[string]any{
					"type":        "array",
					"description": "List of concrete steps to execute",
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
				"summary": map[string]any{
					"type":        "string",
					"description": "Brief summary of the overall approach",
				},
			},
			"required": []string{"steps", "summary"},
		},
	}
}

func submitPhasesTool() toolDef {
	return toolDef{
		Name:        "submit_phases",
		Description: "Submit the proposed pipeline of phases for the task. Call this after analyzing the goal.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"phases": map[string]any{
					"type":        "array",
					"description": "Ordered list of phases for the pipeline",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"name": map[string]any{
								"type":        "string",
								"description": "Short display name (e.g. Research, Plan, Implement, Validate)",
							},
							"type": map[string]any{
								"type":        "string",
								"description": "Phase type: planning, executing, or validating",
								"enum":        []string{"planning", "executing", "validating"},
							},
							"description": map[string]any{
								"type":        "string",
								"description": "What this phase should accomplish",
							},
						},
						"required": []string{"name", "type", "description"},
					},
				},
				"summary": map[string]any{
					"type":        "string",
					"description": "Brief description of the overall approach",
				},
			},
			"required": []string{"phases", "summary"},
		},
	}
}

func reportStepTool() toolDef {
	return toolDef{
		Name:        "report_step",
		Description: "Report the result of completing or failing a step. Call this after each step.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"step_index": map[string]any{
					"type":        "integer",
					"description": "0-based index of the step",
				},
				"status": map[string]any{
					"type":        "string",
					"description": "Result status: completed or failed",
					"enum":        []string{"completed", "failed"},
				},
				"output": map[string]any{
					"type":        "string",
					"description": "What was done and what the result was",
				},
			},
			"required": []string{"step_index", "status", "output"},
		},
	}
}

func submitValidationTool() toolDef {
	return toolDef{
		Name:        "submit_validation",
		Description: "Submit the validation result. Call this after verifying the execution results.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"passed": map[string]any{
					"type":        "boolean",
					"description": "True if the goal was successfully achieved, false otherwise",
				},
				"feedback": map[string]any{
					"type":        "string",
					"description": "Summary of what was verified, or description of what failed and needs fixing",
				},
				"next_phase": map[string]any{
					"type":        "string",
					"description": "If passed=false, which phase to go back to: 'executing' or 'planning'",
					"enum":        []string{"executing", "planning"},
				},
			},
			"required": []string{"passed"},
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

// ─── Sandbox ──────────────────────────────────────────────────────────────────

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

// ─── Tool execution ──────────────────────────────────────────────────────────

func executeTool(name string, rawInput json.RawMessage, workDir string, mcpMgr *MCPManager) (string, bool) {
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
		// Truncate long tool output to prevent token bloat in agent history.
		if len(result) > 4000 {
			result = result[:2000] + "\n\n... [truncated " + strconv.Itoa(len(result)-4000) + " chars] ...\n\n" + result[len(result)-2000:]
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
		content := string(data)
		// Truncate long file content to prevent token bloat in agent history.
		if len(content) > 8000 {
			content = content[:4000] + "\n\n... [truncated " + strconv.Itoa(len(content)-8000) + " chars] ...\n\n" + content[len(content)-4000:]
		}
		return content, false

	default:
		// Attempt MCP routing for namespaced tool names ("server__toolname").
		if mcpMgr != nil {
			server, tool, ok := parseMCPToolName(name)
			if ok {
				var args map[string]any
				if err := json.Unmarshal(rawInput, &args); err != nil {
					return "invalid MCP tool input: " + err.Error(), true
				}
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				mcpResult, err := mcpMgr.CallTool(ctx, server, tool, args)
				if err != nil {
					return "MCP tool error: " + err.Error(), true
				}
				// Collect all TextContent blocks into a single string.
				var sb strings.Builder
				for _, c := range mcpResult.Content {
					if tc, ok := c.(mcp.TextContent); ok {
						if sb.Len() > 0 {
							sb.WriteString("\n")
						}
						sb.WriteString(tc.Text)
					}
				}
				return sb.String(), mcpResult.IsError
			}
		}
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
		// Add cache_control to the last tool for prompt caching.
		tools := make([]toolDef, len(a.tools))
		copy(tools, a.tools)
		tools[len(tools)-1].CacheControl = map[string]any{"type": "ephemeral"}
		payload["tools"] = tools
	}
	sys := a.system
	if a.workDir != "" {
		sys += fmt.Sprintf("\n\nYour working directory is: %s\nAll shell commands execute there. Files you create will be in this sandbox.", a.workDir)
	}
	if sys != "" {
		// Send system as array of content blocks with cache_control for prompt caching.
		payload["system"] = []map[string]any{
			{
				"type":          "text",
				"text":          sys,
				"cache_control": map[string]any{"type": "ephemeral"},
			},
		}
	}
	if a.temperature >= 0 {
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
	if resp != nil {
		if s := resp.Header.Get("retry-after"); s != "" {
			if secs, err := strconv.Atoi(s); err == nil {
				return time.Duration(secs) * time.Second
			}
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
		// Inject progress into goal message so the model sees completed steps
		// after old history has been compacted.
		if turn > 1 && len(a.StepResults) > 0 {
			a.history[0].Content = goal + "\n\nCompleted so far:\n" + formatStepProgress(a.StepResults)
		}

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
		usage := tokenUsage{
			InputTokens:        resp.Usage.InputTokens,
			OutputTokens:       resp.Usage.OutputTokens,
			CacheCreationInput: resp.Usage.CacheCreationInput,
			CacheReadInput:     resp.Usage.CacheReadInput,
		}
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
		phaseComplete := false

		for _, block := range resp.Content {
			if block.Type == "text" && block.Text != "" {
				emit(AgentEvent{Type: "thinking", Text: block.Text})
			}
			if block.Type != "tool_use" {
				continue
			}

			emit(AgentEvent{Type: "tool_call", Tool: block.Name, Input: block.Input})

			// Handle phase-ending tools: submit_plan, submit_validation, submit_phases.
			if block.Name == "submit_plan" || block.Name == "submit_validation" || block.Name == "submit_phases" {
				a.PhaseResult = &PhaseResult{Tool: block.Name, Input: block.Input}
				emit(AgentEvent{Type: "tool_result", Tool: block.Name, Output: "Accepted.", IsError: false})
				toolResults = append(toolResults, contentBlock{
					Type:      "tool_result",
					ToolUseID: block.ID,
					Content:   "Accepted.",
					IsError:   false,
				})
				phaseComplete = true
				continue
			}

			// Handle report_step: accumulate results, emit event, do NOT break.
			if block.Name == "report_step" {
				var input struct {
					StepIndex int    `json:"step_index"`
					Status    string `json:"status"`
					Output    string `json:"output"`
				}
				if err := json.Unmarshal(block.Input, &input); err == nil {
					sr := StepResult{
						Index:  input.StepIndex,
						Status: input.Status,
						Output: input.Output,
					}
					a.StepResults = append(a.StepResults, sr)
					srJSON, _ := json.Marshal(sr)
					emit(AgentEvent{Type: "step_result", Text: string(srJSON)})
				}
				emit(AgentEvent{Type: "tool_result", Tool: block.Name, Output: "Noted.", IsError: false})
				toolResults = append(toolResults, contentBlock{
					Type:      "tool_result",
					ToolUseID: block.ID,
					Content:   "Noted.",
					IsError:   false,
				})
				continue
			}

			// Default tool execution (run_shell, read_file, or MCP).
			result, isError := executeTool(block.Name, block.Input, a.workDir, a.mcpMgr)
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

		// Compact old history to reduce input tokens on next turn.
		a.compactHistory()

		// If a phase-ending tool was called, exit the loop.
		if phaseComplete {
			statsCopy := a.Stats
			emit(AgentEvent{Type: "done", Turn: turn, Stats: &statsCopy})
			return "", nil
		}
	}

	statsCopy := a.Stats
	emit(AgentEvent{Type: "done", Stats: &statsCopy})
	return "", fmt.Errorf("agent reached max turns (%d) without completing", a.maxTurns)
}

// formatStepProgress formats completed step results as a short summary for the goal message.
func formatStepProgress(results []StepResult) string {
	var b strings.Builder
	for _, r := range results {
		status := "✓"
		if r.Status == "failed" {
			status = "✗"
		}
		output := r.Output
		if len(output) > 80 {
			output = output[:80] + "…"
		}
		b.WriteString(fmt.Sprintf("  %s Step %d: %s\n", status, r.Index, output))
	}
	return b.String()
}

// ─── History compaction ──────────────────────────────────────────────────────

// compactHistory compresses old tool results and text blocks to reduce input tokens.
// Keeps the first message (goal) and the last keepRecent messages intact.
func (a *Agent) compactHistory() {
	const keepRecent = 4 // last 4 messages (2 turns: assistant + tool_results)
	cutoff := len(a.history) - keepRecent
	if cutoff <= 1 {
		return // always preserve first message (goal)
	}

	for i := 1; i < cutoff; i++ {
		blocks, ok := a.history[i].Content.([]contentBlock)
		if !ok {
			// String content in assistant/user messages — truncate if long.
			if s, ok := a.history[i].Content.(string); ok && len(s) > 200 {
				a.history[i].Content = s[:200] + "…"
			}
			continue
		}
		for j := range blocks {
			switch blocks[j].Type {
			case "tool_result":
				blocks[j].Content = compactToolResult(blocks[j].Content, blocks[j].ToolUseID)
			case "text":
				if len(blocks[j].Text) > 200 {
					blocks[j].Text = blocks[j].Text[:200] + "…"
				}
			}
		}
		a.history[i].Content = blocks
	}
}

// compactToolResult compresses a tool result string into a short summary.
func compactToolResult(content string, toolUseID string) string {
	// Already compacted (starts with "[").
	if strings.HasPrefix(content, "[") {
		return content
	}
	// "Noted." from report_step — keep as-is.
	if content == "Noted." || content == "Accepted." {
		return content
	}

	lines := strings.SplitN(content, "\n", 2)
	firstLine := strings.TrimSpace(lines[0])
	if len(firstLine) > 80 {
		firstLine = firstLine[:80]
	}

	lineCount := strings.Count(content, "\n") + 1
	return fmt.Sprintf("[output: %s (%d lines)]", firstLine, lineCount)
}

func truncate(s string, maxLen int) string {
	// Replace newlines for single-line display.
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
