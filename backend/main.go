package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
)

type config struct {
	maxTokens    int
	temperature  float64
	system       string
	stop         string
	format       string
	compare      string
	tempCompare  string
	modelCompare string
	agent        string
	session      string
	verbose      bool
	baseURL      string
	model        string
	server       bool
	port         int
	strategy     string
	windowSize   int
}

func main() {
	cfg := parseArgs()

	apiKey := loadEnv(".env", "ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "ANTHROPIC_API_KEY not set in .env")
		os.Exit(1)
	}
	openaiKey := loadEnv(".env", "OPENAI_API_KEY")

	if cfg.compare != "" {
		scanner := bufio.NewScanner(os.Stdin)
		runComparison(apiKey, cfg, cfg.compare, scanner)
		return
	}

	if cfg.tempCompare != "" {
		scanner := bufio.NewScanner(os.Stdin)
		runTempComparison(apiKey, cfg, cfg.tempCompare, scanner)
		return
	}

	if cfg.modelCompare != "" {
		scanner := bufio.NewScanner(os.Stdin)
		runModelComparison(apiKey, openaiKey, cfg, cfg.modelCompare, scanner)
		return
	}

	if cfg.agent != "" {
		agent := newAgentWithTools(apiKey, cfg)
		result, err := agent.Run(cfg.agent, nil, cliAgentEmit)
		agent.Cleanup()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Agent error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(renderMarkdown(result))
		return
	}

	if cfg.server {
		startServer(apiKey, cfg)
		return
	}

	printBanner(cfg, openaiKey)
	runChat(apiKey, openaiKey, cfg)
}

func parseArgs() config {
	cfg := config{}
	flag.IntVar(&cfg.maxTokens, "max-tokens", 1024, "max response tokens")
	flag.StringVar(&cfg.system, "system", "", "system prompt")
	flag.StringVar(&cfg.stop, "stop", "", "stop sequence")
	flag.StringVar(&cfg.format, "format", "", "response format instruction")
	flag.Float64Var(&cfg.temperature, "temperature", -1, "sampling temperature (0.0–1.0, default: API default)")
	flag.StringVar(&cfg.compare, "compare", "", "run 4-way comparison and exit")
	flag.StringVar(&cfg.tempCompare, "tempcompare", "", "run 3-way temperature comparison and exit")
	flag.StringVar(&cfg.modelCompare, "models", "", "run 3-way model comparison and exit")
	flag.StringVar(&cfg.agent, "agent", "", "run agent with tools and exit")
	flag.StringVar(&cfg.session, "session", "", "session name for chat history")
	flag.BoolVar(&cfg.verbose, "verbose", false, "print each request as curl before sending")
	flag.StringVar(&cfg.baseURL, "base-url", "", "OpenAI-compatible base URL (e.g. http://localhost:1234)")
	flag.StringVar(&cfg.model, "model", "", "model name for OpenAI-compatible API")
	flag.BoolVar(&cfg.server, "server", false, "start HTTP server with Vue UI")
	flag.IntVar(&cfg.port, "port", 8080, "HTTP server port")
	flag.StringVar(&cfg.strategy, "strategy", "", "context strategy: summary|window|facts|branch")
	flag.IntVar(&cfg.windowSize, "window-size", 20, "number of messages to keep for window/facts strategies")
	flag.Parse()
	return cfg
}

func printBanner(cfg config, openaiKey string) {
	fmt.Println("=== Claude CLI Chat ===")
	if cfg.baseURL != "" {
		modelName := cfg.model
		if modelName == "" {
			modelName = "(default)"
		}
		fmt.Printf("Endpoint:   %s\n", cfg.baseURL)
		fmt.Printf("Model:      %s\n", modelName)
	} else {
		p := PricingFor(DefaultModel)
		fmt.Printf("Model:      %s ($%.0f/$%.0f per 1M tokens)\n", DefaultModel, p.CostIn, p.CostOut)
	}
	fmt.Printf("Max tokens: %d\n", cfg.maxTokens)
	if cfg.system != "" {
		fmt.Printf("System:     %s\n", cfg.system)
	}
	if cfg.temperature >= 0 {
		fmt.Printf("Temperature:%.1f\n", cfg.temperature)
	}
	if cfg.stop != "" {
		fmt.Printf("Stop:       %q\n", cfg.stop)
	}
	if cfg.format != "" {
		fmt.Printf("Format:     %s\n", cfg.format)
	}
	if cfg.verbose {
		fmt.Printf("Verbose:    on (curl output to stderr)\n")
	}
	if cfg.strategy != "" {
		fmt.Printf("Strategy:   %s\n", cfg.strategy)
		if cfg.strategy == strategyWindow || cfg.strategy == strategyFacts {
			fmt.Printf("Window:     %d messages\n", cfg.windowSize)
		}
	}
	if cfg.session != "" {
		fmt.Printf("Session:    %s\n", cfg.session)
	}
	if openaiKey != "" {
		fmt.Printf("OpenAI:     loaded\n")
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
	fmt.Println("  /temp <question>     — compare temperature 0 / 0.7 / 1.0 side-by-side")
	fmt.Println("  /models <question>   — compare weak/medium/strong models side-by-side")
	fmt.Println("  /agent <task>        — run agent with tools (shell, file read)")
	fmt.Println("  /task <goal>         — run agent in task mode (planning → executing → validating → done)")
	fmt.Println("  /task                — show current task state")
	fmt.Println("  /resume              — resume a paused/interrupted task")
	fmt.Println("  /tokens              — show token usage stats for current session")
	fmt.Println("  /compress            — show context compression status and summaries")
	fmt.Println("  /save [name]         — save session (default: current session)")
	fmt.Println("  /load <name>         — load a named session")
	fmt.Println("  /strategy            — show current context strategy and state")
	fmt.Println("  /facts               — show extracted facts (facts strategy)")
	fmt.Println("  /branch <name>       — create a new branch (branch strategy)")
	fmt.Println("  /switch <name>       — switch to a branch (branch strategy)")
	fmt.Println("  /branches            — list all branches (branch strategy)")
	fmt.Println("  exit / quit          — quit")
	fmt.Println()
	fmt.Println("Flags (set at startup):")
	fmt.Println("  --max-tokens int    max response tokens (default 1024)")
	fmt.Println("  --system string     system prompt")
	fmt.Println("  --stop string       stop sequence")
	fmt.Println("  --format string     response format instruction")
	fmt.Println("  --temperature float  sampling temperature (0.0–1.0)")
	fmt.Println("  --compare string    run 4-way comparison directly and exit")
	fmt.Println("  --tempcompare str   run 3-way temperature comparison and exit")
	fmt.Println("  --models string     run 3-way model comparison and exit")
	fmt.Println("  --agent string      run agent with tools and exit")
	fmt.Println("  --session string    session name for chat history")
	fmt.Println("  --verbose           print each request as curl before sending")
	fmt.Println("  --base-url string   OpenAI-compatible base URL (e.g. http://localhost:1234)")
	fmt.Println("  --model string      model name for OpenAI-compatible API")
	fmt.Println("  --strategy string   context strategy: summary|window|facts|branch")
	fmt.Println("  --window-size int   messages to keep for window/facts (default 20)")
	fmt.Println()
}
