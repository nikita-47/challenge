package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// cliTokenWriter buffers streaming text and renders markdown to stdout line by line.
type cliTokenWriter struct {
	pending strings.Builder
}

func (w *cliTokenWriter) OnToken(text string) {
	w.pending.WriteString(text)
	buf := w.pending.String()
	if i := strings.LastIndex(buf, "\n"); i >= 0 {
		fmt.Print(renderMarkdown(buf[:i+1]))
		w.pending.Reset()
		w.pending.WriteString(buf[i+1:])
	}
}

func (w *cliTokenWriter) Flush() {
	if w.pending.Len() > 0 {
		fmt.Print(renderMarkdown(w.pending.String()))
		w.pending.Reset()
	}
}

// cliAgentEmit prints agent events to stdout in the same format as the old Run method.
func cliAgentEmit(ev AgentEvent) {
	switch ev.Type {
	case "turn":
		fmt.Printf("\033[2m[Agent] Turn %d/%d — calling API...\033[0m\n", ev.Turn, ev.MaxTurn)
	case "thinking":
		fmt.Printf("\033[2m[Agent] Thinking: %s\033[0m\n", truncate(ev.Text, 100))
	case "tool_call":
		fmt.Printf("\033[33m[Agent] Tool: %s\033[0m\n", ev.Tool)
		if ev.Tool == "run_shell" {
			var input struct{ Command string }
			json.Unmarshal(ev.Input, &input)
			fmt.Printf("\033[33m[Agent]   $ %s\033[0m\n", input.Command)
		} else if ev.Tool == "read_file" {
			var input struct{ Path string }
			json.Unmarshal(ev.Input, &input)
			fmt.Printf("\033[33m[Agent]   path: %s\033[0m\n", input.Path)
		}
	case "tool_result":
		fmt.Printf("\033[2m[Agent]   Result: %s\033[0m\n", truncate(ev.Output, 200))
	case "usage":
		if ev.Usage != nil {
			fmt.Printf("\033[2m[Agent] %s\033[0m\n", formatTokenUsage(*ev.Usage))
		}
	case "done":
		if ev.Stats != nil {
			fmt.Printf("\033[1m[Agent] Done after %d turn(s). %s\033[0m\n\n", ev.Turn, ev.Stats.FormatTotal())
		}
	case "error":
		fmt.Fprintf(os.Stderr, "\033[31m[Agent] Error: %s\033[0m\n", ev.Text)
	case "text":
		fmt.Printf("\033[1m[Agent] Goal: %s\033[0m\n", ev.Text)
	case "task_state":
		var ts TaskState
		if json.Unmarshal([]byte(ev.Text), &ts) == nil {
			fmt.Printf("\033[36m[Task] Phase: %s", ts.Phase)
			if len(ts.Steps) > 0 {
				fmt.Printf(" | Steps: %d/%d", ts.completedCount(), len(ts.Steps))
			}
			if ts.Paused {
				fmt.Printf(" | Paused")
			}
			fmt.Printf("\033[0m\n")
		}
	case "step_result":
		var sr StepResult
		if json.Unmarshal([]byte(ev.Text), &sr) == nil {
			fmt.Printf("\033[36m[Task] Step %d [%s]: %s\033[0m\n", sr.Index, sr.Status, truncate(sr.Output, 100))
		}
	}
}

