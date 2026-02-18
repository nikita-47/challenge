# Challenge - CLI Chat with Claude

Simple interactive CLI chat tool that talks to Claude via the Anthropic API.

## Project structure

- `main.go` — single-file Go application, no external dependencies
- `.env` — stores `ANTHROPIC_API_KEY` (not committed)

## How to run

```
go run main.go [flags]
```

### CLI flags

| Flag | Default | Description |
|---|---|---|
| `--max-tokens int` | 1024 | Max response tokens |
| `--system string` | — | System prompt |
| `--stop string` | — | Stop sequence (sent as `stop_sequences` array) |
| `--format string` | — | Format instruction appended to system prompt |

Example:
```
go run main.go --max-tokens 200 --format "bullet points" --stop "END"
```

### Chat commands

| Command | Description |
|---|---|
| `/help` | Show help and flag reference |
| `/clear` | Reset conversation history |
| `/system <text>` | Update system prompt mid-session |
| `exit` / `quit` | Quit |

## Key decisions

- **Single file**: everything lives in `main.go` — keep it that way
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
