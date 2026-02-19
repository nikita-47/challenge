# AI Challenge — CLI Chat with Claude

30-day AI challenge. Each day builds a new feature on top of the previous one.
See `TASKS.md` for all daily assignments and their status.

## Project structure

- `main.go` — app entry, chat loop, API client, markdown rendering (~290 lines)
- `compare.go` — split-screen TUI, panel rendering, comparison orchestrator (~430 lines)
- `TASKS.md` — daily task log (assignments, status, notes)
- `.env` — stores `ANTHROPIC_API_KEY` (not committed)

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
| `--format string` | — | Format instruction appended to system prompt |

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
| `exit` / `quit` | Quit |

## Key decisions

- **Two files**: `main.go` (chat/API) and `compare.go` (split-screen TUI) — keep it that way
- **No external deps**: uses only Go stdlib (net/http, encoding/json, etc.)
- **`.env` loading**: hand-rolled parser, no third-party dotenv library
- **Model**: claude-sonnet-4-5-20250929
- **Conversation history**: full message history is sent on each request for multi-turn context
- **Streaming**: uses SSE (`stream: true`), prints tokens as they arrive via `readStream()`
- **Format injection**: `--format` value is appended to system prompt as `"Always respond in this format: <value>"`

## Rules

- Keep it simple — avoid over-engineering
- No external dependencies unless absolutely necessary
- `.env` must never be committed
- Before starting a new day's task, read `TASKS.md` to understand what was built before
