# AI Challenge — Chat with Claude

30-day AI challenge. Each day builds a new feature on top of the previous one.
See `TASKS.md` for all daily assignments and their status.

## Project structure

### Go backend (`backend/`)
- `backend/main.go` — app entry, CLI flags, config, banner, help
- `backend/chat.go` — interactive chat REPL loop + CLI emit helpers (`cliAgentEmit`, `cliTokenWriter`)
- `backend/api.go` — API client, request building, SSE streaming, OpenAI-compatible client
- `backend/server.go` — HTTP server with SSE endpoints for Vue frontend
- `backend/tokens.go` — token usage tracking and cost calculation
- `backend/render.go` — ANSI markdown rendering (CLI only)
- `backend/env.go` — .env file loader
- `backend/compare.go` — split-screen TUI, panel rendering, comparison orchestrator
- `backend/agent.go` — agentic loop with tool_use (run_shell, read_file), `AgentEvent` type
- `backend/history.go` — session persistence (save/load/delete/list JSON)
- `backend/compress.go` — context compression: summarize old messages, keep recent N as-is
- `backend/strategy.go` — context strategy dispatcher (summary, window, facts, branch)
- `backend/strategy_window.go` — sliding window strategy: keep last N messages
- `backend/strategy_facts.go` — sticky facts strategy: extract key-value facts after each exchange
- `backend/strategy_branch.go` — branching strategy: fork conversations into named branches
- `backend/memory.go` — memory layer CRUD (profiles + projects + operators as .md files in `.memory/`)
- `backend/memory_update.go` — auto-update profile/project memory via LLM after each exchange

### Vue frontend (`frontend/`)
- `src/App.vue` — layout: sidebar + chat
- `src/stores/chat.ts` — Pinia: messages, streaming, tokens
- `src/stores/sessions.ts` — Pinia: session list, load/delete
- `src/stores/memory.ts` — Pinia: profiles/projects/operators lists, CRUD
- `src/composables/useSSE.ts` — fetch + ReadableStream SSE parser
- `src/stores/ui.ts` — Pinia: UI state (right panel, server config)
- `src/components/` — ChatWindow, MessageBubble, ToolCallCard, ChatInput, TokenBar, SessionPanel, ChatInfoPanel, NewChatDialog, BranchSelector, MemoryEditorDialog
- `src/lib/types.ts` — TypeScript types mirroring Go events
- `src/lib/api.ts` — REST API client (sessions)
- `src/lib/utils.ts` — `cn()` utility (clsx + tailwind-merge)
- `src/components/ui/` — shadcn-vue components (Button, ScrollArea, Textarea, Checkbox, Collapsible, Badge, Card, Separator, Dialog, Input, Label, Select, Slider)

### Other
- `TASKS.md` — daily task log (assignments, status, notes)
- `.env` — stores `ANTHROPIC_API_KEY` (not committed)
- `.chat_history/` — saved chat sessions (not committed)
- `.memory/` — memory layers: `profiles/` (long-term) + `projects/` (working) + `operators/` (immutable user identity), .md files (not committed)
- `.claude/agents/` — sub-agent configs: `go.md` (backend, sonnet), `frontend.md` (Vue/TS, sonnet), `qa.md` (tester, haiku)

## How to run

### Server mode (Vue UI) — primary

```bash
# Dev: run Go backend + Vite dev server
./dev.sh start
# Open http://localhost:5173

# Production: build frontend, Go serves static + API
cd frontend && npm run build
go run ./backend --server
# Open http://localhost:8080
```

### CLI mode

```
go run ./backend [flags]
```

> Note: `go.mod` lives in the project root. All relative paths (.env, .chat_history, .memory, frontend/dist) resolve from CWD (project root), not from `backend/`.

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
go run ./backend --max-tokens 200 --format "bullet points" --stop "END"
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

