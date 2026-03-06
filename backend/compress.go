package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// compressThreshold returns the number of messages before compression kicks in.
// Uses the shared windowSize setting so all strategies share the same N.
func compressThreshold(cw *contextWindow) int {
	return getWindowSize(cw)
}

// contextWindow tracks current messages and a single rolling summary.
type contextWindow struct {
	Summary      string            // accumulated summary of all previous messages (empty = none)
	Messages     []message         // current unsummarized messages (≤ windowSize)
	Settings     *sessionSettings  // chat settings (model, temperature, maxTokens, system)
	Stats        *sessionStats     // cumulative token stats for the session
	Facts        map[string]string // key-value memory for facts strategy
	Branches     []branch          // dialog branches for branch strategy
	ActiveBranch string            // current active branch name ("" or "main" = main)
	TaskState    *TaskState        // task state machine (nil = no active task)
}

// compressInfo holds details about a compression event.
type compressInfo struct {
	MessageCount int
	SummaryLen   int
	TokensSaved  int
}

// maybeCompress checks if compression is needed and performs it, mutating cw.
// Call BEFORE appending the new user message so the current question isn't swallowed.
// Returns non-nil compressInfo if compression was performed.
func maybeCompress(apiKey string, cw *contextWindow, stats *tokenStats) (*compressInfo, error) {
	if len(cw.Messages) < compressThreshold(cw) {
		return nil, nil
	}

	summary, usage, err := summarize(apiKey, cw.Summary, cw.Messages)
	if err != nil {
		return nil, fmt.Errorf("summarize: %w", err)
	}

	// Estimate savings: old text vs new summary.
	var oldLen int
	for _, m := range cw.Messages {
		oldLen += len(messageText(m))
	}
	if cw.Summary != "" {
		oldLen += len(cw.Summary)
	}
	saved := (oldLen - len(summary)) / 4
	if saved > 0 {
		stats.TokensSaved += saved
	}

	msgCount := len(cw.Messages)
	stats.Add(usage)

	cw.Summary = summary
	cw.Messages = nil // reset — start accumulating again
	return &compressInfo{
		MessageCount: msgCount,
		SummaryLen:   len(summary),
		TokensSaved:  saved,
	}, nil
}

// buildCompressedMessages returns the message slice to send to the API,
// prepending summary+ack if a summary exists.
// filterAPIMessages returns only user/assistant messages suitable for the Claude API.
func filterAPIMessages(msgs []message) []message {
	result := make([]message, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == "system" {
			continue
		}
		m.ApiRequest = "" // strip debug data before sending to LLM
		result = append(result, m)
	}
	return result
}

func buildCompressedMessages(cw *contextWindow) []message {
	apiMsgs := filterAPIMessages(cw.Messages)

	if cw.Summary != "" {
		summaryMsg := message{
			Role:    "user",
			Content: "Previous conversation summary:\n" + cw.Summary,
		}
		ackMsg := message{
			Role:    "assistant",
			Content: "Understood, I have the context from our previous conversation.",
		}
		result := make([]message, 0, 2+len(apiMsgs))
		result = append(result, summaryMsg, ackMsg)
		result = append(result, apiMsgs...)
		return result
	}

	return apiMsgs
}

// summarize calls Claude API (non-streaming) to produce a concise summary
// from the previous summary (if any) and the current messages.
func summarize(apiKey string, prevSummary string, msgs []message) (string, tokenUsage, error) {
	var conv strings.Builder

	if prevSummary != "" {
		conv.WriteString("Previous summary:\n")
		conv.WriteString(prevSummary)
		conv.WriteString("\n\nNew conversation:\n")
	}

	for _, m := range msgs {
		conv.WriteString(m.Role)
		conv.WriteString(": ")
		conv.WriteString(messageText(m))
		conv.WriteString("\n\n")
	}

	body, _ := json.Marshal(map[string]any{
		"model":      DefaultModel,
		"max_tokens": 512,
		"system":     "Summarize the following conversation concisely, preserving key facts, decisions, and context needed for continuation. If a previous summary is provided, merge it with the new conversation into one unified summary. Be brief.",
		"messages": []map[string]string{
			{"role": "user", "content": conv.String()},
		},
	})

	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", tokenUsage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(resp.Body)
		return "", tokenUsage{}, fmt.Errorf("API error (%d): %s", resp.StatusCode, errBody)
	}

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", tokenUsage{}, fmt.Errorf("decode error: %w", err)
	}

	var text strings.Builder
	for _, block := range result.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}

	usage := tokenUsage{
		InputTokens:  result.Usage.InputTokens,
		OutputTokens: result.Usage.OutputTokens,
	}

	return text.String(), usage, nil
}
