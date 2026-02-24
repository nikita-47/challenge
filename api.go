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
	Content string `json:"content"`
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
		openaiMsgs = append(openaiMsgs, map[string]string{"role": m.Role, "content": m.Content})
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

func streamChat(apiKey string, cfg config, msgs []message) (string, error) {
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
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, errBody)
	}

	return readStream(resp.Body)
}

func readStream(r io.Reader) (string, error) {
	var full, pending strings.Builder
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
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
			text := event.Delta.Text
			full.WriteString(text)
			pending.WriteString(text)

			buf := pending.String()
			if i := strings.LastIndex(buf, "\n"); i >= 0 {
				fmt.Print(renderMarkdown(buf[:i+1]))
				pending.Reset()
				pending.WriteString(buf[i+1:])
			}
		}
	}

	if pending.Len() > 0 {
		fmt.Print(renderMarkdown(pending.String()))
	}

	if err := scanner.Err(); err != nil {
		return full.String(), err
	}
	return full.String(), nil
}
