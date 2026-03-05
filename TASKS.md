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
- `backend/taskstate.go` — **NEW** — `TaskState`, `TaskStep`, `TaskPhase` types, FSM transition validation, `applyAction()` for 8 actions (set_plan, start_step, complete_step, fail_step, validate, done, pause, resume), `SystemPromptSection()` for injecting FSM rules + current state into system prompt, `FormatStatus()` for CLI display
- `backend/agent.go` — `TaskState *TaskState` field on `Agent`, `newAgentWithTaskState()` constructor (25 max turns, includes `update_task_state` tool), `taskStateTool()` definition with full JSON schema, inline handling of `update_task_state` in `Run()` loop (before `executeTool`), pause/done early returns, `task_state` AgentEvent emission, TaskState section injected into system prompt in `buildPayload()`
- `backend/history.go` — `TaskState` field in `sessionFile`, persisted in `saveSessionCW()`/`loadSessionCW()`
- `backend/compress.go` — `TaskState` field in `contextWindow`
- `backend/server.go` — `TaskMode bool` in `chatRequest`, task-aware agent creation in `handleChat()` (creates `newAgentWithTaskState` when taskMode or active non-done TaskState), `taskState` in session GET response
- `backend/chat.go` — `/task <goal>` (start task mode), `/task` (show status), `/resume` (resume paused task), `task_state` event display in `cliAgentEmit`
- `backend/main.go` — `/task`, `/resume` in `printHelp()`
- `frontend/src/lib/types.ts` — `TaskPhase`, `TaskStepStatus`, `TaskStep`, `TaskState`, `TaskStateEvent` types, added to `SSEEvent` union
- `frontend/src/stores/chat.ts` — `taskState` ref, `task_state` SSE handler, `startTask()`, `pauseTask()`, `resumeTask()` methods, `taskMode: true` in request body when task active
- `frontend/src/stores/sessions.ts` — `taskState` loaded from session data
- `frontend/src/components/TaskStatePanel.vue` — **NEW** — compact panel with phase indicators (planning→executing→validating→done with active pulse), step list with status icons, progress counter, pause/resume buttons
- `frontend/src/components/ChatInfoPanel.vue` — task info section (phase, progress, goal, expected action)
- `frontend/src/App.vue` — `TaskStatePanel` integrated between toolbar and ChatWindow

**Key code:** `TaskState.applyAction()`, `TaskState.SystemPromptSection()`, `newAgentWithTaskState()`, `update_task_state` tool, `task_state` AgentEvent

---

## Day N — Template

**Assignment:** _paste the assignment here_

**What was built on top of previous day:**
- ...

**Status:** Pending
