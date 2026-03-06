package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

// buildFactsMessages returns facts preamble + last N messages.
func buildFactsMessages(cw *contextWindow) []message {
	n := getWindowSize(cw)
	msgs := filterAPIMessages(activeMessages(cw))
	if len(msgs) > n {
		msgs = msgs[len(msgs)-n:]
	}

	if len(cw.Facts) == 0 {
		return msgs
	}

	// Build facts preamble with sorted keys for deterministic order.
	keys := make([]string, 0, len(cw.Facts))
	for k := range cw.Facts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var factsText strings.Builder
	factsText.WriteString("Key facts from our conversation:\n")
	for _, k := range keys {
		factsText.WriteString(fmt.Sprintf("- %s: %s\n", k, cw.Facts[k]))
	}

	factMsg := message{Role: "user", Content: factsText.String()}
	ackMsg := message{Role: "assistant", Content: "Understood, I have these facts as context."}

	result := make([]message, 0, 2+len(msgs))
	result = append(result, factMsg, ackMsg)
	result = append(result, msgs...)
	return result
}

// maybeExtractFacts extracts key facts from the last user+assistant exchange.
// Called AFTER the assistant response is appended.
func maybeExtractFacts(apiKey string, cw *contextWindow, stats *tokenStats) error {
	msgs := activeMessages(cw)
	if len(msgs) < 2 {
		return nil
	}

	// Take last 2 messages (should be user + assistant).
	recent := msgs[len(msgs)-2:]
	if recent[0].Role != "user" || recent[1].Role != "assistant" {
		return nil
	}

	newFacts, usage, err := extractFacts(apiKey, cw.Facts, recent)
	if err != nil {
		return err
	}
	stats.Add(usage)

	if cw.Facts == nil {
		cw.Facts = make(map[string]string)
	}
	for k, v := range newFacts {
		if v == "" {
			delete(cw.Facts, k)
		} else {
			cw.Facts[k] = v
		}
	}
	return nil
}

// extractFacts calls Claude API (non-streaming) to extract key-value facts.
func extractFacts(apiKey string, existing map[string]string, msgs []message) (map[string]string, tokenUsage, error) {
	var prompt strings.Builder
	if len(existing) > 0 {
		prompt.WriteString("Current facts:\n")
		b, _ := json.Marshal(existing)
		prompt.Write(b)
		prompt.WriteString("\n\n")
	}
	prompt.WriteString("New conversation:\n")
	for _, m := range msgs {
		prompt.WriteString(m.Role)
		prompt.WriteString(": ")
		prompt.WriteString(messageText(m))
		prompt.WriteString("\n\n")
	}

	body, _ := json.Marshal(map[string]any{
		"model":      DefaultModel,
		"max_tokens": 512,
		"system": `Extract key facts from this conversation exchange. Output ONLY a JSON object with updated/new key-value pairs.
Rules:
- Keys should be snake_case identifiers (e.g., "user_name", "project_type", "preferred_language")
- Values should be short strings (1-2 sentences max)
- Include only important facts: goals, decisions, preferences, constraints, names, technical choices
- Set value to "" to delete an outdated fact
- If no new facts found, output {}
- Do NOT wrap in markdown code blocks, output raw JSON only`,
		"messages": []map[string]string{
			{"role": "user", "content": prompt.String()},
		},
	})

	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, tokenUsage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, tokenUsage{}, fmt.Errorf("API error (%d): %s", resp.StatusCode, errBody)
	}

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, tokenUsage{}, fmt.Errorf("decode error: %w", err)
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

	// Parse JSON output.
	facts := make(map[string]string)
	raw := strings.TrimSpace(text.String())
	// Strip markdown code block wrapping if present.
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) >= 3 {
			raw = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}
	if err := json.Unmarshal([]byte(raw), &facts); err != nil {
		// Non-fatal: return empty facts if parsing fails.
		return map[string]string{}, usage, nil
	}

	return facts, usage, nil
}
