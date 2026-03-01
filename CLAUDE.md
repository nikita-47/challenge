# AI Challenge — CLI Chat with Claude

30-day AI challenge. Each day builds a new feature on top of the previous one.
See `TASKS.md` for all daily assignments and their status.

## Project structure

### Go backend
- `main.go` — app entry, CLI flags, config, banner, help
- `chat.go` — interactive chat REPL loop + CLI emit helpers (`cliAgentEmit`, `cliTokenWriter`)
- `api.go` — API client, request building, SSE streaming, OpenAI-compatible client
- `server.go` — HTTP server with SSE endpoints for Vue frontend
- `tokens.go` — token usage tracking and cost calculation
- `render.go` — ANSI markdown rendering (CLI only)
- `env.go` — .env file loader
- `compare.go` — split-screen TUI, panel rendering, comparison orchestrator
- `agent.go` — agentic loop with tool_use (run_shell, read_file), `AgentEvent` type
- `history.go` — session persistence (save/load/delete/list JSON)
- `compress.go` — context compression: summarize old messages, keep recent N as-is
- `strategy.go` — context strategy dispatcher (summary, window, facts, branch)
- `strategy_window.go` — sliding window strategy: keep last N messages
- `strategy_facts.go` — sticky facts strategy: extract key-value facts after each exchange
- `strategy_branch.go` — branching strategy: fork conversations into named branches

### Vue frontend (`frontend/`)
- `src/App.vue` — layout: sidebar + chat
- `src/stores/chat.ts` — Pinia: messages, streaming, tokens
- `src/stores/sessions.ts` — Pinia: session list, load/delete
- `src/composables/useSSE.ts` — fetch + ReadableStream SSE parser
- `src/stores/ui.ts` — Pinia: UI state (right panel, server config)
- `src/components/` — ChatWindow, MessageBubble, ToolCallCard, ChatInput, TokenBar, SessionPanel, ChatInfoPanel, NewChatDialog, BranchSelector
- `src/lib/types.ts` — TypeScript types mirroring Go events
- `src/lib/api.ts` — REST API client (sessions)
- `src/lib/utils.ts` — `cn()` utility (clsx + tailwind-merge)
- `src/components/ui/` — shadcn-vue components (Button, ScrollArea, Textarea, Checkbox, Collapsible, Badge, Card, Separator, Dialog, Input, Label, Select, Slider)

### Other
- `TASKS.md` — daily task log (assignments, status, notes)
- `.env` — stores `ANTHROPIC_API_KEY` (not committed)
- `.chat_history/` — saved chat sessions (not committed)

## How to run

```
go run . [flags]
```

> Note: always `go run .` (not `go run main.go`) — the project spans multiple files.

### CLI flags

| Flag | Default | Description |
|---|---|---|
| `--max-tokens int` | 1024 | Max response tokens |
| `--system string` | — | System prompt |
| `--stop string` | — | Stop sequence (sent as `stop_sequences` array) |
| `--temperature float` | API default | Sampling temperature (0.0–1.0) |
| `--format string` | — | Format instruction appended to system prompt |
| `--agent string` | — | Run agent with tools (shell, file read) and exit |
| `--session string` | — | Session name for chat history persistence |
| `--base-url string` | — | OpenAI-compatible base URL (e.g. `http://localhost:1234`) |
| `--model string` | — | Model name for OpenAI-compatible API |

| `--strategy string` | summary | Context strategy (summary, window, facts, branch) |
| `--window-size int` | 20 | Window size for window/facts strategies |
| `--server` | false | Start HTTP server with Vue UI |
| `--port int` | 8080 | HTTP server port |

Example:
```
go run . --max-tokens 200 --format "bullet points" --stop "END"
```

### Server mode (Vue UI)

```
# Dev: run Go backend + Vite dev server separately
go run . --server --port 8080
cd frontend && npm run dev     # Vite on :5173, proxies /api → :8080

# Production: build frontend, Go serves static + API
cd frontend && npm run build
go run . --server
# Open http://localhost:8080
```

### Chat commands