func runChat(apiKey, openaiKey string, cfg config) {
	scanner := bufio.NewScanner(os.Stdin)
	sessionName := cfg.session
	cw := &contextWindow{}
	loaded, err := loadSessionCW(sessionName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load session: %v\n", err)
	}
	if loaded != nil {
		cw = loaded
	}
	// Apply strategy from CLI flags if session has no strategy set.
	if cw.Settings == nil {
		cw.Settings = &sessionSettings{}
	}
	if cfg.strategy != "" && cw.Settings.Strategy == "" {
		cw.Settings.Strategy = cfg.strategy
	}
	if cfg.windowSize > 0 && cw.Settings.WindowSize == 0 {
		cw.Settings.WindowSize = cfg.windowSize
	}

	stats := cw.Stats.toTokenStats()
	if len(cw.Messages) > 0 {
		name := sessionName
		if name == "" {
			name = "default"
		}
		fmt.Printf("Resumed session '%s' (%d messages", name, len(cw.Messages))
		if cw.Summary != "" {
			fmt.Printf(", has summary")
		}
		if s := getStrategy(cw); s != strategySummary {
			fmt.Printf(", strategy: %s", s)
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
				fmt.Printf("No summary yet (compression triggers at %d messages).\n", compressThreshold(cw))
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
			agent := newAgentWithTools(apiKey, cfg)
			result, err := agent.Run(task, cw.Messages, cliAgentEmit)
			agent.Cleanup()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Agent error: %v\n\n", err)
			} else {
				fmt.Print("Claude: ")
				fmt.Println(renderMarkdown(result))
				fmt.Println()

				// Add flattened messages to history for persistence.
				appendMessage(cw, message{Role: "user", Content: task})
				appendMessage(cw, message{Role: "assistant", Content: result})

				cw.Stats = statsFromToken(stats)
				if err := saveSessionCW(sessionName, cw); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to auto-save session: %v\n", err)
				}
			}
			// Merge agent stats into session stats.
			stats.TotalInput += agent.Stats.TotalInput
			stats.TotalOutput += agent.Stats.TotalOutput
			stats.Exchanges += agent.Stats.Exchanges
			continue
		case input == "/task":
			if cw.TaskState == nil {
				fmt.Println("No active task. Use /task <goal> to start one.")
			} else {
				fmt.Println(cw.TaskState.FormatStatus())
			}
			fmt.Println()
			continue
		case strings.HasPrefix(input, "/task "):
			goal := strings.TrimPrefix(input, "/task ")
			cw.TaskState = &TaskState{
				Goal:      goal,
				Phase:     PhasePlanning,
				Artifacts: make(map[string]string),
			}
			fmt.Printf("\033[1m[Task] Starting: %s\033[0m\n", goal)
			if err := runTaskPhase(apiKey, cfg, cw, nil, cliAgentEmit, cfg.baseURL, cfg.model); err != nil {
				fmt.Fprintf(os.Stderr, "Task error: %v\n\n", err)
			}
			fmt.Println(cw.TaskState.FormatStatus())
			appendMessage(cw, message{Role: "user", Content: goal})
			appendMessage(cw, message{Role: "assistant", Content: "[task phase: " + cw.TaskState.Phase + "]"})
			cw.Stats = statsFromToken(stats)
			if err := saveSessionCW(sessionName, cw); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to auto-save session: %v\n", err)
			}
			continue
		case input == "/resume":
			if cw.TaskState == nil {
				fmt.Println("No task to resume.")
				fmt.Println()
				continue
			}
			if cw.TaskState.Phase == PhaseDone {
				fmt.Println("Task already completed.")
				fmt.Println()
				continue
			}
			if !cw.TaskState.Paused {
				fmt.Printf("Task is not paused (current phase: %s).\n\n", cw.TaskState.Phase)
				continue
			}
			cw.TaskState.Paused = false
			fmt.Printf("\033[1m[Task] Resuming phase: %s\033[0m\n", cw.TaskState.Phase)
			if err := runTaskPhase(apiKey, cfg, cw, nil, cliAgentEmit, cfg.baseURL, cfg.model); err != nil {
				fmt.Fprintf(os.Stderr, "Task error: %v\n\n", err)
			}
			fmt.Println(cw.TaskState.FormatStatus())
			appendMessage(cw, message{Role: "user", Content: "Continue the task"})
			appendMessage(cw, message{Role: "assistant", Content: "[task phase: " + cw.TaskState.Phase + "]"})
			cw.Stats = statsFromToken(stats)
			if err := saveSessionCW(sessionName, cw); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to auto-save session: %v\n", err)
			}
			continue
		case input == "/save" || strings.HasPrefix(input, "/save "):
			saveName := strings.TrimPrefix(input, "/save")
			saveName = strings.TrimSpace(saveName)
			if saveName == "" {
				saveName = sessionName
			}
			cw.Stats = statsFromToken(stats)
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
				stats = cw.Stats.toTokenStats()
				fmt.Printf("Loaded session '%s' (%d messages", loadName, len(cw.Messages))
				if cw.Summary != "" {
					fmt.Printf(", has summary")
				}
				fmt.Println(").")
			}
			fmt.Println()
			continue
		case input == "/strategy":
			s := getStrategy(cw)
			fmt.Printf("Strategy: %s\n", s)
			if s == strategyWindow || s == strategyFacts {
				fmt.Printf("Window size: %d\n", getWindowSize(cw))
			}
			if s == strategyFacts && len(cw.Facts) > 0 {
				fmt.Printf("Facts: %d entries\n", len(cw.Facts))
			}
			if s == strategyBranch {
				branches := listBranches(cw)
				active := cw.ActiveBranch
				if active == "" {
					active = "main"
				}
				fmt.Printf("Branches: %s (active: %s)\n", strings.Join(branches, ", "), active)
			}
			fmt.Println()
			continue
		case input == "/facts":
			if getStrategy(cw) != strategyFacts {
				fmt.Println("Facts are only available with the 'facts' strategy.")
			} else if len(cw.Facts) == 0 {
				fmt.Println("No facts extracted yet.")
			} else {
				fmt.Println("Current facts:")
				for k, v := range cw.Facts {
					fmt.Printf("  %s: %s\n", k, v)
				}
			}
			fmt.Println()
			continue
		case strings.HasPrefix(input, "/branch "):
			name := strings.TrimSpace(strings.TrimPrefix(input, "/branch "))
			if getStrategy(cw) != strategyBranch {
				fmt.Println("Branching is only available with the 'branch' strategy.")
			} else if err := createBranch(cw, name); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			} else {
				fmt.Printf("Created and switched to branch '%s'.\n", name)
				cw.Stats = statsFromToken(stats)
				_ = saveSessionCW(sessionName, cw)
			}
			fmt.Println()
			continue
		case strings.HasPrefix(input, "/switch "):
			name := strings.TrimSpace(strings.TrimPrefix(input, "/switch "))
			if getStrategy(cw) != strategyBranch {
				fmt.Println("Branch switching is only available with the 'branch' strategy.")
			} else if err := switchBranch(cw, name); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			} else {
				msgs := activeMessages(cw)
				fmt.Printf("Switched to branch '%s' (%d messages).\n", name, len(msgs))
				cw.Stats = statsFromToken(stats)
				_ = saveSessionCW(sessionName, cw)
			}
			fmt.Println()
			continue
		case input == "/branches":
			if getStrategy(cw) != strategyBranch {
				fmt.Println("Branching is only available with the 'branch' strategy.")
			} else {
				active := cw.ActiveBranch
				if active == "" {
					active = "main"
				}
				for _, name := range listBranches(cw) {
					marker := "  "
					if name == active {
						marker = "* "
					}
					count := len(cw.Messages)
					if name != "main" {
						for _, b := range cw.Branches {
							if b.Name == name {
								count = b.ForkIndex + len(b.Messages)
								break
							}
						}
					}
					fmt.Printf("%s%s (%d messages)\n", marker, name, count)
				}
			}
			fmt.Println()
			continue
		}

		// Pre-process context (compress for summary strategy, noop for others).
		ci, compErr := maybeProcess(apiKey, cw, &stats)
		if compErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: context processing failed, continuing with full history: %v\n", compErr)
		} else if ci != nil {
			appendMessage(cw, message{
				Role: "system",
				Event: &messageEvent{
					Type:         "compress",
					MessageCount: ci.MessageCount,
					SummaryLen:   ci.SummaryLen,
					TokensSaved:  ci.TokensSaved,
				},
			})
			fmt.Printf("\033[2m[compressed %d messages → summary (%d chars) | saved ~%d tokens]\033[0m\n",
				ci.MessageCount, ci.SummaryLen, ci.TokensSaved)
		}

		appendMessage(cw, message{Role: "user", Content: input})
		compressed := buildAPIMessages(cw)

		fmt.Print("\nClaude: ")
		tw := &cliTokenWriter{}
		var reply string
		var usage tokenUsage
		if cfg.baseURL != "" {
			reply, usage, err = streamChatOpenAI(cfg.baseURL, cfg.model, cfg, compressed, tw.OnToken)
		} else {
			reply, usage, err = streamChat(apiKey, cfg, compressed, tw.OnToken)
		}
		tw.Flush()
		if err != nil {
			fmt.Fprintln(os.Stderr, "\nError:", err)
			cw.Messages = cw.Messages[:len(cw.Messages)-1]
			continue
		}
		fmt.Print("\n")

		stats.Add(usage)
		fmt.Printf("%s  %s\n\n", formatTokenUsage(usage), stats.FormatTotal())

		appendMessage(cw, message{Role: "assistant", Content: reply})

		// Post-process: extract facts for facts strategy.
		if getStrategy(cw) == strategyFacts {
			if err := maybeExtractFacts(apiKey, cw, &stats); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: fact extraction failed: %v\n", err)
			} else if len(cw.Facts) > 0 {
				fmt.Printf("\033[2m[facts updated: %d entries]\033[0m\n", len(cw.Facts))
			}
		}

		cw.Stats = statsFromToken(stats)
		if err := saveSessionCW(sessionName, cw); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to auto-save session: %v\n", err)
		}
	}
}
