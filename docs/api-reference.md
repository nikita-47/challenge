# API Reference

Backend HTTP API. Base URL: `http://localhost:8080`

## Chat

### POST /api/chat

Main chat endpoint. Streams SSE events.

**Request body:**

```json
{
  "message": "string",
  "session": "string",
  "model": "string",
  "system": "string",
  "maxTokens": 4096,
  "temperature": 0.7,
  "strategy": "summary|window|facts|branch",
  "windowSize": 20,
  "profile": "string",
  "project": "string",
  "operator": "string",
  "taskMode": false,
  "enabledTools": ["run_shell", "read_file"],
  "invariants": ["string"],
  "mcpTools": ["server__toolname"],
  "ragDocIds": ["doc_id"],
  "ragStrategy": "auto|semantic|sentence|structure|size",
  "ragTopK": 5,
  "ragThreshold": 0.3,
  "ragQueryRewrite": false
}
```

**SSE event types:**

| Type | Fields | Description |
|------|--------|-------------|
| `text_delta` | `text` | Incremental text token |
| `text` | `text` | Agent goal echo |
| `api_request` | `text` | Full API request JSON |
| `turn` | `turn`, `max_turn` | Agent turn counter |
| `thinking` | `text` | Agent thinking/reasoning |
| `tool_call` | `tool`, `input` | Tool invocation |
| `tool_result` | `tool`, `output`, `is_error` | Tool execution result |
| `usage` | `usage: {input, output, cache_creation_input, cache_read_input}` | Token usage per call |
| `done` | `turn`, `stats` | Stream complete |
| `error` | `message` or `text` | Error occurred |
| `facts_updated` | `facts` | Facts strategy updated facts |
| `step_result` | `text` (JSON StepResult) | Task step completed |
| `task_state` | `text` (JSON TaskState) | Task state updated |
| `rag_step` | `step`, `status`, `detail` | RAG pipeline step progress |
| `rag_context` | `results`, `all_results`, `rejected`, `rewritten_query`, `threshold`, `no_context` | RAG search results |
| `memory_updated` | - | Memory auto-update completed |
| `compress` | `messageCount`, `summaryLen`, `tokensSaved` | Context compression event |

---

## Sessions

### GET /api/sessions

List all saved sessions.

**Response:** `[{ "name": "string", "profile": "string", "project": "string", "operator": "string" }]`

### GET /api/sessions/:name

Load session with messages, settings, stats, branches, task state.

**Response:**
```json
{
  "messages": [{ "role": "user|assistant|system", "content": "string" }],
  "summary": "string",
  "settings": { "model": "", "strategy": "", "windowSize": 0, "profile": "", "project": "", "operator": "" },
  "stats": { "total_input": 0, "total_output": 0, "exchanges": 0, "tokens_saved": 0 },
  "facts": { "key": "value" },
  "branches": [{ "name": "", "forkIndex": 0, "messageCount": 0, "createdAt": "" }],
  "activeBranch": "main",
  "taskState": null
}
```

### GET /api/sessions/:name/raw

Raw session JSON file.

### PUT /api/sessions/:name

Rename session. Body: `{ "newName": "string" }`

### DELETE /api/sessions/:name

Delete session.

---

## Branches

### GET /api/sessions/:name/branches

List branches for a session.

### POST /api/sessions/:name/branches

Create branch. Body: `{ "name": "string" }`

### DELETE /api/sessions/:name/branches

Delete branch. Body: `{ "name": "string" }`

### PUT /api/sessions/:name/branch

Switch active branch. Body: `{ "name": "string" }`

---

## Memory

Three memory layers: profiles, projects, operators.

### GET /api/memory/{layer}

List files in layer. Layer: `profiles`, `projects`, `operators`.

### POST /api/memory/{layer}

Create file. Body: `{ "name": "string", "content": "string" }`

### GET /api/memory/{layer}/:name

Get file content. Response: `{ "name": "", "content": "" }`

### PUT /api/memory/{layer}/:name

Update file. Body: `{ "content": "string" }`

### DELETE /api/memory/{layer}/:name

Delete file.

---

## Settings

### GET /api/settings

Get provider settings. Response: `{ "provider": "claude|local|railway", "localURL": "", "localModel": "", "localKey": "" }`

### POST /api/settings

Update provider. Body: `{ "provider": "claude|local|railway", "localURL": "", "localModel": "", "localKey": "" }`

### GET /api/config

Get full config. Response includes model, maxTokens, temperature, system, provider, pricing.

---

## MCP (Model Context Protocol)

### GET /api/mcp/servers

List all configured MCP servers with status.

**Response:** `[{ "name": "", "connected": true, "toolsCount": 0, "error": "" }]`

### POST /api/mcp/servers/:name/connect

Connect to a specific MCP server.

### POST /api/mcp/servers/:name/disconnect

Disconnect from a specific MCP server.

### GET /api/mcp/tools

List tools from connected servers. Optional `?server=name` query param.

**Response:** `[{ "server": "", "name": "", "description": "", "inputSchema": {} }]`

### POST /api/mcp/tools/call

Call an MCP tool directly.

**Body:** `{ "server": "string", "tool": "string", "arguments": {} }`

### POST /api/mcp/reload

Reload MCP config and reconnect all servers.

---

## Documents (RAG)

### GET /api/docs

List all documents. Response: array of DocumentMeta.

### POST /api/docs/upload

Upload and index a document. Multipart form:
- `file`: .txt, .md, or .pdf file
- `chunk_size`: int (100-5000, default 1000)
- `overlap`: int (0-chunk_size/2, default 200)

### GET /api/docs/:id

Get document metadata.

### DELETE /api/docs/:id

Delete document and its index.

### GET /api/docs/:id/chunks

Get all chunks with similarity stats across 4 strategies.

### POST /api/docs/:id/search

Search document chunks. Body: `{ "query": "string", "top_k": 5, "strategy": "auto" }`
