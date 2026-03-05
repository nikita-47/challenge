# Chat with Claude

A chat application with a Vue UI and Go backend, powered by the Claude API. Built as a 35-day AI challenge — each day adds a new feature.

## Quick Start

1. **Get an API key** from [console.anthropic.com](https://console.anthropic.com)

2. **Create a `.env` file** in the project root:

   ```
   ANTHROPIC_API_KEY=sk-ant-...
   ```

3. **Start dev servers:**

   ```bash
   ./dev.sh start
   ```

4. **Open** [http://localhost:5173](http://localhost:5173)

## Features

- **Chat UI** — Vue 3 frontend with real-time SSE streaming
- **Sessions** — save, load, delete conversations; auto-persist with token stats
- **Memory** — 4-layer memory model: operator (immutable) → profile (long-term) → project (working) → short-term (chat). Auto-updated via LLM after each exchange
- **Context strategies** — pluggable: summary (rolling compression), window (last N), facts (key-value extraction), branch (fork conversations)
- **Agent mode** — agentic loop with tool use (shell commands, file reading)
- **Branching** — fork conversations into named branches, switch between them
- **Token tracking** — per-session usage stats with cost calculation
- **Compare modes** — temperature, model, and reasoning comparison (CLI)

## Architecture

```
backend/          Go server — API, SSE streaming, sessions, memory, agent
frontend/         Vue 3 + Vite + Tailwind + Pinia + shadcn-vue
.env              API key (not committed)
.chat_history/    Saved sessions (not committed)
.memory/          Memory layers as .md files (not committed)
go.mod            Go module (project root)
dev.sh            Dev server management
```

## Dev Workflow

```bash
./dev.sh start        # Start Go + Vite
./dev.sh stop         # Stop everything
./dev.sh restart-go   # After .go file changes
./dev.sh status       # Check what's running
```

Go changes require `./dev.sh restart-go`. Frontend changes are picked up by Vite HMR automatically.
