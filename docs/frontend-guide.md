# Frontend Guide

Vue 3 SPA with Composition API, TypeScript, Tailwind CSS v4, Pinia, shadcn-vue.

## Project Structure

```
frontend/src/
  App.vue                    # Root layout, view switching
  components/
    ChatWindow.vue           # Message list with ScrollArea
    ChatInput.vue            # Input textarea + send settings
    MessageBubble.vue        # Message rendering (markdown, tools, RAG)
    ToolCallCard.vue         # Collapsible tool call display
    TokenBar.vue             # Footer: tokens, cost, session name
    SessionPanel.vue         # Left sidebar: sessions/memory/mcp tabs
    ChatInfoPanel.vue        # Right sidebar: session info, raw JSON
    MCPPanel.vue             # MCP server management tab
    SendSettingsPopover.vue  # Task mode, tools, MCP, RAG settings
    GlobalSettings.vue       # Provider selector (claude/local/railway)
    NewChatDialog.vue        # New chat creation dialog
    MemoryEditorDialog.vue   # Memory file editor
    BranchSelector.vue       # Branch switching for branch strategy
    TaskStatePanel.vue       # Task phase/step progress
    PipelineView.vue         # Full-screen pipeline view
    DocsView.vue             # Full-screen document management
    RAGPipelineSteps.vue     # RAG step progress in messages
    RAGChunkSplitView.vue    # Before/after filter split view
    RAGSourcesBlock.vue      # Citation sources block
    ui/                      # shadcn-vue components
  stores/
    chat.ts                  # Main chat store
    ui.ts                    # UI state (sidebar, views, provider)
    sessions.ts              # Session CRUD
    memory.ts                # Memory layer CRUD
    mcp.ts                   # MCP server/tools management
    pipeline.ts              # Pipeline runs via MCP
    docs.ts                  # Document management
  composables/
    useSSE.ts                # SSE streaming via fetch
  lib/
    types.ts                 # TypeScript types
    api.ts                   # HTTP API wrappers
    ragCitations.ts          # Citation parsing and rendering
    utils.ts                 # Utilities (cn())
```

## Stores

### useChatStore (chat.ts)

Main store for chat functionality:
- `messages`: reactive message array
- `isStreaming`: streaming state
- `currentSession`: active session name
- `usage`, `totalUsage`, `exchanges`: token tracking
- `taskState`: current task state
- `activeMcpTools`: selected MCP tools
- `activeRagDocIds`: selected RAG documents
- `ragTopK`, `ragThreshold`, `ragQueryRewrite`, `ragStrategy`: RAG settings

Key methods:
- `sendMessage(text)`: Send message with streaming
- `startTask(goal, tools, invariants)`: Start task mode
- `continueTask()`: Continue paused task
- `stopStreaming()`: Cancel current stream

### useUIStore (ui.ts)

- `activeView`: `'chat' | 'pipeline' | 'docs'`
- `sidebarOpen`: sidebar visibility
- Provider settings management

### useDocsStore (docs.ts)

- `documents`: document list
- `activeDoc`: selected document
- `chunkIndex`: chunk data for active doc
- `startPolling(id)`: polls indexing status every 2s

## SSE Event Handling

`streamRequest(url, body, callback, signal)` in `useSSE.ts`:
1. POST fetch with JSON body
2. Read response.body via ReadableStream
3. Parse `data: {...}\n` SSE format
4. Each event dispatched to callback

Chat store handles 17 event types (see api-reference.md).

## Views

Three full-screen views controlled by `activeView`:
- **chat**: ChatWindow + ChatInput + sidebars
- **pipeline**: PipelineView (HN pipeline execution)
- **docs**: DocsView (document upload, 4-strategy chunk comparison)

## shadcn-vue Components

Always use shadcn-vue instead of custom HTML:
Badge, Button, Input, ScrollArea, Collapsible, Select, Dialog, Tooltip, Slider, Checkbox, Separator, Card, Popover.

Located in `frontend/src/components/ui/`.

## RAG UI

- **RAGPipelineSteps**: Shows 5-step progress (rewrite/embed/search/filter/inject)
- **RAGChunkSplitView**: Two-column before/after threshold filter
- **RAGSourcesBlock**: Collapsible sources with ref badges
- **ragCitations.ts**: Parses `<!-- sources [...] -->` and replaces `[N]` with styled badges

## Styling

- Tailwind CSS v4
- Dark theme with green accent (`hsl(150 60% 45%)`)
- Monospace font throughout
- Number inputs: always hide spinners (`appearance-none`)
