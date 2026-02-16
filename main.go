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

func main() {
	apiKey := loadEnv(".env", "ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "ANTHROPIC_API_KEY not set in .env")
		os.Exit(1)
	}

	fmt.Println("Chat with Claude (type \"exit\" to quit)")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	var history []message

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "exit" {
			break
		}

		history = append(history, message{Role: "user", Content: input})

		reply, err := chat(apiKey, history)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			history = history[:len(history)-1]
			continue
		}

		history = append(history, message{Role: "assistant", Content: reply})
		fmt.Printf("\nClaude: %s\n\n", reply)
	}
}

func chat(apiKey string, messages []message) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model":      "claude-sonnet-4-5-20250929",
		"max_tokens": 1024,
		"messages":   messages,
	})

	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, respBody)
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	json.Unmarshal(respBody, &result)

	var parts []string
	for _, block := range result.Content {
		parts = append(parts, block.Text)
	}
	return strings.Join(parts, ""), nil
}

func loadEnv(path, key string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if k, v, ok := strings.Cut(line, "="); ok && strings.TrimSpace(k) == key {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
