# Challenge - CLI Chat with Claude

Simple interactive CLI chat tool that talks to Claude via the Anthropic API.

## Project structure

- `main.go` — single-file Go application, no external dependencies
- `.env` — stores `ANTHROPIC_API_KEY` (not committed)

## How to run

```
go run main.go
```

## Key decisions

- **Single file**: everything lives in `main.go` — keep it that way
- **No external deps**: uses only Go stdlib (net/http, encoding/json, etc.)
- **`.env` loading**: hand-rolled parser, no third-party dotenv library
- **Model**: claude-sonnet-4-5-20250929
- **Conversation history**: full message history is sent on each request for multi-turn context

## Rules

- Keep it simple — avoid over-engineering
- No external dependencies unless absolutely necessary
- `.env` must never be committed
