package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
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
		tools:       defaultTools(),
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

func executeTool(name string, rawInput json.RawMessage) (string, bool) {
	switch name {
	case "run_shell":
		var input struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return "invalid input: " + err.Error(), true
		}
		cmd := exec.Command("/bin/sh", "-c", input.Command)
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
		data, err := os.ReadFile(input.Path)
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
		"tools":      a.tools,
	}
	if a.system != "" {
		payload["system"] = a.system
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
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errBody)
	}

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}
	return &result, nil
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

		resp, err := a.callAPI()
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
		for _, block := range resp.Content {
			if block.Type == "text" && block.Text != "" {
				emit(AgentEvent{Type: "thinking", Text: block.Text})
			}
			if block.Type != "tool_use" {
				continue
			}

			emit(AgentEvent{Type: "tool_call", Tool: block.Name, Input: block.Input})

			result, isError := executeTool(block.Name, block.Input)
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
