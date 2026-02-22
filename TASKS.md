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

## Day N — Template

**Assignment:** _paste the assignment here_

**What was built on top of previous day:**
- ...

**Status:** Pending
