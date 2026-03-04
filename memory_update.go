package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// maybeUpdateMemory analyzes the last user+assistant exchange and updates
// profile/project memory if the LLM finds new relevant information.
// Operator memory is never auto-updated (immutable).
func maybeUpdateMemory(apiKey string, cw *contextWindow, stats *tokenStats) error {
	if cw.Settings == nil {
		fmt.Fprintln(os.Stderr, "[memory_update] skip: settings is nil")
		return nil
	}

	msgs := activeMessages(cw)
	fmt.Fprintf(os.Stderr, "[memory_update] activeMessages count: %d, profile=%q, project=%q\n",
		len(msgs), cw.Settings.Profile, cw.Settings.Project)

	if len(msgs) < 2 {
		fmt.Fprintln(os.Stderr, "[memory_update] skip: less than 2 messages")
		return nil
	}

	// Find last user+assistant pair (skip system messages at the end).
	var userIdx, assistantIdx int = -1, -1
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "assistant" && assistantIdx == -1 {
			assistantIdx = i
		} else if msgs[i].Role == "user" && assistantIdx != -1 && userIdx == -1 {
			userIdx = i
			break
		}
	}

	if userIdx == -1 || assistantIdx == -1 || userIdx+1 != assistantIdx {
		fmt.Fprintf(os.Stderr, "[memory_update] skip: no valid user+assistant pair found (userIdx=%d, assistantIdx=%d)\n", userIdx, assistantIdx)
		// Fall back to original last-2 approach.
		recent := msgs[len(msgs)-2:]
		if recent[0].Role != "user" || recent[1].Role != "assistant" {
			fmt.Fprintf(os.Stderr, "[memory_update] skip: last 2 msgs are not user+assistant (roles: %s, %s)\n",
				recent[0].Role, recent[1].Role)
			return nil
		}
		userIdx = len(msgs) - 2
		assistantIdx = len(msgs) - 1
	}

	recent := msgs[userIdx : assistantIdx+1]

	assistantText := messageText(recent[1])
	fmt.Fprintf(os.Stderr, "[memory_update] assistant text len=%d, preview=%q\n",
		len(assistantText), truncate(assistantText, 80))

	// Skip if assistant response is empty (e.g. agent error).
	if assistantText == "" {
		fmt.Fprintln(os.Stderr, "[memory_update] skip: assistant text is empty")
		return nil
	}

	if cw.Settings.Profile != "" {
		fmt.Fprintf(os.Stderr, "[memory_update] analyzing profile %q\n", cw.Settings.Profile)
		current, err := getProfile(cw.Settings.Profile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[memory_update] getProfile error: %v\n", err)
		}
		updated, usage, err := analyzeMemoryUpdate(apiKey, "profile", cw.Settings.Profile, current, recent)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[memory_update] analyzeMemoryUpdate(profile) error: %v\n", err)
			return err
		}
		stats.Add(usage)
		fmt.Fprintf(os.Stderr, "[memory_update] profile result=%q (NO_UPDATE=%v)\n", truncate(updated, 60), updated == "NO_UPDATE")
		if updated != "" && updated != "NO_UPDATE" {
			if saveErr := saveProfile(cw.Settings.Profile, updated); saveErr != nil {
				fmt.Fprintf(os.Stderr, "[memory_update] saveProfile error: %v\n", saveErr)
				return saveErr
			}
			fmt.Fprintf(os.Stderr, "[memory_update] profile %q saved successfully\n", cw.Settings.Profile)
		}
	}

	if cw.Settings.Project != "" {
		fmt.Fprintf(os.Stderr, "[memory_update] analyzing project %q\n", cw.Settings.Project)
		current, err := getProject(cw.Settings.Project)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[memory_update] getProject error: %v\n", err)
		}
		updated, usage, err := analyzeMemoryUpdate(apiKey, "project", cw.Settings.Project, current, recent)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[memory_update] analyzeMemoryUpdate(project) error: %v\n", err)
			return err
		}
		stats.Add(usage)
		fmt.Fprintf(os.Stderr, "[memory_update] project result=%q (NO_UPDATE=%v)\n", truncate(updated, 60), updated == "NO_UPDATE")
		if updated != "" && updated != "NO_UPDATE" {
			if saveErr := saveProject(cw.Settings.Project, updated); saveErr != nil {
				fmt.Fprintf(os.Stderr, "[memory_update] saveProject error: %v\n", saveErr)
				return saveErr
			}
			fmt.Fprintf(os.Stderr, "[memory_update] project %q saved successfully\n", cw.Settings.Project)
		}
	}

	fmt.Fprintln(os.Stderr, "[memory_update] done")
	return nil
}

