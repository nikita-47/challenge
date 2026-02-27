package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// UnmarshalJSON handles both string and []contentBlock content for backward compatibility.
func (m *message) UnmarshalJSON(data []byte) error {
	var raw struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m.Role = raw.Role

	// Try string first (old sessions).
	var s string
	if err := json.Unmarshal(raw.Content, &s); err == nil {
		m.Content = s
		return nil
	}

	// Otherwise treat as []contentBlock.
	var blocks []contentBlock
	if err := json.Unmarshal(raw.Content, &blocks); err == nil {
		m.Content = blocks
		return nil
	}

	// Fallback: store as-is.
	m.Content = string(raw.Content)
	return nil
}

// messageText extracts plain text from message.Content (string or []contentBlock).
func messageText(m message) string {
	switch v := m.Content.(type) {
	case string:
		return v
	case []contentBlock:
		var b strings.Builder
		for _, block := range v {
			if block.Type == "text" {
				b.WriteString(block.Text)
			}
		}
		return b.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

type modelInfo struct {
	name     string
	provider string
	baseURL  string
	apiKey   string
	model    string
	costIn   float64 // cost per 1M input tokens
	costOut  float64 // cost per 1M output tokens
}

func buildSystemPrompt(cfg config) string {
	parts := []string{}
	if cfg.system != "" {
		parts = append(parts, cfg.system)
	}
	if cfg.format != "" {
		parts = append(parts, "Always respond in this format: "+cfg.format)
	}
	if cfg.stop != "" {
		parts = append(parts, "Always end your response with: "+cfg.stop)
	}
	return strings.Join(parts, "\n")
}

func buildRequest(cfg config, msgs []message) map[string]any {
	req := map[string]any{
		"model":      "claude-sonnet-4-5-20250929",
		"max_tokens": cfg.maxTokens,
		"messages":   msgs,
		"stream":     true,
	}

	if cfg.temperature >= 0 {
		req["temperature"] = cfg.temperature
	}
	if sp := buildSystemPrompt(cfg); sp != "" {
		req["system"] = sp
	}
	if cfg.stop != "" {
		req["stop_sequences"] = []string{cfg.stop}
	}

	return req
}

func buildOpenAIRequest(model string, cfg config, msgs []message) map[string]any {
	openaiMsgs := make([]map[string]string, 0, len(msgs)+1)
	if sp := buildSystemPrompt(cfg); sp != "" {
		openaiMsgs = append(openaiMsgs, map[string]string{"role": "system", "content": sp})
	}
	for _, m := range msgs {
		openaiMsgs = append(openaiMsgs, map[string]string{"role": m.Role, "content": messageText(m)})
	}

	req := map[string]any{
		"model":          model,
		"max_tokens":     cfg.maxTokens,
		"messages":       openaiMsgs,
		"stream":         true,
		"stream_options": map[string]any{"include_usage": true},
	}

	if cfg.temperature >= 0 {
		req["temperature"] = cfg.temperature
	}
	if cfg.stop != "" {
		req["stop"] = []string{cfg.stop}
	}

	return req
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:8] + "****"
}

func formatCurl(apiKey string, body []byte) string {
	var pretty bytes.Buffer
	json.Indent(&pretty, body, "  ", "  ")
	var b strings.Builder
	fmt.Fprintf(&b, "curl -X POST .../v1/messages \\\n")
	fmt.Fprintf(&b, "  -H \"x-api-key: %s\" \\\n", maskKey(apiKey))
	fmt.Fprintf(&b, "  -d '%s'\n", pretty.String())
	return b.String()
}

func printCurl(apiKey string, body []byte) {
	fmt.Fprintf(os.Stderr, "\n\033[2m── curl ────────────────────────────────────────────────────\033[0m\n")
	fmt.Fprintf(os.Stderr, "\033[2m%s\033[0m", formatCurl(apiKey, body))
	fmt.Fprintf(os.Stderr, "\033[2m────────────────────────────────────────────────────────────\033[0m\n\n")
}

func streamChat(apiKey string, cfg config, msgs []message, onToken func(string)) (string, tokenUsage, error) {
	body, _ := json.Marshal(buildRequest(cfg, msgs))

	if cfg.verbose {
		printCurl(apiKey, body)
	}

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

	return readStream(resp.Body, onToken)
}

func streamChatOpenAI(baseURL, model string, cfg config, msgs []message, onToken func(string)) (string, tokenUsage, error) {
	if model == "" {
		model = "default"
	}
	body, _ := json.Marshal(buildOpenAIRequest(model, cfg, msgs))

	req, _ := http.NewRequest("POST", baseURL+"/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", tokenUsage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(resp.Body)
		return "", tokenUsage{}, fmt.Errorf("API error (%d): %s", resp.StatusCode, errBody)
	}

	return readStreamOpenAI(resp.Body, onToken)
}

func readStreamOpenAI(r io.Reader, onToken func(string)) (string, tokenUsage, error) {
	var full strings.Builder
	var usage tokenUsage
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if len(event.Choices) > 0 && event.Choices[0].Delta.Content != "" {
			text := event.Choices[0].Delta.Content
			full.WriteString(text)
			if onToken != nil {
				onToken(text)
			}
		}
		if event.Usage != nil {
			usage.InputTokens = event.Usage.PromptTokens
			usage.OutputTokens = event.Usage.CompletionTokens
		}
	}

	// Fallback: estimate tokens if not reported.
	if usage.OutputTokens == 0 && full.Len() > 0 {
		usage.OutputTokens = full.Len() / 4
	}

	if err := scanner.Err(); err != nil {
		return full.String(), usage, err
	}
	return full.String(), usage, nil
}

func readStream(r io.Reader, onToken func(string)) (string, tokenUsage, error) {
	var full strings.Builder
	var usage tokenUsage
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var raw json.RawMessage
		if err := json.Unmarshal([]byte(data), &raw); err != nil {
			continue
		}

		var event struct {
			Type string `json:"type"`
		}
		json.Unmarshal(raw, &event)

		switch event.Type {
		case "message_start":
			var ms struct {
				Message struct {
					Usage struct {
						InputTokens int `json:"input_tokens"`
					} `json:"usage"`
				} `json:"message"`
			}
			json.Unmarshal(raw, &ms)
			usage.InputTokens = ms.Message.Usage.InputTokens

		case "content_block_delta":
			var cbd struct {
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			}
			json.Unmarshal(raw, &cbd)
			if cbd.Delta.Type == "text_delta" {
				text := cbd.Delta.Text
				full.WriteString(text)
				if onToken != nil {
					onToken(text)
				}
			}

		case "message_delta":
			var md struct {
				Usage struct {
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			}
			json.Unmarshal(raw, &md)
			usage.OutputTokens = md.Usage.OutputTokens
		}
	}

	if err := scanner.Err(); err != nil {
		return full.String(), usage, err
	}
	return full.String(), usage, nil
}
