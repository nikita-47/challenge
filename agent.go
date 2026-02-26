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
	apiKey   string
	model    string
	maxTurns int
	tools    []toolDef
	history  []message
	Stats    tokenStats
}

// ─── Constructor ─────────────────────────────────────────────────────────────

func newAgent(apiKey string) *Agent {
	return &Agent{
		apiKey:   apiKey,
		model:    "claude-sonnet-4-5-20250929",
		maxTurns: 10,
		tools:    defaultTools(),
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

func (a *Agent) callAPI() (*apiResponse, error) {
	body, _ := json.Marshal(map[string]any{
		"model":      a.model,
		"max_tokens": 4096,
		"messages":   a.history,
		"tools":      a.tools,
	})

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

func (a *Agent) Run(goal string, chatHistory []message) (string, error) {
	fmt.Printf("\033[1m[Agent] Goal: %s\033[0m\n", goal)

	// Initialize history: chat context + goal.
	if len(chatHistory) > 0 {
		a.history = make([]message, len(chatHistory))
		copy(a.history, chatHistory)
		a.history = append(a.history, message{Role: "user", Content: goal})
	} else {
		a.history = []message{{Role: "user", Content: goal}}
	}

	for turn := 1; turn <= a.maxTurns; turn++ {
		fmt.Printf("\033[2m[Agent] Turn %d/%d — calling API...\033[0m\n", turn, a.maxTurns)

		resp, err := a.callAPI()
		if err != nil {
			return "", fmt.Errorf("turn %d: %w", turn, err)
		}

		// Track tokens.
		usage := tokenUsage{InputTokens: resp.Usage.InputTokens, OutputTokens: resp.Usage.OutputTokens}
		a.Stats.Add(usage)
		fmt.Printf("\033[2m[Agent] %s\033[0m\n", formatTokenUsage(usage))

		// Append assistant response to history.
		a.history = append(a.history, message{Role: "assistant", Content: resp.Content})

		// If end_turn, extract text and return.
		if resp.StopReason == "end_turn" {
			var text strings.Builder
			for _, block := range resp.Content {
				if block.Type == "text" {
					text.WriteString(block.Text)
				}
			}
			fmt.Printf("\033[1m[Agent] Done after %d turn(s). %s\033[0m\n\n", turn, a.Stats.FormatTotal())
			return text.String(), nil
		}

		// Process tool_use blocks.
		var toolResults []contentBlock
		for _, block := range resp.Content {
			if block.Type == "text" && block.Text != "" {
				fmt.Printf("\033[2m[Agent] Thinking: %s\033[0m\n", truncate(block.Text, 100))
			}
			if block.Type != "tool_use" {
				continue
			}

			// Print tool call info.
			fmt.Printf("\033[33m[Agent] Tool: %s\033[0m\n", block.Name)
			if block.Name == "run_shell" {
				var input struct{ Command string }
				json.Unmarshal(block.Input, &input)
				fmt.Printf("\033[33m[Agent]   $ %s\033[0m\n", input.Command)
			} else if block.Name == "read_file" {
				var input struct{ Path string }
				json.Unmarshal(block.Input, &input)
				fmt.Printf("\033[33m[Agent]   path: %s\033[0m\n", input.Path)
			}

			result, isError := executeTool(block.Name, block.Input)
			fmt.Printf("\033[2m[Agent]   Result: %s\033[0m\n", truncate(result, 200))

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

	fmt.Printf("\033[1m[Agent] %s\033[0m\n", a.Stats.FormatTotal())
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
