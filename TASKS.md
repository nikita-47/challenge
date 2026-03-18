# Daily Tasks

Each day's assignment, what was built, and key notes.

---

## Day 1 — First LLM API request ✅

**Assignment:** Write minimal code that sends a request to an LLM via API, receives a response, and prints it to the console or simple interface (CLI/Web).

**What was built:**
- `main.go`: single-file Go CLI that calls the Anthropic Messages API
- Streaming response via SSE (`stream: true`)
- `.env` file loading for `ANTHROPIC_API_KEY`
- Basic REPL loop: read user input → send to API → print response

**Key code:** `streamChat()`, `readStream()`, `loadEnv()`

---

## Day 2 — Response format control ✅

**Assignment:** Send the same request with and without output controls:
- Explicit format description
- Response length limit (`max_tokens`)
- Stop sequence

Compare responses: without constraints vs. with constraints.

**What was built on top of Day 1:**
- `--max-tokens` flag (limits response length)
- `--format` flag (appends `"Always respond in this format: <value>"` to system prompt)
- `--stop` flag (sends `stop_sequences` array in API request + instructs model to end with stop string)
- `--system` flag (custom system prompt)
- `/system <text>` chat command (update system prompt mid-session)
- `/help` and `/clear` commands

**Key code:** `parseArgs()`, `buildSystemPrompt()`, `buildRequest()`

---

## Day 3 — Different reasoning approaches ✅

**Assignment:** Take one logical/algorithmic/analytical problem and solve it four ways via API:
1. Direct answer — no extra instructions
2. Step-by-step — prompt: "solve step by step"
3. Meta-prompting — ask the model to first write a prompt for solving the task, then use that prompt
4. Expert panel — create a group of experts in the prompt (e.g. analyst, engineer, critic), get a solution from each

Compare: do the answers differ? Which approach gave the most accurate result?

**Problem chosen:** Birthday paradox — how many people for >50% chance two share a birthday? (Answer: 23)

**What was built on top of Day 2:**
- `/compare <question>` command — streams all 4 approaches simultaneously in a split-screen TUI
- `splitScreen` + `panel` types for ANSI terminal layout (4 quadrants + status bar)
- `streamToPanel` / `readStreamToPanel` — SSE streaming directly into a panel region
- `runComparison` orchestrator — 4 goroutines in parallel, `context.WithCancel` for cancellation
- Ctrl+C → cancels all 4 HTTP streams mid-flight via context propagation
- Progress counter in status bar (X/4 готово), input blocked during comparison
- New imports: `context`, `os/signal`, `sync`, `syscall`, `unsafe`

**Key code:** `runComparison()`, `newSplitScreen()`, `streamToPanel()`, `splitScreen.write()`

---

## Day 4 — Temperature ✅

**Assignment:** Send the same request with temperature 0, 0.7, and 1.0. Compare accuracy, creativity, and diversity. Determine which temperature suits which tasks.

**What was built on top of Day 3:**
- `--temperature` flag (sets sampling temperature 0.0–1.0 in API request)
- `--tempcompare` flag (run 3-way temperature comparison directly and exit)
- `/temp <question>` chat command — streams the same question at temp 0, 0.7, and 1.0 in a 3-column split-screen TUI
- `newTempScreen()` — 3-column panel layout (reuses `splitScreen` infrastructure)
- `drawTempBorders()` / `redrawTemp()` — rendering for the 3-column variant
- `runTempComparison()` — orchestrator: 3 goroutines, one per temperature, with Ctrl+C cancellation
- `panelCount` field on `splitScreen` — generalizes done counter for 3 or 4 panels
- `--verbose` works in panel modes: curl output rendered inside each panel (not to stderr)
- `formatCurl()` — returns curl command as string for in-panel rendering

**Key code:** `runTempComparison()`, `newTempScreen()`, `drawTempBorders()`

**Observations:**
- Anthropic API max temperature is 1.0 (not 1.5 as some docs suggest)
- Temperature difference is most visible on creative tasks (stories, metaphors, naming)
- Factual/analytical questions produce nearly identical output at any temperature

---

## Day 5 — Model version comparison ✅

**Assignment:** Send the same query to weak/medium/strong models, measure response time, token count, cost, and compare quality.

**Models used:**
- Weak: Qwen2.5-Coder-14B (LM Studio, localhost:1234, free)
- Medium: GPT-4o-mini (OpenAI API, $0.15/$0.60 per 1M tokens)
- Strong: Claude Sonnet 4.5 (Anthropic API, $3.00/$15.00 per 1M tokens)

**What was built on top of Day 4:**
- `OPENAI_API_KEY` loading from `.env`
- `modelInfo` struct — describes a model (name, provider, baseURL, apiKey, model, cost)
- `buildOpenAIRequest()` — builds OpenAI-compatible chat completions request body
- `/models <question>` command — streams the same question to all 3 models in a 3-column split-screen TUI
- `--models` flag — run model comparison directly from CLI
- `metrics` struct — tracks duration, input/output tokens, cost per model
- `newModelScreen()` / `drawModelBorders()` / `redrawModel()` — 3-column TUI layout for model comparison
- `streamToPanelOpenAI()` — streams OpenAI-compatible API (OpenAI + LM Studio) to panel, extracts token usage
- `streamToPanelAnthropic()` — streams Anthropic API to panel, extracts input/output tokens from SSE events
- `runModelComparison()` — orchestrator: 3 goroutines, one per model, with Ctrl+C cancellation
- `printComparisonTable()` — renders ASCII table with time, tokens, cost, provider after streaming completes
- Token extraction: Anthropic SSE (`message_start`/`message_delta`), OpenAI SSE (`stream_options.include_usage`), LM Studio fallback (~4 chars per token)

