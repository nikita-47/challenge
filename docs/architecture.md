# Architecture

## Overview

AI Challenge is a full-stack chat application with Claude AI integration. Built as a 35-day progressive challenge where each day adds a new feature.

**Stack:**
- Backend: Go (single binary, package main, 27 files)
- Frontend: Vue 3 + Vite + Tailwind CSS v4 + Pinia + shadcn-vue
- AI: Claude API (Anthropic) + local LLM support (OpenAI-compatible)
- Embeddings: Ollama (nomic-embed-text, local)
- MCP: Model Context Protocol servers (Go, stdio transport)

## Backend Architecture

All Go code lives in `backend/` as a single `main` package. Entry point: `main.go`.

### Core Modules

| File | Responsibility |
|------|---------------|
| `server.go` | HTTP server, route registration, `handleChat()`, SSE helpers, task phase orchestrator |
| `api.go` | Request/response types, `streamChat()`, `streamChatOpenAI()`, SSE stream readers |
| `agent.go` | Agentic loop with tools, `Agent.Run()`, tool execution, history compaction |
| `taskstate.go` | Task state machine, phase prompts (proposing/planning/executing/validating) |
| `history.go` | Session persistence (JSON files in `.chat_history/`) |
| `strategy.go` | Context strategy dispatch (window, summary, facts, branch) |
| `mcp.go` | MCP client manager, HTTP handlers, tool name parsing |
| `docs.go` | Document store, upload/index/search HTTP handlers |
| `rag.go` | RAG pipeline: query rewrite, embed, search, filter, XML inject |
| `embeddings.go` | Ollama embedding client (`nomic-embed-text`) |
| `indexer.go` | Combined index builder (4 chunking strategies) |
| `chunker.go` | 4 chunking strategies: size, sentence, structure (markdown), semantic |
| `similarity.go` | Cosine similarity, chunk search, response builder |
| `memory.go` | Memory layer CRUD (profiles, projects, operators) |
| `models.go` | Model constants and pricing |
| `tokens.go` | Token usage tracking and cost calculation |

### Chat Flow

1. `POST /api/chat` -> `handleChat()`
2. Load session from `.chat_history/<name>.json`
3. Apply strategy, memory layers, system prompt
4. Context compression if needed (`maybeProcess`)
5. RAG pipeline if documents selected (`performRAGSearch`)
6. Route by provider:
   - **Task mode** -> `runTaskPhase()` (multi-phase agent)
   - **Local/Railway** -> `streamChatOpenAI()` (OpenAI-compatible)
   - **Claude** -> `Agent.Run()` (with optional MCP tools)
7. Post-process: facts extraction, memory update
8. Save session

### Agent System

The `Agent` struct runs a multi-turn agentic loop:
- Built-in tools: `run_shell`, `read_file`
- Phase tools: `submit_plan`, `submit_phases`, `submit_validation`, `report_step`
- MCP tools: routed via `MCPManager` using `server__toolname` naming
- History compaction: old tool results compressed to one-liners after 4 messages

### Task Phases

Dynamic pipeline proposed by the agent:
```
proposing -> [planning -> executing -> validating] -> done
```

- **Proposing**: Agent analyzes goal, proposes phase pipeline via `submit_phases`
- **Planning**: Creates structured plan via `submit_plan`
- **Executing**: Runs plan steps using tools, reports via `report_step`
- **Validating**: Verifies results via `submit_validation`
- User approval required between phases (paused state)

### Memory Model

4 layers, stored in `.memory/`:
- **Operator** (`.memory/operators/`): Immutable identity, highest priority
- **Profile** (`.memory/profiles/`): Long-term user profile
- **Project** (`.memory/projects/`): Working project context
- **Short-term**: Chat messages in session

Assembled in `buildFullSystemPrompt()`: operator -> profile -> project -> system.
Auto-updated via Haiku after each exchange (`maybeUpdateMemory()`).

### RAG Pipeline

1. **Query Rewrite** (optional): Haiku rewrites question for semantic search
2. **Embed**: Ollama `nomic-embed-text` generates query embedding
3. **Search**: Cosine similarity across 4 chunking strategies per document
4. **Filter**: Threshold-based filtering (configurable)
5. **Inject**: XML `<documents>` block with citation rules prepended to message

Indexing pipeline (`runIndexPipeline`): upload -> 4 chunking strategies -> embed each chunk -> save CombinedIndex JSON.

### Provider Support

Three providers via `providerSettings`:
- **claude**: Anthropic API with streaming
- **local**: Any OpenAI-compatible API (LM Studio, etc.)
- **railway**: Remote Ollama on Railway with auth proxy

## Frontend Architecture

Vue 3 SPA with Composition API and `<script setup>`.

### Stores (Pinia)

| Store | Purpose |
|-------|---------|
| `useChatStore` | Messages, streaming, SSE events, RAG, MCP, task state |
| `useUIStore` | Sidebar, activeView (chat/pipeline/docs), provider settings |
| `useSessionsStore` | Session CRUD |
| `useMemoryStore` | Memory layer CRUD |
| `useMCPStore` | MCP server connections, tools |
| `usePipelineStore` | Pipeline runs via MCP |
| `useDocsStore` | Document management, index polling |

### Key Components

| Component | Purpose |
|-----------|---------|
| `ChatWindow.vue` | Message list with auto-scroll |
| `ChatInput.vue` | Text input with settings popover |
| `MessageBubble.vue` | Message rendering with markdown, tools, RAG |
| `SessionPanel.vue` | Left sidebar (sessions, memory, MCP tabs) |
| `SendSettingsPopover.vue` | Task mode, tools, MCP tools, RAG settings |
| `DocsView.vue` | Document management, 4-strategy chunk comparison |
| `PipelineView.vue` | HN pipeline execution view |
| `RAGPipelineSteps.vue` | RAG step progress in message stream |
| `RAGSourcesBlock.vue` | Citation sources block |

### Views

`activeView` in UI store switches full-screen layouts:
- `chat`: Main chat interface
- `pipeline`: Pipeline execution view
- `docs`: Document management view

### SSE Streaming

`useSSE.ts` -> `streamRequest()`: POST fetch with ReadableStream, parses `data: {...}\n` format. Chat store handles 17 event types.

## MCP Servers

All in `mcp-servers/`, using `github.com/mark3labs/mcp-go` v0.45.0, stdio transport.

| Server | Tools | Purpose |
|--------|-------|---------|
| `hackernews` | 4 | HackerNews API (top stories, items, users, search) |
| `scheduler` | 6 | Task scheduler (reminder, url_monitor, hn_digest, pipeline) |
| `pipeline` | 7 | HN search -> Claude summary -> file save |
| `devtools` | 5 | Project context (git, files) |

Config: `.mcp_servers.json` (Claude Desktop format).

## File Storage

| Directory | Content | Committed |
|-----------|---------|-----------|
| `.chat_history/` | Session JSON files | No |
| `.memory/` | Memory layer files | No |
| `.documents/` | Uploaded docs + indexes | No |
| `.sandbox/` | Agent sandboxes (temp) | No |
| `docs/` | Project documentation | Yes |
| `mcp-servers/` | MCP server source code | Yes |
| `deploy/` | Railway deployment files | Yes |