| Command | Description |
|---|---|
| `/help` | Show help and flag reference |
| `/clear` | Reset conversation history |
| `/system <text>` | Update system prompt mid-session |
| `/temp <question>` | Compare temperature 0 / 0.7 / 1.0 side-by-side |
| `/agent <task>` | Run agent with tools (shell, file read, sees chat context) |
| `/tokens` | Show token usage stats for current session |
| `/compress` | Show context compression status and summaries |
| `/strategy` | Show current context strategy |
| `/facts` | Show extracted sticky facts (facts strategy) |
| `/branch <name>` | Create a new branch from current point |
| `/switch <name>` | Switch to a named branch (or "main") |
| `/branches` | List all branches |
| `/save [name]` | Save session (default: current session) |
| `/load <name>` | Load a named session |
| `exit` / `quit` | Quit |

## Key decisions

- **File layout**: `main.go` (entry/CLI), `chat.go` (REPL), `api.go` (API client), `server.go` (HTTP/SSE), `render.go` (markdown), `env.go` (.env), `compare.go` (TUI), `agent.go` (agent), `history.go` (sessions), `compress.go` (context compression)
- **No external deps in Go**: uses only Go stdlib (net/http, encoding/json, etc.)
- **Frontend stack**: Vue 3 + Vite + Tailwind CSS v4 + Pinia + marked + shadcn-vue (radix-vue, class-variance-authority)
- **IO decoupling**: `readStream()` and `Agent.Run()` accept callbacks (`onToken func(string)` / `emit func(AgentEvent)`), CLI and HTTP use different implementations
- **`.env` loading**: hand-rolled parser, no third-party dotenv library
- **Model**: claude-sonnet-4-5-20250929
- **Conversation history**: last 10 messages sent as-is, older messages compressed via summary before new user message is appended; auto-saved to `.chat_history/` as JSON with summaries, system events, chat settings (model, temperature, maxTokens, system prompt), and cumulative token stats (exchanges, tokens in/out, tokens saved). Stats persist across reloads and session loads. Compression works in both CLI and HTTP endpoints.
- **System events**: stored inline in messages array as `role: "system"` with `event` field (`messageEvent` in Go, `SystemEvent` in TS). Used for compression notifications; extensible for future event types. Filtered out before sending to Claude API via `filterAPIMessages()`.
- **Streaming**: uses SSE (`stream: true`); CLI renders via `cliTokenWriter`, HTTP sends SSE events to browser
- **Format injection**: `--format` value is appended to system prompt as `"Always respond in this format: <value>"`
- **Context strategies**: pluggable via `strategy.go` dispatcher. Summary (default, rolling compression), Window (last N messages), Facts (extract key-value facts + last N), Branch (fork conversations). Strategy selected at chat creation, persisted in session settings.
- **HTTP API**: POST `/api/chat` (SSE stream, unified agent endpoint with tools), GET/DELETE `/api/sessions[/:name]`, GET `/api/sessions/:name/raw`, POST/GET `/api/sessions/:name/branches`, PUT `/api/sessions/:name/branch`

## Dev workflow (autonomous)

Dev servers should be running throughout the session. Start once at the beginning, restart Go only after `.go` changes.

### Check & start servers
Before starting a server, check if the port is already in use:
```bash
lsof -i :8080 -t  # empty = not running, PID = already running
lsof -i :5173 -t  # same for Vite (may use 5174 if 5173 busy)
```

Start only what's not running:
```bash
# Go backend (run in background)
go run . --server --port 8080

# Vite dev server with HMR (run in background, only once per session)
cd frontend && npm run dev
```

### When to restart
- **Go code changed** → kill Go server (`lsof -i :8080 -t | xargs kill`), rebuild and restart
- **Frontend code changed** → do nothing, Vite HMR picks it up automatically
- **Vite dev server** → never restart unless config changed (vite.config.ts, tailwind, etc.)

### Build & type check
Run before browser testing:
```bash
go build .                           # Go compilation
cd frontend && npx vue-tsc --noEmit  # TypeScript check
```

### Browser verification (Playwright MCP)
After implementing features, **always verify in browser** before reporting done:
1. Navigate to Vite dev URL (usually `http://localhost:5173` or `:5174`)
2. Test the feature end-to-end: interact with UI, send messages, verify responses
3. On issues: check `browser_console_messages` and `browser_network_requests`

## Rules

- Keep it simple — avoid over-engineering
- No external dependencies in Go unless absolutely necessary (frontend deps via npm are fine)
- `.env` must never be committed
- Before starting a new day's task, read `TASKS.md` to understand what was built before
- Every new feature/tool must be accessible both from interactive chat (`/command`) and from CLI (`--flag`) so it can be used non-interactively
- When testing context strategies, use the cheapest model (`claude-3-5-haiku-20241022`) and window size N=3 to minimize token costs