**Key code:** `runModelComparison()`, `streamToPanelOpenAI()`, `streamToPanelAnthropic()`, `printComparisonTable()`

---

## Day 6 — First Agent with Tools ✅

**Assignment:** Build an agentic loop: LLM decides which tool to call, executes it, gets the result, and repeats until the goal is achieved or max turns reached.

**What was built on top of Day 5:**
- `agent.go` — new file (~170 lines) with `Agent` struct and agentic loop
- `agentMessage`, `contentBlock`, `toolDef`, `apiResponse` types for non-streaming tool_use API
- Two tools: `run_shell` (executes `/bin/sh -c` commands) and `read_file` (reads file contents)
- `executeTool()` — dispatcher that routes tool calls to shell or file read
- `(*Agent).callAPI()` — non-streaming POST to Anthropic API with tool definitions
- `(*Agent).Run(goal)` — agentic loop: sends goal → processes tool_use blocks → returns final text
- Max 10 turns, colored terminal output: goal (bold), turns (dim), tool calls (yellow), results (dim)
- `--agent` flag — run agent from CLI and exit
- `/agent <task>` chat command — run agent from interactive chat
- `truncate()` helper for single-line display of long results

**Key code:** `newAgent()`, `Agent.Run()`, `executeTool()`, `defaultTools()`

---

## Day 7 — Chat History Persistence ✅

**Assignment:** Save chat history to JSON and restore on startup. Named sessions via `/save` and `/load`. `--session` flag for CLI. `/clear` deletes the session file.

**What was built on top of Day 6:**
- `history.go` — new file (~60 lines) with session persistence
- `sessionFile` struct — wraps `saved_at` timestamp and `messages` array
- `sessionPath(name)` — resolves session name to `.chat_history/<name>.json` (empty → `"default"`)
- `saveSession()` / `loadSession()` / `deleteSession()` — JSON file operations with graceful error handling
- Auto-save after every message exchange; auto-load on startup with "Resumed session" message
- `/save [name]` command — save current history to named session
- `/load <name>` command — load named session and switch auto-save target
- `/clear` — now also deletes the session file for a clean restart
- `--session <name>` flag — start with a specific named session
- Banner shows session name when `--session` is used
- `.chat_history/` added to `.gitignore`

**Key code:** `saveSession()`, `loadSession()`, `deleteSession()`, `sessionPath()`

---

## Day 8 — Token Counting + Agent-Chat Integration ✅

**Assignment:** Count tokens for each request, full history, and model response. Show cost growth. Integrate agent with chat history so `/agent` sees chat context and results go into history.

**What was built on top of Day 7:**
- `tokens.go` — new file with `tokenUsage` (per-request) and `tokenStats` (cumulative session) structs
- `tokenStats.Add()`, `TotalTokens()`, `TotalCost()`, `FormatTotal()`, `formatTokenUsage()` — tracking and display
- Cost model: Claude Sonnet 4.5 — $3/$15 per 1M input/output tokens
- `message.Content` changed from `string` to `any` — supports both plain text and `[]contentBlock` (tool_use)
- `UnmarshalJSON` on `message` — backward-compatible loading of old string-based sessions and new block-based content
- `messageText()` helper — extracts plain text from `Content` regardless of type (string or []contentBlock)
- `readStream()` now returns `tokenUsage` — parses `message_start` (input_tokens) and `message_delta` (output_tokens) from SSE
- `streamChat()` returns `(string, tokenUsage, error)` — propagates token counts to caller
- `agentMessage` type removed — agent uses unified `message` type everywhere
- `apiResponse.Usage` — parsed from non-streaming agent API calls
- `Agent.Stats` — cumulative `tokenStats` across all agent turns, printed per-turn and at end
- `Agent.Run(goal, chatHistory)` — accepts chat history for context; agent sees previous conversation
- `/agent` handler in chat: passes `history`, adds flattened task+result to history, merges agent stats
- `/tokens` command — displays current session token statistics
- `/clear` resets token stats
- Banner shows cost model ($3/$15 per 1M tokens)
- `buildOpenAIRequest()` uses `messageText()` for Content extraction (supports any type)

**Key code:** `tokenUsage`, `tokenStats`, `messageText()`, `message.UnmarshalJSON()`, updated `readStream()`, `Agent.Run(goal, chatHistory)`

---

## Day 9 — Context Compression ✅

**Assignment:** Implement context management: keep last N messages as-is, replace older ones with summary (every 10 messages), store summaries separately and inject instead of full history. Compare token usage before/after.

**What was built on top of Day 8:**
- `compress.go` — new file (~130 lines) with rolling context compression
- `contextWindow` struct — tracks `Summary` (single string) and `Messages` (current unsummarized, ≤10)
- Rolling compression algorithm:
  1. Accumulate messages until 10
  2. Compress (previous summary + 10 messages) → one new summary via API call
  3. Clear messages, start accumulating again
  4. Repeat — always one summary, max 10 unsummarized messages
- `maybeCompress(apiKey, cw, stats)` — triggers compression at threshold, mutates cw
  - `buildCompressedMessages(cw)` — returns API-ready message slice (summary + ack + messages)
