# Developer Assistant — Day 31

**Date:** 2026-03-31

## Task

Create a developer assistant that understands the project: generate documentation, implement RAG on it with local embeddings, add MCP server for project context, and a `/help` command.

## Research

Consilium of two agents (Go architect + Frontend expert) analyzed the codebase:
- 27 Go files, 23 HTTP endpoints, existing RAG pipeline with Ollama embeddings
- 7 Pinia stores, 20+ Vue components, SSE streaming, MCP integration
- MCP server template: mcp-go + stdio, hackernews as minimal example
- No slash commands in web UI, interception point: `handleChat()` in server.go

## Plan

1. Generate 5 documentation files in `docs/`
2. Auto-index docs at server startup via existing RAG pipeline
3. Create MCP server `devtools` for project context
4. Add `/help` command detection in backend
5. Add `/help` visual hint in frontend

## Implementation

### New files
- `docs/api-reference.md` — all HTTP endpoints, request/response schemas, SSE events
- `docs/data-schemas.md` — all Go structs (message, chatRequest, Agent, TaskState, DocumentMeta, etc.)
- `docs/architecture.md` — high-level overview: backend, frontend, MCP, RAG, agent system
- `docs/mcp-servers.md` — all 4 MCP servers (hackernews, scheduler, pipeline, devtools) + creation template
- `docs/frontend-guide.md` — stores, components, SSE, views, shadcn-vue
- `mcp-servers/devtools/main.go` — 5 tools: dev_git_branch, dev_git_status, dev_git_log, dev_list_files, dev_read_file

### Modified files
- `backend/docs.go` — `autoIndexProjectDocs()`, `FindByOriginalName()`, `AllReadyDocIDs()`
- `backend/server.go` — `/help` command detection, auto-index call at startup
- `frontend/src/components/ChatInput.vue` — `isHelpCommand` computed, Badge, dynamic placeholder
- `dev.sh` — devtools build step
- `.mcp_servers.json` / `.mcp_servers.example.json` — devtools entry
- `TASKS.md`, `CLAUDE.md` — updated

## Validation

| Check | Result |
|-------|--------|
| Go build | OK |
| TypeScript (vue-tsc) | OK |
| MCP devtools binary | OK |
| Auto-index 5 docs at startup | OK (all "ready") |
| MCP devtools connected | OK (5 tools) |
| dev_git_branch | "master" |
| dev_list_files docs/ | 5 files |
| /help RAG pipeline | All 5 steps pass |
| /help answer with citations | OK ([N] refs + sources block) |

## Status: Done
