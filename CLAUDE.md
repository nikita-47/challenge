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

### Vue frontend (`frontend/`)
- `src/App.vue` — layout: sidebar + chat
- `src/stores/chat.ts` — Pinia: messages, streaming, tokens
- `src/stores/sessions.ts` — Pinia: session list, load/delete
- `src/composables/useSSE.ts` — fetch + ReadableStream SSE parser
- `src/stores/ui.ts` — Pinia: UI state (right panel, server config)
- `src/components/` — ChatWindow, MessageBubble, ToolCallCard, ChatInput, TokenBar, SessionPanel, ChatInfoPanel, NewChatDialog
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
- **Conversation history**: last 10 messages sent as-is, older messages compressed via summary before new user message is appended; auto-saved to `.chat_history/` as JSON with summaries
- **Streaming**: uses SSE (`stream: true`); CLI renders via `cliTokenWriter`, HTTP sends SSE events to browser
- **Format injection**: `--format` value is appended to system prompt as `"Always respond in this format: <value>"`
- **HTTP API**: POST `/api/chat` (SSE stream), POST `/api/agent` (SSE stream), GET/DELETE `/api/sessions[/:name]`

## Rules

- Keep it simple — avoid over-engineering
- No external dependencies in Go unless absolutely necessary (frontend deps via npm are fine)
- `.env` must never be committed
- Before starting a new day's task, read `TASKS.md` to understand what was built before
- Every new feature/tool must be accessible both from interactive chat (`/command`) and from CLI (`--flag`) so it can be used non-interactively