- **File layout**: `backend/` (all Go files), `frontend/` (Vue app), `go.mod` in project root
- **No external deps in Go**: uses only Go stdlib (net/http, encoding/json, etc.)
- **Frontend stack**: Vue 3 + Vite + Tailwind CSS v4 + Pinia + marked + shadcn-vue (radix-vue, class-variance-authority)
- **IO decoupling**: `readStream()` and `Agent.Run()` accept callbacks (`onToken func(string)` / `emit func(AgentEvent)`), CLI and HTTP use different implementations
- **`.env` loading**: hand-rolled parser, no third-party dotenv library
- **Model**: claude-sonnet-4-5-20250929
- **Conversation history**: last 10 messages sent as-is, older messages compressed via summary before new user message is appended; auto-saved to `.chat_history/` as JSON with summaries, system events, chat settings (model, temperature, maxTokens, system prompt), and cumulative token stats (exchanges, tokens in/out, tokens saved). Stats persist across reloads and session loads. Compression works in both CLI and HTTP endpoints.
- **System events**: stored inline in messages array as `role: "system"` with `event` field (`messageEvent` in Go, `SystemEvent` in TS). Used for compression notifications; extensible for future event types. Filtered out before sending to Claude API via `filterAPIMessages()`.
- **Streaming**: uses SSE (`stream: true`); CLI renders via `cliTokenWriter`, HTTP sends SSE events to browser
- **Format injection**: `--format` value is appended to system prompt as `"Always respond in this format: <value>"`
- **Context strategies**: pluggable via `backend/strategy.go` dispatcher. Summary (default, rolling compression), Window (last N messages), Facts (extract key-value facts + last N), Branch (fork conversations). Strategy selected at chat creation, persisted in session settings.
- **HTTP API**: POST `/api/chat` (SSE stream, unified agent endpoint with tools), GET/DELETE `/api/sessions[/:name]`, GET `/api/sessions/:name/raw`, POST/GET `/api/sessions/:name/branches`, PUT `/api/sessions/:name/branch`, GET/POST `/api/memory/profiles`, GET/PUT/DELETE `/api/memory/profiles/:name`, GET/POST `/api/memory/projects`, GET/PUT/DELETE `/api/memory/projects/:name`, GET/POST `/api/memory/operators`, GET/PUT/DELETE `/api/memory/operators/:name`
- **Memory model**: 4 layers — short-term (chat messages), working (`.memory/projects/*.md`), long-term (`.memory/profiles/*.md`), operator (`.memory/operators/*.md`, immutable). Operator → profile → project → system prompt order in `buildFullSystemPrompt()`. Profile/project auto-updated via `maybeUpdateMemory()` (Haiku) after each exchange. Persisted in `sessionSettings`.

## Dev workflow (autonomous)

Dev servers should be running throughout the session. Start once at the beginning, restart Go only after `.go` changes.

### Dev servers (`dev.sh`)
```bash
./dev.sh start        # запустить Go + Vite
./dev.sh stop         # остановить всё (Go + Vite + Playwright)
./dev.sh restart-go   # после изменений .go файлов
./dev.sh status       # проверить что запущено
```

Отдельные команды: `start-go`, `stop-go`, `start-vite`, `stop-vite`, `stop-playwright`.

- **Go code changed** → `./dev.sh restart-go`
- **Frontend code changed** → ничего, Vite HMR подхватит
- **Vite dev server** → не перезапускать, только если изменился config

### Browser testing mode (local LLM)
Для UI-тестирования использовать локальную LLM вместо Claude API:
```bash
./dev.sh start-test   # LM Studio + Go (с local LLM) + Vite
./dev.sh stop-test    # остановить всё
```
Модель: qwen2.5-0.5b-instruct-mlx (0.5B, мгновенные ответы, 0 токенов).

### Build & type check
Run before browser testing:
```bash
go build -o challenge ./backend      # Go compilation
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
- Не использовать Python — все задачи решаются Go, bash или Node