- `summarize(apiKey, prevSummary, msgs)` — non-streaming API call; merges previous summary with new messages
- `sessionFile` extended with `Summary` field (backward-compatible via `omitempty`)
- `saveSessionCW()` / `loadSessionCW()` — persist/load `contextWindow` state
- `tokenStats.TokensSaved` — tracks estimated input tokens saved by compression
- `/compress` command — shows current summary text
- Compression output: `[compressed N messages → summary (M chars) | saved ~K tokens]`

**Bugfix:** `compressHistory()` split into `maybeCompress()` + `buildCompressedMessages()`. Compression now runs BEFORE appending the new user message, so the current question is never swallowed into the summary.

**Key code:** `contextWindow`, `maybeCompress()`, `buildCompressedMessages()`, `summarize()`, `saveSessionCW()`, `loadSessionCW()`

---

## Day 10 — Context Management Strategies ✅

**Assignment:** Implement 3 alternative context management strategies beyond simple compression: Sliding Window, Sticky Facts (key-value memory), and Branching. Strategy selected at chat creation, configurable window size N.

**What was built on top of Day 9:**
- `strategy.go` — central dispatcher: `buildAPIMessages()`, `maybeProcess()`, `activeMessages()`, `appendMessage()`, `getStrategy()`, `getWindowSize()`
- `strategy_window.go` — Sliding Window: `buildWindowMessages()` keeps last N messages only
- `strategy_facts.go` — Sticky Facts: `buildFactsMessages()` prepends facts as user+ack pair; `maybeExtractFacts()` calls API after each exchange to extract key-value facts; `extractFacts()` non-streaming API call
- `strategy_branch.go` — Branching: `branch` struct, `createBranch()`, `switchBranch()`, `listBranches()`, `deleteBranch()`, `buildBranchMessages()`
- `BranchSelector.vue` — toolbar component for branch switching and creation (dropdown + new branch input)
- `NewChatDialog.vue` — added strategy selector (Summary/Window/Facts/Branch) and conditional windowSize input
- `ChatInfoPanel.vue` — shows strategy type, window size, facts list, branch info; added Info/Raw tab switcher to view raw session JSON
- `ChatWindow.vue` — handles `branch_fork` system event as visual fork marker
- `history.go` — extended `sessionFile`/`sessionSettings` with Strategy, WindowSize, Facts, Branches, ActiveBranch
- `server.go` — branch REST endpoints (POST/GET `/branches`, PUT `/branch`, GET `/raw`); facts_updated SSE event; strategy dispatcher integration
- `chat.go` — CLI commands `/strategy`, `/facts`, `/branch`, `/switch`, `/branches`; strategy dispatcher integration
- `main.go` — `--strategy` and `--window-size` CLI flags
- `types.ts` — `ContextStrategy`, `BranchInfo`, `FactsUpdatedEvent` types
- `api.ts` — `createBranchAPI()`, `switchBranchAPI()`, `fetchBranchesAPI()`, `fetchSessionRaw()`
- `chat.ts` / `sessions.ts` — facts, branches, activeBranch state management

**Key code:** `buildAPIMessages()`, `maybeProcess()`, `activeMessages()`, `appendMessage()`, `buildWindowMessages()`, `buildFactsMessages()`, `extractFacts()`, `createBranch()`, `switchBranch()`, `handleGetSessionRaw()`

---

## Day 11 — Local LLM for UI Testing ✅

**Assignment:** Add LM Studio integration to `dev.sh` for fast, free UI testing with a local LLM instead of Claude API. Add `start-test`/`stop-test` commands that spin up LM Studio + Go (with `--base-url`) + Vite in one step.

**What was built on top of Day 10:**
- `dev.sh` — new commands: `start-lms`, `stop-lms`, `start-test`, `stop-test`; LM Studio status in `./dev.sh status`
- LM Studio integration: auto-start server on port 1234, auto-load `qwen2.5-0.5b-instruct-mlx` model
- `start-test` launches Go server with `--base-url http://localhost:1234 --model qwen2.5-0.5b-instruct-mlx`
- `CLAUDE.md` — added "Browser testing mode (local LLM)" section with `start-test`/`stop-test` docs
- `.claude/settings.json` — added `lms` CLI permission

**Bugfix:** `lms server status` writes to stderr, not stdout — fixed `2>/dev/null` → `2>&1` for grep. Also `"not running"` matched `grep "running"` — narrowed to `grep "is running"`.

**Key code:** `start_lms()`, `stop_lms()`, `start_go_test()` in `dev.sh`

---

## Day 12 — Memory Model (Memory Layers) ✅

**Assignment:** Implement a 3-layer memory model for the assistant: short-term (chat messages, already exists), working memory (project .md files shared across chats), and long-term memory (profile .md files selected at chat creation). Storage on disk, API, selection at chat creation, minimal CRUD from UI.

**What was built on top of Day 11:**
- `memory.go` — CRUD functions for .md files in `.memory/profiles/` and `.memory/projects/` with path validation
- `history.go` — extended `sessionSettings` with `Profile` and `Project` fields
- `api.go` — `buildFullSystemPrompt()` assembles system prompt from memory layers (profile → project → user system prompt)
- `server.go` — extended `chatRequest` with profile/project; injected memory into handleChat; added REST endpoints for profiles/projects CRUD
- `frontend/src/lib/types.ts` — `MemoryFile` type, extended `ChatSettings` with profile/project
- `frontend/src/lib/api.ts` — memory API functions (fetchProfiles, fetchProfile, createProfile, updateProfile, deleteProfileAPI + same for projects)
- `frontend/src/stores/memory.ts` — Pinia store for profiles/projects lists
- `frontend/src/components/NewChatDialog.vue` — profile/project selects in new chat dialog
- `frontend/src/components/MemoryEditorDialog.vue` — dialog for creating/editing .md memory files
- `frontend/src/components/ChatInfoPanel.vue` — memory section in info tab + new "mem" tab with CRUD for profiles/projects
- `frontend/src/stores/sessions.ts` — profile/project mapping on session load
- `.gitignore` — added `.memory/`

