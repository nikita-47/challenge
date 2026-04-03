package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

const filesSystemPrompt = `You are a file assistant for a software project. You help users explore, search, analyze, and modify project files using the available tools.

Capabilities:
- Search for patterns/usages across the codebase (dev_grep)
- Browse directory structure (dev_list_files)
- Read file contents (dev_read_file)
- Create or modify files (dev_write_file)
- View git info (dev_git_log, dev_git_status, dev_git_branch)

Guidelines:
- Actively use tools to gather context before answering
- Use dev_grep to find relevant files, then dev_read_file for details
- When generating files (changelog, README, etc.), always save with dev_write_file
- Be thorough: check multiple locations, cross-reference results
- Provide a clear summary of findings and actions taken
- Always respond in the same language as the user's message`

const maxFilesTurns = 10

func handleFilesChat(w http.ResponseWriter, r *http.Request, apiKey string, mcpMgr *MCPManager) {
	var req struct {
		Message string `json:"message"`
		History []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"history"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	sseSetup(w)

	// Check devtools MCP is connected.
	if mcpMgr == nil {
		sseWrite(w, map[string]any{"type": "error", "message": "MCP manager not available"})
		sseWrite(w, map[string]any{"type": "done"})
		return
	}

	// Collect devtools tool definitions.
	allDefs := mcpMgr.GetToolDefs(nil)
	var tools []toolDef
	for _, d := range allDefs {
		if strings.HasPrefix(d.Name, "devtools__") {
			tools = append(tools, d)
		}
	}
	if len(tools) == 0 {
		sseWrite(w, map[string]any{"type": "error", "message": "devtools MCP server not connected. Please connect it first."})
		sseWrite(w, map[string]any{"type": "done"})
		return
	}

	// Strip namespacing for Claude API (use short names).
	type apiTool struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		InputSchema any    `json:"input_schema"`
	}
	apiTools := make([]apiTool, len(tools))
	shortToFull := make(map[string]string, len(tools))
	for i, t := range tools {
		short := strings.TrimPrefix(t.Name, "devtools__")
		apiTools[i] = apiTool{Name: short, Description: t.Description, InputSchema: t.InputSchema}
		shortToFull[short] = t.Name
	}

	// Build messages from history + current.
	type msgBlock struct {
		Type      string          `json:"type"`
		Text      string          `json:"text,omitempty"`
		ID        string          `json:"id,omitempty"`
		Name      string          `json:"name,omitempty"`
		Input     json.RawMessage `json:"input,omitempty"`
		ToolUseID string          `json:"tool_use_id,omitempty"`
		Content   string          `json:"content,omitempty"`
		IsError   bool            `json:"is_error,omitempty"`
	}
	type apiMsg struct {
		Role    string `json:"role"`
		Content any    `json:"content"` // string or []msgBlock
	}

	history := req.History
	if len(history) > 10 {
		history = history[len(history)-10:]
	}

	messages := make([]apiMsg, 0, len(history)+1)
	for _, h := range history {
		messages = append(messages, apiMsg{Role: h.Role, Content: h.Content})
	}
	messages = append(messages, apiMsg{Role: "user", Content: req.Message})

	// Tool-use loop.
	ctx := r.Context()
	for turn := 0; turn < maxFilesTurns; turn++ {
		respBody, err := filesCallClaude(ctx, apiKey, filesSystemPrompt, messages, apiTools)
		if err != nil {
			sseWrite(w, map[string]any{"type": "error", "message": err.Error()})
			break
		}

		// Parse response.
		var result struct {
			Content  []contentBlock `json:"content"`
			StopReason string       `json:"stop_reason"`
			Usage    struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(respBody, &result); err != nil {
			sseWrite(w, map[string]any{"type": "error", "message": "failed to parse response"})
			break
		}

		// Emit usage.
		sseWrite(w, map[string]any{
			"type": "usage",
			"usage": map[string]any{
				"input":  result.Usage.InputTokens,
				"output": result.Usage.OutputTokens,
			},
		})

		// Process content blocks.
		var assistantBlocks []msgBlock
		hasToolUse := false

		for _, block := range result.Content {
			switch block.Type {
			case "text":
				if block.Text != "" {
					sseWrite(w, map[string]any{"type": "text_delta", "text": block.Text})
				}
				assistantBlocks = append(assistantBlocks, msgBlock{Type: "text", Text: block.Text})

			case "tool_use":
				hasToolUse = true
				// Emit tool_call event for frontend.
				var inputMap map[string]any
				json.Unmarshal(block.Input, &inputMap)
				sseWrite(w, map[string]any{
					"type":  "tool_call",
					"tool":  block.Name,
					"id":    block.ID,
					"input": inputMap,
				})
				assistantBlocks = append(assistantBlocks, msgBlock{
					Type:  "tool_use",
					ID:    block.ID,
					Name:  block.Name,
					Input: block.Input,
				})
			}
		}

		// Append assistant message.
		messages = append(messages, apiMsg{Role: "assistant", Content: assistantBlocks})

		if !hasToolUse {
			break
		}

		// Execute tools and build tool results.
		var toolResults []msgBlock
		for _, block := range result.Content {
			if block.Type != "tool_use" {
				continue
			}

			var args map[string]any
			json.Unmarshal(block.Input, &args)

			toolResult, toolErr := mcpMgr.CallTool(context.Background(), "devtools", block.Name, args)

			resultText := ""
			isError := false
			if toolErr != nil {
				resultText = "Error: " + toolErr.Error()
				isError = true
			} else if toolResult != nil {
				for _, c := range toolResult.Content {
					if tc, ok := c.(mcp.TextContent); ok {
						resultText = tc.Text
						break
					}
				}
			}

			// Truncate long results.
			if len(resultText) > 8000 {
				resultText = resultText[:8000] + "\n... (truncated)"
			}

			sseWrite(w, map[string]any{
				"type":     "tool_result",
				"tool":     block.Name,
				"output":   resultText,
				"is_error": isError,
			})

			toolResults = append(toolResults, msgBlock{
				Type:      "tool_result",
				ToolUseID: block.ID,
				Content:   resultText,
				IsError:   isError,
			})
		}

		messages = append(messages, apiMsg{Role: "user", Content: toolResults})
	}

	sseWrite(w, map[string]any{"type": "done"})
}

func filesCallClaude(ctx context.Context, apiKey, system string, messages any, tools any) ([]byte, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"model":       ModelSonnet,
		"max_tokens":  4096,
		"temperature": 0.2,
		"system":      system,
		"messages":    messages,
		"tools":       tools,
	})

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Claude API error %d: %s", resp.StatusCode, truncateStr(string(body), 500))
	}

	return body, nil
}

// truncateStr is like truncate from agent.go but avoids name conflict.
func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
