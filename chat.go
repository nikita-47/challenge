package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func runChat(apiKey, openaiKey string, cfg config) {
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
			result, err := agent.Run(task)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Agent error: %v\n\n", err)
			} else {
				fmt.Print("Claude: ")
				fmt.Println(renderMarkdown(result))
				fmt.Println()
			}
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
		fmt.Print("\n\n")

		history = append(history, message{Role: "assistant", Content: reply})
	}
}