**Key code:** `buildFullSystemPrompt()`, `handleMemoryList()`, `handleMemoryItem()`, `useMemoryStore`, `MemoryEditorDialog.vue`

---

## Day 13 — Assistant Personalization ✅

**Assignment:** Add operator memory layer (immutable user identity, first in system prompt), auto-update profile/project memory via LLM after each exchange, full operator CRUD in UI.

**What was built on top of Day 12:**
- `memory_update.go` — new file: `maybeUpdateMemory()` analyzes last user+assistant exchange, calls Haiku to decide if profile/project memory needs updating; `analyzeMemoryUpdate()` non-streaming API call with memory-manager system prompt
- `memory.go` — added operator CRUD helpers: `memoryOperatorsDir()`, `listOperators()`, `getOperator()`, `saveOperator()`, `deleteOperator()`
- `api.go` — operator injected FIRST in `buildFullSystemPrompt()` (before profile)
- `history.go` — `Operator` field in `sessionSettings` and `sessionInfo`, exposed in `listSessions()`
- `server.go` — `/api/memory/operators` CRUD endpoints, `Operator` in `chatRequest`, `maybeUpdateMemory()` call after each exchange with `memory_updated` SSE event
- `frontend/src/lib/api.ts` — full operator API: `fetchOperators`, `fetchOperator`, `createOperator`, `updateOperator`, `deleteOperatorAPI`
- `frontend/src/stores/memory.ts` — `operators` state, `loadOperators()`, `addOperator()`, `removeOperator()`
- `frontend/src/components/NewChatDialog.vue` — operator select dropdown (before profile)
- `frontend/src/components/SessionPanel.vue` — operator CRUD in memory tab, operator badge in session list
- `frontend/src/components/ChatInfoPanel.vue` — operator display in memory section
- `frontend/src/components/MemoryEditorDialog.vue` — extended `kind` type with `'operator'`
- `.claude/agents/*.md` — added `description` field to all sub-agents

**Key code:** `maybeUpdateMemory()`, `analyzeMemoryUpdate()`, `memoryOperatorsDir()`, operator CRUD, `memory_updated` SSE event

---

## Day 14 — Task State Machine ✅

**Assignment:** Implement a finite state machine for agent tasks — agent plans steps, executes them sequentially, validates results. State is persistent, can be paused and resumed.

**What was built on top of previous day:**
- `backend/taskstate.go` — `TaskState`, `TaskStep`, `StepResult` types, deterministic phase transitions (planning → executing → validating → done), phase-specific system prompts (`buildPlanningPrompt()`, `buildExecutingPrompt()`, `buildValidatingPrompt()`), local LLM text-mode prompts and parsers (`parsePlanText()`, `parseExecutionText()`, `parseValidationText()`), `FormatStatus()` for CLI display
- `backend/agent.go` — `PhaseResult`, `StepResults` fields on `Agent`, phase-specific constructors (`newPlanningAgent()`, `newExecutingAgent()`, `newValidatingAgent()`), 3 phase tools (`submitPlanTool()`, `reportStepTool()`, `submitValidationTool()`), `submit_plan`/`submit_validation` handling in `Run()` loop with `phaseComplete` early exit, `report_step` accumulation
- `backend/server.go` — `runTaskPhase()` orchestrator (switch by phase, agent creation, result processing, transitions), `runTaskPhaseLocal()` for local LLM (text-only, no tools), `providerSettings` struct with global provider state, `/api/settings` GET/POST endpoints, task mode integration in `handleChat()` with provider-aware routing, `SendSettings` popover support via `enabledTools` in `chatRequest`
- `backend/chat.go` — `/task <goal>` (start task mode), `/task` (show status), `/resume` (resume paused task), `step_result` event display in `cliAgentEmit`
- `backend/history.go` — `TaskState` field in `sessionFile`, persisted in `saveSessionCW()`/`loadSessionCW()`
- `backend/compress.go` — `TaskState` field in `contextWindow`
- `frontend/src/lib/types.ts` — `TaskPhase` (without 'paused'), `TaskStep`, `StepResult`, `TaskState` (with `paused`, `artifacts`, `step_results`, `validation_count`), `StepResultEvent`, added to `SSEEvent` union
- `frontend/src/stores/chat.ts` — `taskState` ref, `task_state`/`step_result` SSE handlers, `startTask()`, `continueTask()`, `cancelTask()` methods, `isTaskContinue` logic (empty message for continue), `taskMode: true` + `enabledTools` in request body
- `frontend/src/stores/sessions.ts` — `taskState` loaded from session data
- `frontend/src/components/TaskStatePanel.vue` — phase indicators with paused state (yellow), step list with checkmarks, progress counter, Continue button (visible when paused), Cancel button, plan summary artifact display, feedback/error sections
- `frontend/src/components/ChatInput.vue` — disabled when paused, placeholder "Task paused — click Continue above", `SendSettingsPopover` hidden during active task
- `frontend/src/components/ChatInfoPanel.vue` — task info section (phase, paused, validation_count, progress, step_results, goal)
- `frontend/src/App.vue` — `TaskStatePanel` integrated between toolbar and ChatWindow