// analyzeMemoryUpdate calls Claude (non-streaming) to decide whether the
// memory document needs updating based on the latest conversation exchange.
func analyzeMemoryUpdate(apiKey, kind, name, currentContent string, conversation []message) (string, tokenUsage, error) {
	fmt.Fprintf(os.Stderr, "[memory_update] analyzeMemoryUpdate: kind=%s, name=%s, contentLen=%d, msgs=%d\n",
		kind, name, len(currentContent), len(conversation))
	systemPrompt := fmt.Sprintf(`You are a memory manager for a chat assistant. You maintain a "%s" memory document named "%s".

A %s document stores important, persistent information that should be remembered across conversations.
- "profile" documents contain user preferences, background, goals, and personal context.
- "project" documents contain project-specific context, decisions, technical choices, and working notes.

You will be given the current content of the document and the latest conversation exchange.
Your task: decide if the conversation contains new information worth adding or updating in the document.

Rules:
- If there is nothing new or relevant, respond with exactly: NO_UPDATE
- If there is new information, respond with the FULL updated markdown document (not just the diff).
- Keep the document concise and well-structured.
- Do NOT wrap the output in markdown code blocks.
- Do NOT explain your reasoning — output only the document content or NO_UPDATE.`, kind, name, kind)

	var prompt strings.Builder
	prompt.WriteString("## Current document content\n\n")
	if currentContent == "" {
		prompt.WriteString("(empty)\n\n")
	} else {
		prompt.WriteString(currentContent)
		prompt.WriteString("\n\n")
	}
	prompt.WriteString("## Latest conversation exchange\n\n")
	for _, m := range conversation {
		prompt.WriteString(m.Role)
		prompt.WriteString(": ")
		prompt.WriteString(messageText(m))
		prompt.WriteString("\n\n")
	}
	prompt.WriteString("Should the document be updated? If yes, provide the full updated document. If no, respond NO_UPDATE.")

	body, _ := json.Marshal(map[string]any{
		"model":      "claude-3-5-haiku-20241022",
		"max_tokens": 1024,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": prompt.String()},
		},
	})

	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	fmt.Fprintf(os.Stderr, "[memory_update] calling Haiku API for %s/%s...\n", kind, name)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[memory_update] HTTP error: %v\n", err)
		return "", tokenUsage{}, err
	}
	defer resp.Body.Close()

	fmt.Fprintf(os.Stderr, "[memory_update] Haiku API response status: %d\n", resp.StatusCode)
	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "[memory_update] Haiku API error body: %s\n", errBody)
		return "", tokenUsage{}, fmt.Errorf("API error (%d): %s", resp.StatusCode, errBody)
	}

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "[memory_update] decode error: %v\n", err)
		return "", tokenUsage{}, fmt.Errorf("decode error: %w", err)
	}
	fmt.Fprintf(os.Stderr, "[memory_update] Haiku response: stopReason=%s, contentBlocks=%d\n",
		result.StopReason, len(result.Content))

	usage := tokenUsage{
		InputTokens:  result.Usage.InputTokens,
		OutputTokens: result.Usage.OutputTokens,
	}

	var text strings.Builder
	for _, block := range result.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}

	return strings.TrimSpace(text.String()), usage, nil
}
