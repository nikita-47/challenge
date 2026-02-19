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

## Day N — Template

**Assignment:** _paste the assignment here_

**What was built on top of previous day:**
- ...

**Status:** Pending