**Key code:** `runTaskPhase()`, `runTaskPhaseLocal()`, `newPlanningAgent()`, `newExecutingAgent()`, `newValidatingAgent()`, `submitPlanTool()`, `reportStepTool()`, `submitValidationTool()`, `parsePlanText()`, `parseValidationText()`, `continueTask()`, `cancelTask()`

---

## Day 15 — Token Optimization + Invariants + Prompt Caching ✅

**Assignment:** Optimize agent loop token consumption: compress old tool results, add efficiency rules to phase prompts, reduce maxTurns, add prompt caching with cache_control. Add invariants (user-defined constraints) to task mode. Add shared sandbox across task phases.

**What was built on top of previous day:**
- `backend/agent.go` — `compactHistory()` compresses old tool_result/text blocks after each turn (keeps last 4 messages intact), `compactToolResult()` summarizes to one-line `[output: ... (N lines)]`, `formatStepProgress()` injects completed steps into goal message for orientation after compaction, `buildPayload()` adds `cache_control: ephemeral` to system prompt and last tool for prompt caching, `apiResponse.Usage` extended with `CacheCreationInput`/`CacheReadInput`, tool output truncation in `executeTool()` (run_shell 4K, read_file 8K), maxTurns reduced (executing 20→12, validating 10→6)
- `backend/taskstate.go` — `Invariants` field on `TaskState`, `formatInvariantsBlock()` generates INVARIANTS section for all phase prompts, `EnsureSandbox()`/`CleanupSandbox()` for shared sandbox across phases, `buildPlanningPrompt()` "MINIMAL plan 2-4 steps", `buildExecutingPrompt()` "Efficiency Rules" block, `buildValidatingPrompt()` "Be decisive: 2-3 tool calls"
- `backend/tokens.go` — `CacheCreationInput`/`CacheReadInput` in `tokenUsage`/`tokenStats`, updated `TotalCost()` with cache pricing ($3.75/M write, $0.30/M read), cache info in `formatTokenUsage()`/`FormatTotal()`
- `backend/history.go` — `CacheCreationInput`/`CacheReadInput` in `sessionStats`, round-trip to `tokenStats`
- `backend/server.go` — shared sandbox via `ts.EnsureSandbox()`, cleanup on task done, `Invariants` in `chatRequest`, phase constructors use shared `sandboxDir`
- `backend/chat.go` — old sandbox cleanup when starting new task
- `frontend/src/components/SendSettingsPopover.vue` — invariants CRUD: add/remove rules with `!` badge, stored as string array
- `frontend/src/components/TaskStatePanel.vue` — invariants display in task panel
- `frontend/src/components/ChatInfoPanel.vue` — invariants display in info panel
- `frontend/src/components/ChatInput.vue` — passes invariants to `startTask()`
- `frontend/src/stores/chat.ts` — `startTask()` accepts invariants, sends in request body
- `frontend/src/lib/types.ts` — `invariants?: string[]` on `TaskState`

**Prod test results (bubble sort, 2 invariants):**
- Planning: 1 turn, Executing: 5 turns, Validating: 2 turns = 8 total
- 13,361 input / 1,800 output tokens, $0.067
- vs baseline: 60,167 input, $0.42 → **-78% tokens, -84% cost**

**Key code:** `compactHistory()`, `compactToolResult()`, `formatStepProgress()`, `formatInvariantsBlock()`, `EnsureSandbox()`, `CleanupSandbox()`, cache_control in `buildPayload()`

---

## Day 16 — Dynamic Task Phases ✅

**Assignment:** Replace hardcoded 4-phase pipeline (planning→executing→validating→done) with dynamic phase proposal. Agent analyzes the goal, proposes a custom pipeline of phases, user approves or sends feedback before execution begins.

**What was built on top of previous day:**
- `backend/taskstate.go` — `PhaseSpec` struct (Name/Type/Description/Status), `PhaseProposing` constant, `Phases []PhaseSpec` + `CurrentPhaseIndex` on `TaskState`, `validatePhases()` enforces constraints (planning before executing, must end with validating, 2-5 phases), `buildProposingPrompt()`/`buildProposingPromptLocal()` system prompts, `parseProposedPhasesText()` parser with fallback, updated `FormatStatus()` for pipeline display
- `backend/agent.go` — `newProposingAgent()` (tools: submit_phases, maxTurns: 3), `submitPhasesTool()` schema, `submit_phases` added to phase-ending tool handling in agentic loop
- `backend/server.go` — `PhaseProposing` case in `runTaskPhase()`/`runTaskPhaseLocal()`, `advancePhase()` helper for dynamic phase transitions, phase description context injection ("Phase focus: ..."), approval flow (empty message = approve, non-empty = feedback → re-run proposing), validation failure rewinds to appropriate phase in pipeline
- `backend/chat.go` — `/task <goal>` starts with `PhaseProposing`
- `frontend/src/lib/types.ts` — `PhaseSpec` interface, `'proposing'` added to `TaskPhase`, `phases?`/`current_phase_index?` on `TaskState`
- `frontend/src/stores/chat.ts` — `startTask()` initial phase = `'proposing'`
- `frontend/src/components/TaskStatePanel.vue` — dynamic phases from `taskState.phases`: "analyzing..." during proposing, proposed pipeline detail with approve/modify UI, phase progress with status colors, null-safe `steps`/`step_results` access
- `frontend/src/components/ChatInfoPanel.vue` — `'proposing'` color (purple), pipeline section with dynamic phases and status, null-safe `steps`/`step_results` access

