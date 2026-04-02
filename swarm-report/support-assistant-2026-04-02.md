# Day 33: Support Assistant Widget

**Date**: 2026-04-02
**Status**: Done

## Description

Mini-service for user support — floating chat widget (bottom-left circle), RAG-powered FAQ answers, MCP ticket server for issue tracking.

## Research Summary

- Go backend: reuse SSE streaming, RAG pipeline, MCP server pattern (scheduler)
- Frontend: floating widget approach (not fullscreen view), SupportWidget in App.vue outside v-if chain
- MCP tickets: CRUD with JSON persistence, 5 tools

## Plan

1. FAQ docs (docs/support/*.md) — auto-indexed at startup
2. MCP tickets server (mcp-servers/tickets/) — create, list, get, close, add_message
3. Backend support endpoint (POST /api/support/chat) — RAG + ticket context + Haiku streaming
4. Frontend SupportWidget.vue — floating overlay with SSE chat

## Implementation

### New files
- `docs/support/faq-general.md` — general FAQ (sessions, models, providers, strategies)
- `docs/support/faq-features.md` — features guide (RAG, MCP, pipeline, review, memory, tasks)
- `docs/support/faq-troubleshooting.md` — troubleshooting (common errors, fixes)
- `mcp-servers/tickets/main.go` — MCP server with 5 tools (ticket_create, ticket_list, ticket_get, ticket_close, ticket_add_message)
- `mcp-servers/tickets/store.go` — Thread-safe store with JSON persistence, debounced save
- `backend/support.go` — handleSupportChat: FAQ doc filtering, ticket MCP context, RAG search, Claude Haiku streaming
- `frontend/src/stores/support.ts` — Pinia store with SSE streaming, message history, abort
- `frontend/src/components/SupportWidget.vue` — floating widget (circle button + chat panel overlay)

### Modified files
- `backend/server.go` — registered POST /api/support/chat
- `backend/docs.go` — autoIndexProjectDocs extended for docs/support/*.md
- `frontend/src/App.vue` — added SupportWidget outside v-if chain
- `frontend/src/lib/types.ts` — added SupportMessage interface
- `dev.sh` — added tickets MCP build
- `.mcp_servers.example.json` — added tickets entry

## Validation

All 9 E2E steps passed:
1. Floating "?" button visible at bottom-left
2. Chat panel opens with "Support" header
3. Empty state: "How can I help you?"
4. User message appears as bubble (right side)
5. AI response streams with FAQ citations [1]-[5]
6. Minimize button closes panel
7. Messages preserved on reopen
8. Widget visible on docs view
9. Widget functional on docs view

API validation: SSE streaming with rag_step events, 5 FAQ chunks retrieved (cosine similarity 0.59-0.63).

## Issues and Rollbacks

None.
