package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func runChat(apiKey, openaiKey string, cfg config) {
	scanner := bufio.NewScanner(os.Stdin)
	sessionName := cfg.session
	var stats tokenStats

	cw := &contextWindow{}
	loaded, err := loadSessionCW(sessionName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load session: %v\n", err)
	}
	if loaded != nil {
		cw = loaded
	}
	if len(cw.Messages) > 0 {
		name := sessionName
		if name == "" {
			name = "default"
		}
		fmt.Printf("Resumed session '%s' (%d messages", name, len(cw.Messages))
		if cw.Summary != "" {
			fmt.Printf(", has summary")
		}
		fmt.Println(")")
		fmt.Println()
	}

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
			cw = &contextWindow{}
			stats = tokenStats{}
			if err := deleteSession(sessionName); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to delete session file: %v\n", err)
			}
			fmt.Println("History cleared.")
			fmt.Println()
			continue
		case input == "/tokens":
			if stats.Exchanges == 0 {
				fmt.Println("No token usage yet.")
			} else {
				fmt.Println(stats.FormatTotal())
			}
			fmt.Println()
			continue
		case input == "/compress":
			if cw.Summary == "" {
				fmt.Printf("No summary yet (compression triggers at %d messages).\n", compressThreshold)
			} else {
				fmt.Printf("Current messages: %d | Summary:\n\n%s\n", len(cw.Messages), cw.Summary)
			}
			fmt.Println()
			continue
		case strings.HasPrefix(input, "/system "):
			cfg.system = strings.TrimPrefix(input, "/system ")
			fmt.Printf("System prompt updated: %s\n\n", cfg.system)
			continue
		case strings.HasPrefix(input, "/compare "):
			question := strings.TrimPrefix(input, "/compare ")
			runComparison(apiKey, cfg, question, scanner)
			printBanner(cfg, openaiKey)
			continue
		case strings.HasPrefix(input, "/temp "):
			question := strings.TrimPrefix(input, "/temp ")
			runTempComparison(apiKey, cfg, question, scanner)
			printBanner(cfg, openaiKey)
			continue
		case strings.HasPrefix(input, "/models "):
			question := strings.TrimPrefix(input, "/models ")
			runModelComparison(apiKey, openaiKey, cfg, question, scanner)
			printBanner(cfg, openaiKey)
			continue
		case strings.HasPrefix(input, "/agent "):
			task := strings.TrimPrefix(input, "/agent ")
			agent := newAgent(apiKey)
			result, err := agent.Run(task, cw.Messages)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Agent error: %v\n\n", err)
			} else {
				fmt.Print("Claude: ")
				fmt.Println(renderMarkdown(result))
				fmt.Println()

				// Add flattened messages to history for persistence.
				cw.Messages = append(cw.Messages, message{Role: "user", Content: task})
				cw.Messages = append(cw.Messages, message{Role: "assistant", Content: result})

				if err := saveSessionCW(sessionName, cw); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to auto-save session: %v\n", err)
				}
			}
			// Merge agent stats into session stats.
			stats.TotalInput += agent.Stats.TotalInput
			stats.TotalOutput += agent.Stats.TotalOutput
			stats.Exchanges += agent.Stats.Exchanges
			continue
		case input == "/save" || strings.HasPrefix(input, "/save "):
			saveName := strings.TrimPrefix(input, "/save")
			saveName = strings.TrimSpace(saveName)
			if saveName == "" {
				saveName = sessionName
			}
			if err := saveSessionCW(saveName, cw); err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to save session: %v\n", err)
			} else {
				name := saveName
				if name == "" {
					name = "default"
				}
				fmt.Printf("Session saved as '%s'.\n", name)
			}
			fmt.Println()
			continue
		case strings.HasPrefix(input, "/load "):
			loadName := strings.TrimSpace(strings.TrimPrefix(input, "/load "))
			loaded, err := loadSessionCW(loadName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to load session: %v\n", err)
			} else if loaded == nil {
				fmt.Printf("Session '%s' not found.\n", loadName)
			} else {
				cw = loaded
				sessionName = loadName
				stats = tokenStats{} // reset stats for loaded session
				fmt.Printf("Loaded session '%s' (%d messages", loadName, len(cw.Messages))
				if cw.Summary != "" {
					fmt.Printf(", has summary")
				}
				fmt.Println(").")
			}
			fmt.Println()
			continue
		}

		// Compress history BEFORE adding the new message so the current
		// question doesn't get swallowed into the summary.
		if err := maybeCompress(apiKey, cw, &stats); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: compression failed, continuing with full history: %v\n", err)
		}

		cw.Messages = append(cw.Messages, message{Role: "user", Content: input})
		compressed := buildCompressedMessages(cw)

		fmt.Print("\nClaude: ")
		var reply string
		var usage tokenUsage
		if cfg.baseURL != "" {
			reply, usage, err = streamChatOpenAI(cfg.baseURL, cfg.model, cfg, compressed)
		} else {
			reply, usage, err = streamChat(apiKey, cfg, compressed)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "\nError:", err)
			cw.Messages = cw.Messages[:len(cw.Messages)-1]
			continue
		}
		fmt.Print("\n")

		stats.Add(usage)
		fmt.Printf("%s  %s\n\n", formatTokenUsage(usage), stats.FormatTotal())

		cw.Messages = append(cw.Messages, message{Role: "assistant", Content: reply})

		if err := saveSessionCW(sessionName, cw); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to auto-save session: %v\n", err)
		}
	}
}