**Key code:** `PhaseSpec`, `validatePhases()`, `newProposingAgent()`, `submitPhasesTool()`, `buildProposingPrompt()`, `parseProposedPhasesText()`, `advancePhase()`, approval flow in `handleChat()`

---

## Day 17 — MCP Client Integration ✅

**Assignment:** Install MCP SDK/client, write code that establishes an MCP connection and retrieves the list of available tools. First integration: Railway MCP server.

**What was built on top of previous day:**
- `backend/mcp.go` — `MCPManager` (sync.RWMutex, multi-server), `MCPConnection`, `MCPServerConfig`, `MCPServersFile` config types, `Connect()`/`Disconnect()`/`ConnectAll()`/`DisconnectAll()` lifecycle, `GetTools()`/`CallTool()` via `mcp-go` library, 5 HTTP handlers under `/api/mcp/`
- `backend/server.go` — MCPManager init at startup, 5 route registrations (`/api/mcp/servers`, `/api/mcp/servers/`, `/api/mcp/tools`, `/api/mcp/tools/call`, `/api/mcp/reload`)
- `go.mod` — first external dependency: `github.com/mark3labs/mcp-go v0.45.0` (MCP protocol, JSON-RPC 2.0, stdio/SSE/HTTP transports)
- `.mcp_servers.example.json` — example config (Claude Desktop format)
- `frontend/src/stores/mcp.ts` — Pinia store: servers, tools, loading, connect/disconnect/reload actions
- `frontend/src/components/MCPPanel.vue` — MCP panel: Select dropdown for server selection, on/off toggle, status dot, scrollable tool list with names, descriptions, expandable JSON schemas
- `frontend/src/components/SessionPanel.vue` — third tab "mcp" in left sidebar
- `frontend/src/lib/api.ts` — `fetchMCPServers()`, `fetchMCPTools()`, `connectMCPServer()`, `disconnectMCPServer()`, `reloadMCPConfig()`
- `frontend/src/lib/types.ts` — `MCPServerStatus`, `MCPToolInfo` interfaces

**Railway MCP:** 14 tools discovered (check-railway-status, list-projects, deploy, list-services, get-logs, create-environment, etc.)

**Key code:** `MCPManager`, `NewMCPManager()`, `Connect()`, `Disconnect()`, `GetTools()`, `CallTool()`, `handleMCPServers()`, `handleMCPTools()`, `handleMCPToolCall()`, `useMCPStore`, `MCPPanel.vue`

---

## Day 18 — Custom MCP Server (HackerNews) ✅

**Assignment:** Implement a custom MCP server wrapping an external API. Register tools with input parameters, return results. Connect to the agent and invoke tools from the application.

**What was built on top of previous day:**
- `mcp-servers/hackernews/main.go` — standalone MCP server binary (stdio transport via `mcp-go/server`), wraps HackerNews Firebase API + Algolia Search. 4 tools: `hn_top_stories` (top N stories), `hn_get_item` (item by ID), `hn_get_user` (profile by username), `hn_search` (Algolia full-text search with tags filter). No API keys needed.
- `backend/mcp.go` — `GetToolDefs()` converts MCP tools to agent `toolDef` format (JSON round-trip for schema), `parseMCPToolName()` splits `server__toolname` for routing
- `backend/agent.go` — `mcpMgr *MCPManager` field on Agent, `executeTool()` extended with MCP routing: parses `server__toolname`, calls `MCPManager.CallTool()` with 30s timeout, extracts TextContent
- `backend/server.go` — `McpTools []string` in `chatRequest`, `handleChat()` receives `*MCPManager`, creates agent with MCP tools when `mcpTools` is non-empty
- `frontend/src/components/SendSettingsPopover.vue` — "MCP Servers" section with grouped UI: server-level checkbox (toggle all tools), per-tool checkboxes, `N/M` counter, violet highlight on settings button when MCP tools active
- `frontend/src/stores/chat.ts` — `activeMcpTools` ref, sent as `mcpTools` in chat request
- `dev.sh` — `build_mcp_servers()` auto-builds HackerNews binary before `start-go`
- `.mcp_servers.example.json` — added `hackernews` entry

**Key code:** `mcp-servers/hackernews/main.go`, `GetToolDefs()`, `parseMCPToolName()`, `executeTool()` MCP routing, `connectedServers`, `toggleServer()`, `activeMcpTools`

---

## Day 19 — Scheduler MCP Server (Background Tasks) ✅

**Assignment:** Create an MCP tool with scheduled or periodic execution. The tool should save data (JSON), execute on schedule, and return aggregated results. An agent that works 24/7 and periodically produces summaries.

**What was built on top of previous day:**
- `mcp-servers/scheduler/store.go` — `Store` with `sync.RWMutex`, JSON persistence (`scheduler_data.json`), ring buffer results (max 50), debounced save (2s timer), CRUD methods, `crypto/rand` ID generation
- `mcp-servers/scheduler/runner.go` — `RunTask()` dispatcher, `runReminder()` (one-shot `time.Timer`), `runURLMonitor()` (periodic `time.Ticker` + HTTP GET with timing/size), `runHNDigest()` (periodic HN Firebase fetch), `fetchURL()`, `fetchHNTopStories()`
- `mcp-servers/scheduler/main.go` — MCP server with 6 tools (`sched_create`, `sched_list`, `sched_status`, `sched_delete`, `sched_pause`, `sched_resume`), startup task restoration from JSON, `rootCtx` for goroutine lifecycle, `dataFilePath()` with env override
- `dev.sh` — scheduler build in `build_mcp_servers()`
- `.mcp_servers.example.json` — scheduler entry
- `.gitignore` — scheduler binary + data file

