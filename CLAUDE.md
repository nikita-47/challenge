# AI Challenge — CLI Chat with Claude

30-day AI challenge. Each day builds a new feature on top of the previous one.
See `TASKS.md` for all daily assignments and their status.

## Project structure

- `main.go` — app entry, CLI flags, config, banner, help (~120 lines)
- `chat.go` — interactive chat REPL loop (~80 lines)
- `api.go` — API client, request building, SSE streaming (~160 lines)
- `render.go` — ANSI markdown rendering (~25 lines)
- `env.go` — .env file loader (~20 lines)
- `compare.go` — split-screen TUI, panel rendering, comparison orchestrator (~1000 lines)
- `agent.go` — agentic loop with tool_use (run_shell, read_file) (~240 lines)
- `history.go` — session persistence (save/load/delete JSON) (~60 lines)
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

Example:
```
go run . --max-tokens 200 --format "bullet points" --stop "END"
```

### Chat commands

| Command | Description |
|---|---|
| `/help` | Show help and flag reference |
| `/clear` | Reset conversation history |
| `/system <text>` | Update system prompt mid-session |
| `/temp <question>` | Compare temperature 0 / 0.7 / 1.0 side-by-side |
| `/agent <task>` | Run agent with tools (shell, file read) |
| `/save [name]` | Save session (default: current session) |
| `/load <name>` | Load a named session |
| `exit` / `quit` | Quit |

## Key decisions

- **File layout**: `main.go` (entry/CLI), `chat.go` (REPL), `api.go` (API client), `render.go` (markdown), `env.go` (.env), `compare.go` (TUI), `agent.go` (agent), `history.go` (sessions)
- **No external deps**: uses only Go stdlib (net/http, encoding/json, etc.)
- **`.env` loading**: hand-rolled parser, no third-party dotenv library
- **Model**: claude-sonnet-4-5-20250929
- **Conversation history**: full message history is sent on each request for multi-turn context; auto-saved to `.chat_history/` as JSON after each exchange
- **Streaming**: uses SSE (`stream: true`), prints tokens as they arrive via `readStream()`
- **Format injection**: `--format` value is appended to system prompt as `"Always respond in this format: <value>"`

## Rules

- Keep it simple — avoid over-engineering
- No external dependencies unless absolutely necessary
- `.env` must never be committed
- Before starting a new day's task, read `TASKS.md` to understand what was built before
- Every new feature/tool must be accessible both from interactive chat (`/command`) and from CLI (`--flag`) so it can be used non-interactively
