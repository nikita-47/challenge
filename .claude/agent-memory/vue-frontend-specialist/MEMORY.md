# Vue Frontend Specialist Memory

## Project Structure
- Types: `frontend/src/lib/types.ts` — all shared TS interfaces/types
- Stores: `frontend/src/stores/chat.ts` — Pinia setup store, SSE stream handler
- Components: `frontend/src/components/` — Vue SFCs
- UI primitives: `frontend/src/components/ui/` — shadcn-vue (Badge, Collapsible, Button, Input, Select, Slider, Tooltip, etc.)
- Composables: `frontend/src/composables/` — useSSE, etc.
- Utilities: `frontend/src/lib/` — ragCitations.ts, types.ts

## RAG Architecture (Day 24+)
- SSE events: `rag_step` (pipeline steps), `rag_context` (results + no_context flag), `rag_sources` parsed from hidden HTML comment in message content
- `parseRAGSources()` in `ragCitations.ts` strips `<!-- sources\n[...]\n-->` block from content
- `renderCitations()` in `ragCitations.ts` replaces `[N]` in rendered HTML with `.rag-cite` spans (skips `<a>`, `<code>`, `<pre>`)
- `RAGContextEvent` has `no_context?: boolean` — set when all chunks filtered or none found
- `ChatMessage` fields: `ragContext`, `ragSteps`, `ragAllResults`, `ragRejected`, `ragRewrittenQuery`, `ragThreshold`, `ragNoContext`, `ragSources`

## Component Patterns
- RAG components use `Collapsible` from `@/components/ui/collapsible` with `[-]/[+]` toggle text
- Border style: `border border-border/50 bg-background/50 text-xs rounded-sm`
- Trigger: `hover:bg-muted/30 transition-colors`
- Score colors: emerald>=70%, amber>=40%, zinc<40%
- Anti-hallucination indicator: amber-500/30 border, amber-500/5 bg, amber-400 text

## Chat Store SSE Pattern
- `case 'done'` block: parse RAG sources AFTER stream ends (content is complete)
- Guard: only parse if `msg.ragContext || msg.ragNoContext` (RAG was active)
- `case 'rag_context'`: set all ragXxx fields including `ragNoContext`

## Key Conventions
- Always `<script setup lang="ts">` — no Options API
- No single-line returns (always use braces)
- `computed()` for derived state, `ref()` for primitives
- shadcn-vue components preferred over raw HTML
- `:deep()` in `<style scoped>` for styling v-html content
- No `any` types — use proper interfaces

## Files Changed in Day 24 RAG Citations Task
- `frontend/src/lib/types.ts` — added `RAGSource`, extended `ChatMessage`, `RAGContextEvent`
- `frontend/src/lib/ragCitations.ts` — NEW: parseRAGSources, renderCitations utilities
- `frontend/src/stores/chat.ts` — done case parses sources, rag_context sets ragNoContext
- `frontend/src/components/RAGSourcesBlock.vue` — NEW: collapsible sources list with badges
- `frontend/src/components/MessageBubble.vue` — citation rendering, anti-hallucination banner, RAGSourcesBlock