**3 task types:** `reminder` (one-shot after delay), `url_monitor` (periodic HTTP health check with uptime%/avg response), `hn_digest` (periodic HackerNews top stories digest)

**Key code:** `Store`, `RunTask()`, `runURLMonitor()`, `runHNDigest()`, `runReminder()`, `fetchURL()`, `handleCreate()`, `handleStatus()`, `debouncedSave()`, `generateID()`

---

## Day 20 — Pipeline MCP Server + Full-Screen Layout ✅

**Assignment:** Create a pipeline MCP server (search → summarize → save), build a full-screen pipeline UI (replacing cramped sidebar tab), add run deletion, and integrate scheduler with pipeline for delayed execution from chat.

**What was built on top of previous day:**
- `mcp-servers/pipeline/main.go` — standalone MCP server (stdio), 7 tools: `pipe_search`, `pipe_summarize`, `pipe_save`, `pipe_run`, `pipe_status`, `pipe_list`, `pipe_delete`. HackerNews Algolia search → Claude Haiku summarization → file save.
- `mcp-servers/pipeline/store.go` — `Store` with `sync.RWMutex`, JSON persistence, debounced save, `Delete()` method with running-status guard
- `mcp-servers/pipeline/runner.go` — `RunPipeline()` in goroutine: 3 steps (search → summarize → save), step-level status tracking
- `frontend/src/components/PipelineView.vue` — full-screen pipeline layout: header bar (query input + run button + back to chat), left column (280px runs list with delete ×), center (horizontal flow diagram + output panel for selected step)
- `frontend/src/components/PipelinePanel.vue` — original sidebar pipeline panel (now replaced by PipelineView)
- `frontend/src/stores/pipeline.ts` — Pinia store: `loadList()`, `loadStatus()`, `startPipeline()`, `deleteRun()`, polling with auto-stop
- `frontend/src/stores/ui.ts` — `activeView: 'chat' | 'pipeline'` + `setView()` for view-level switching
- `frontend/src/App.vue` — conditional rendering: PipelineView vs chat layout
- `frontend/src/components/SessionPanel.vue` — pipeline tab removed, "▸ pipeline" button added for view switch
- `mcp-servers/scheduler/store.go` — `TypePipeline` task type
- `mcp-servers/scheduler/main.go` — `pipeline` type in `sched_create` with `query`, `backend_url` params
- `mcp-servers/scheduler/runner.go` — `runPipeline()`: one-shot Timer → HTTP POST to backend `/api/mcp/tools/call` → triggers `pipe_run`

**Key code:** `PipelineView.vue`, `usePipelineStore`, `pipe_delete`, `handleDelete()`, `Store.Delete()`, `runPipeline()`, `pipelineClient`, `TypePipeline`, `activeView`, `setView()`

---

## Day 21 — Document Indexing (RAG) ✅

**Assignment:** Build a RAG document indexing system: upload .txt/.md/.pdf files, run a chunking → embeddings pipeline, store a JSON index, compare chunking strategies side-by-side.

**What was built on top of previous day:**
- `backend/docs.go` — `DocumentStore` (RWMutex + debounced JSON save) + HTTP handlers for `/api/docs/*`. Upload saves file, launches goroutine, returns `DocumentMeta`. Delete blocks if indexing in progress.
- `backend/chunker.go` — 3 strategies: `ChunkSize` (sliding window, rune-level), `ChunkSentence` (sentence boundaries via `.`/`!`/`?`/`\n`), `ChunkStructure` (markdown headings / PDF pages / txt paragraphs)
- `backend/embeddings.go` — Ollama REST client (`POST /api/embeddings`, model: `nomic-embed-text`, 30s timeout)
- `backend/indexer.go` — `runIndexPipeline()` runs all 3 strategies unconditionally, embeds all chunks, saves `CombinedIndex { size, sentence, structure }` to `.documents/index/{id}.json`. `chunk_count` = sum of all 3.
- `backend/pdf.go` — `ExtractPDFText()` via `ledongthuc/pdf` with `defer recover()` panic guard
- `frontend/src/components/DocsView.vue` — full-screen view: header (choose file + upload), left column (doc list with status badges + hover-delete), center (meta bar + 3-column strategy comparison)
- `frontend/src/stores/docs.ts` — Pinia store: `upload()`, `selectDoc()`, `removeDoc()`, polling via `setInterval` every 2s
- `frontend/src/stores/ui.ts` — `activeView` extended to `'chat' | 'pipeline' | 'docs'`
- `frontend/src/components/SessionPanel.vue` — "▸ documents" button added alongside pipeline
- `go.mod` — added `github.com/ledongthuc/pdf`

**Key code:** `DocumentStore`, `runIndexPipeline()`, `CombinedIndex`, `ChunkSize()`, `ChunkSentence()`, `ChunkStructure()`, `GetEmbeddings()`, `DocsView.vue`, `useDocsStore`

---

## Day 22 — Semantic Chunking + UI Polish ✅

**Assignment:** Add 4th chunking strategy (semantic — embedding-based boundary detection), fix text preservation in semantic chunks, replace native tooltips with shadcn Tooltip, migrate custom HTML elements to shadcn components.

