package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type config struct {
	maxTokens int
	system    string
	stop      string
	format    string
	compare   string
}

// ─── Markdown rendering ───────────────────────────────────────────────────────

var (
	reCodeBlock  = regexp.MustCompile("(?s)```[a-z]*\n?(.*?)```")
	reCodeInline = regexp.MustCompile("`([^`\n]+)`")
	reBold       = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)
	reHeading    = regexp.MustCompile(`(?m)^#{1,3} (.+)$`)
	reHRule      = regexp.MustCompile(`(?m)^[-*_]{3,}\s*$`)
	reBullet     = regexp.MustCompile(`(?m)^(\s*)[*-] `)
)

func renderMarkdown(s string) string {
	s = reCodeBlock.ReplaceAllString(s, "\033[33m$1\033[0m")
	s = reBold.ReplaceAllString(s, "\033[1m$1\033[0m")
	s = reCodeInline.ReplaceAllString(s, "\033[33m$1\033[0m")
	s = reHeading.ReplaceAllString(s, "\033[1m$1\033[0m")
	s = reHRule.ReplaceAllString(s, strings.Repeat("─", 60))
	s = reBullet.ReplaceAllString(s, "$1• ")
	return s
}

// ─── App ──────────────────────────────────────────────────────────────────────

func main() {
	cfg := parseArgs()

	apiKey := loadEnv(".env", "ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "ANTHROPIC_API_KEY not set in .env")
		os.Exit(1)
	}

	if cfg.compare != "" {
		scanner := bufio.NewScanner(os.Stdin)
		runComparison(apiKey, cfg, cfg.compare, scanner)
		return
	}

	printBanner(cfg)
	runChat(apiKey, cfg)
}

func parseArgs() config {
	cfg := config{}
	flag.IntVar(&cfg.maxTokens, "max-tokens", 1024, "max response tokens")
	flag.StringVar(&cfg.system, "system", "", "system prompt")
	flag.StringVar(&cfg.stop, "stop", "", "stop sequence")
	flag.StringVar(&cfg.format, "format", "", "response format instruction")
	flag.StringVar(&cfg.compare, "compare", "", "run 4-way comparison and exit")
	flag.Parse()
	return cfg
}

func printBanner(cfg config) {
	fmt.Println("=== Claude CLI Chat ===")
	fmt.Printf("Model:      claude-sonnet-4-5-20250929\n")
	fmt.Printf("Max tokens: %d\n", cfg.maxTokens)
	if cfg.system != "" {
		fmt.Printf("System:     %s\n", cfg.system)
	}
	if cfg.stop != "" {
		fmt.Printf("Stop:       %q\n", cfg.stop)
	}
	if cfg.format != "" {
		fmt.Printf("Format:     %s\n", cfg.format)
	}
	fmt.Println()
	fmt.Println("Type /help for commands, \"exit\" or \"quit\" to quit.")
	fmt.Println()
}

func printHelp() {
	fmt.Println("Commands:")
	fmt.Println("  /help                — show this help")
	fmt.Println("  /clear               — reset conversation history")
	fmt.Println("  /system <text>       — update system prompt")
	fmt.Println("  /compare <question>  — stream 4 reasoning approaches side-by-side")
	fmt.Println("  exit / quit          — quit")
	fmt.Println()
	fmt.Println("Flags (set at startup):")
	fmt.Println("  --max-tokens int    max response tokens (default 1024)")
	fmt.Println("  --system string     system prompt")
	fmt.Println("  --stop string       stop sequence")
	fmt.Println("  --format string     response format instruction")
	fmt.Println("  --compare string    run 4-way comparison directly and exit")
	fmt.Println()
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

func runChat(apiKey string, cfg config) {
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

		switch {
		case input == "exit" || input == "quit":
			fmt.Println("Goodbye!")
			return
		case input == "/help":
			printHelp()
			continue
		case input == "/clear":
			history = nil
			fmt.Println("History cleared.")
			fmt.Println()
			continue
		case strings.HasPrefix(input, "/system "):
			cfg.system = strings.TrimPrefix(input, "/system ")
			fmt.Printf("System prompt updated: %s\n\n", cfg.system)
			continue
		case strings.HasPrefix(input, "/compare "):
			question := strings.TrimPrefix(input, "/compare ")
			runComparison(apiKey, cfg, question, scanner)
			printBanner(cfg)
			continue
		}

		history = append(history, message{Role: "user", Content: input})

		fmt.Print("\nClaude: ")
		reply, err := streamChat(apiKey, cfg, history)
		if err != nil {
			fmt.Fprintln(os.Stderr, "\nError:", err)
			history = history[:len(history)-1]
			continue
		}
		fmt.Println("\n")

		history = append(history, message{Role: "assistant", Content: reply})
	}
}

// ─── API ──────────────────────────────────────────────────────────────────────

func buildRequest(cfg config, msgs []message) map[string]any {
	req := map[string]any{
		"model":      "claude-sonnet-4-5-20250929",
		"max_tokens": cfg.maxTokens,
		"messages":   msgs,
		"stream":     true,
	}

	if sp := buildSystemPrompt(cfg); sp != "" {
		req["system"] = sp
	}
	if cfg.stop != "" {
		req["stop_sequences"] = []string{cfg.stop}
	}

	return req
}

func streamChat(apiKey string, cfg config, msgs []message) (string, error) {
	body, _ := json.Marshal(buildRequest(cfg, msgs))

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

// readStream prints tokens as they arrive, rendering markdown line-by-line.
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

			// Render complete lines as they arrive.
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

// ─── Env ──────────────────────────────────────────────────────────────────────

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