**What was built on top of previous day:**
- `backend/chunker.go` — `ChunkSemantic()`: embed each sentence → cosine similarity between adjacent → boundary where sim < mean−1σ. `splitSentenceSpans()` tracks byte offsets to preserve original formatting (newlines, indentation) instead of `strings.Join`
- `backend/similarity.go` — new file: `cosineSimilarity()`, `buildChunkResponse()` strips raw embeddings from API response, returns `ChunkWithSimilarity` with per-chunk similarity + aggregate stats (`StrategyResult`)
- `backend/indexer.go` — pipeline extended to 4 strategies (size, sentence, structure, semantic)
- `frontend/src/components/DocsView.vue` — all `title` attrs → shadcn `Tooltip`/`TooltipContent`/`TooltipProvider`. Custom spans → `Badge`, `Input`, `Button`. Click-to-expand full chunk text. Size/overlap params labeled as size-based strategy. Number input spinners hidden
- `frontend/src/components/ui/tooltip/` — shadcn tooltip component added

**Key code:** `ChunkSemantic()`, `splitSentenceSpans()`, `sentenceSpan`, `cosineSimilarity()`, `buildChunkResponse()`, `ChunkWithSimilarity`, `StrategyResult`

---

## Day 23 — First RAG Query ✅

**Assignment:** Add RAG querying to chat — user selects a document, asks a question, system retrieves relevant chunks via embedding similarity, injects them into the prompt. Two modes: with RAG / without RAG.

**What was built on top of previous day:**
- `backend/similarity.go` — `SearchChunks()` with auto-strategy (searches all 4 strategies, returns best top-K across all), `SearchResult` struct
- `backend/docs.go` — `POST /api/docs/{id}/search` endpoint, `loadCombinedIndex()` extracted as reusable function, `handleSearchDoc()` handler
- `backend/server.go` — `chatRequest` extended with `RagDocIDs`, `RagStrategy`, `RagTopK`. RAG pipeline in `handleChat`: embed query → load index per doc → cross-doc top-K → XML context inject → SSE `rag_context` event. `docStore` passed to `handleChat`
- `frontend/src/lib/types.ts` — `RAGSearchResult`, `RAGContextEvent`, `ragContext` field on `ChatMessage`
- `frontend/src/stores/chat.ts` — `activeRagDocIds` ref, RAG params in request body, `rag_context` SSE handler
- `frontend/src/components/SendSettingsPopover.vue` — "Documents" section with checkboxes for ready docs, emerald highlight on gear icon
- `frontend/src/components/MessageBubble.vue` — collapsible RAG context block above assistant response (chunks with scores and text preview)
- Removed "default" session fallback — new sessions auto-generate `chat-{timestamp}`

**Key code:** `SearchChunks()`, `SearchResult`, `loadCombinedIndex()`, `handleSearchDoc()`, `activeRagDocIds`, `RAGSearchResult`, `RAGContextEvent`

---

## Day 24 — RAG Reranking, Filtering & Pipeline Visibility ✅

**Assignment:** Add a second stage after search: reranker or relevance filter (similarity threshold). Configure cutoff threshold and top-K before/after filtering. Compare quality with and without filter. Show the pipeline visually in UI.

**What was built on top of previous day:**
- `backend/rag.go` (new) — `rewriteQuery()` (Haiku rewrites query for semantic search), `filterResults()` (splits by threshold into passed/rejected), `performRAGSearch()` (full pipeline orchestrator with SSE `rag_step` events at each stage: rewrite → embed → search → filter → inject)
- `backend/server.go` — `chatRequest` extended with `RagThreshold float64` and `RagQueryRewrite bool`; inline RAG block replaced by `performRAGSearch()` call; `ragResultWithDoc` promoted to package-level type
- `frontend/src/components/RAGPipelineSteps.vue` (new) — thinking-steps collapsible block in message flow: per-step icon (✓/↻/→/✗), color (emerald/amber/zinc/red), and contextual detail (rewritten query, chunk count, passed/rejected counts)
- `frontend/src/components/RAGChunkSplitView.vue` (new) — two-column split view: "Before filter (N)" with all top-K chunks, "After filter (M)" with only passed; rejected chunks dimmed; falls back to single column when no filtering
- `frontend/src/components/MessageBubble.vue` — added `RAGPipelineSteps` above RAG context block, replaced flat chunk list with `RAGChunkSplitView` (with fallback for older messages)
- `frontend/src/components/SendSettingsPopover.vue` — RAG Settings section (visible when doc selected): Threshold slider (0–1, step 0.05), Top K input, Query Rewrite checkbox, Strategy select (auto/semantic/sentence/structure/size)
- `frontend/src/stores/chat.ts` — `ragTopK`, `ragThreshold`, `ragQueryRewrite`, `ragStrategy` reactive refs; `rag_step` SSE handler (upserts steps by name); `rag_context` handler extended to store `ragAllResults`, `ragRejected`, `ragRewrittenQuery`, `ragThreshold`
- `frontend/src/lib/types.ts` — `RAGStep`, `RAGStepName`, `RAGStepStatus`, `RAGStepEvent` types; `RAGContextEvent` extended; `ChatMessage` extended with `ragSteps`, `ragAllResults`, `ragRejected`, `ragRewrittenQuery`, `ragThreshold`

**Key code:** `performRAGSearch()`, `rewriteQuery()`, `filterResults()`, `RAGPipelineSteps.vue`, `RAGChunkSplitView.vue`, `ragThreshold`, `rag_step` SSE event
