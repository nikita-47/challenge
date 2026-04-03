# AI Challenge — Chat with Claude

35-дневный AI-челлендж. Каждый день — новая фича поверх предыдущей.
Задания и статус — в `TASKS.md`.

## Структура проекта

- `backend/` — Go backend
- `frontend/` — Vue 3 frontend
- `go.mod` — в корне проекта
- `TASKS.md` — журнал задач (задания, статус, заметки)
- `.env` — хранит `ANTHROPIC_API_KEY` (не коммитится)
- `.chat_history/` — сохранённые сессии чата (не коммитится)
- `.memory/` — слои памяти: `profiles/` (долгосрочная) + `projects/` (рабочая) + `operators/` (неизменяемая идентичность пользователя), .md файлы (не коммитится)
- `.claude/agents/` — конфигурации субагентов: `go-backend-specialist.md` (backend, sonnet), `vue-frontend-specialist.md` (Vue/TS, sonnet), `qa-tester.md` (тестировщик, haiku)
- `.sandbox/` — песочницы для агентов (не коммитится)
- `mcp-servers/` — кастомные MCP-серверы (Go бинарники, stdio транспорт)
- `mcp-servers/hackernews/` — MCP-сервер для HackerNews API (4 tools: top stories, get item, get user, search)
- `mcp-servers/scheduler/` — MCP-сервер планировщика (6 tools: create, list, status, delete, pause, resume). 4 типа задач: reminder, url_monitor, hn_digest, pipeline. JSON persistence, горутины с Ticker/Timer
- `mcp-servers/pipeline/` — MCP-сервер pipeline (7 tools: search, summarize, save, run, status, list, delete). HN Algolia → Claude Haiku → file save. Async execution в горутинах
- `.mcp_servers.json` — конфиг MCP-серверов (не коммитится), формат Claude Desktop
- `.mcp_servers.example.json` — пример конфига MCP
- `.documents/` — загруженные документы: `uploads/` (файлы), `index/` (CombinedIndex JSON), `meta.json` (не коммитится)
- `deploy/ollama-railway/` — Ollama + auth proxy для Railway (Dockerfile, entrypoint.sh, main.go)
- `docs/` — документация проекта (api-reference, data-schemas, architecture, mcp-servers, frontend-guide). Авто-индексируется в RAG при старте сервера
- `mcp-servers/devtools/` — MCP-сервер контекста проекта (7 tools: dev_git_branch, dev_git_status, dev_git_log, dev_list_files, dev_read_file, dev_grep, dev_write_file)
- `mcp-servers/tickets/` — MCP-сервер тикетов поддержки (5 tools: ticket_create, ticket_list, ticket_get, ticket_close, ticket_add_message). JSON persistence, stdio транспорт
- `docs/support/` — FAQ документы для ассистента поддержки (faq-general.md, faq-features.md, faq-troubleshooting.md). Авто-индексируются в RAG при старте

## Правила

- `.env` никогда не коммитить
- При тестировании стратегий контекста использовать самую дешёвую модель (`claude-3-5-haiku-20241022`) и размер окна N=3
- **Тестирование — ТОЛЬКО через локальную LLM (LM Studio).** Оркестратор и QA-агент при Validation ОБЯЗАНЫ использовать локальную LLM вместо Claude API. Запуск: `./dev.sh start-test`. При необходимости можно переключать модели в LM Studio. Цель — экономия токенов: ни один тестовый запрос не должен уходить в Claude API.

## Ключевые решения

- **Frontend стек**: Vue 3 + Vite + Tailwind CSS v4 + Pinia + marked + shadcn-vue (radix-vue, class-variance-authority)
- **Загрузка `.env`**: собственный парсер, без сторонних dotenv-библиотек
- **Модель памяти**: 4 слоя — краткосрочная (сообщения чата), рабочая (`.memory/projects/*.md`), долгосрочная (`.memory/profiles/*.md`), операторская (`.memory/operators/*.md`, неизменяемая). Порядок в `buildFullSystemPrompt()`: operator → profile → project → system prompt. Profile/project автообновляются через `maybeUpdateMemory()` (Haiku) после каждого обмена. Сохраняется в `sessionSettings`.
- **Оптимизация токенов агента**: `compactHistory()` сжимает tool_result блоки старше 2 turns в однострочные summary, `formatStepProgress()` инжектит прогресс в goal message. Prompt caching через `cache_control: ephemeral` на system prompt и последнем tool. Efficiency Rules в промптах фаз. Результат: -78% input tokens, -84% стоимость.
- **Инварианты**: пользовательские ограничения (invariants) передаются при создании задачи и инжектируются во все phase prompts через `formatInvariantsBlock()`. Агент обязан проверять инварианты перед каждым действием.
- **Shared sandbox**: все фазы задачи работают в одной песочнице (`TaskState.SandboxDir`), создаётся через `EnsureSandbox()`, удаляется через `CleanupSandbox()` при завершении задачи.
- **MCP клиент**: `MCPManager` в `backend/mcp.go` управляет множеством MCP-серверов через `mcp-go` библиотеку (первая внешняя зависимость). Конфиг `.mcp_servers.json` (формат Claude Desktop). Stdio транспорт — MCP-сервер как subprocess. HTTP API: `/api/mcp/servers`, `/api/mcp/tools`, `/api/mcp/tools/call`, `/api/mcp/reload`. Frontend: таб "mcp" в сайдбаре с Select dropdown, connect/disconnect, список tools.
- **MCP сервер (HackerNews)**: `mcp-servers/hackernews/main.go` — standalone бинарник, stdio транспорт через `mcp-go/server`. 4 tools: `hn_top_stories`, `hn_get_item`, `hn_get_user`, `hn_search`. No API keys. Сборка: `go build -o mcp-servers/hackernews/hackernews ./mcp-servers/hackernews/`. `dev.sh` собирает автоматически при `start-go`.
- **Agent ↔ MCP мост**: `GetToolDefs()` конвертирует MCP tools → agent `toolDef`. Naming: `server__toolname` (двойной underscore). `executeTool()` роутит MCP calls через `MCPManager.CallTool()`. Frontend: `activeMcpTools` в chat store, `mcpTools` в chatRequest. UI: SendSettingsPopover с группировкой по серверам.
- **MCP сервер (Scheduler)**: `mcp-servers/scheduler/` — 3 файла (main.go, store.go, runner.go), stdio транспорт. 6 tools: `sched_create`, `sched_list`, `sched_status`, `sched_delete`, `sched_pause`, `sched_resume`. 4 типа задач: `reminder` (one-shot Timer), `url_monitor` (periodic Ticker + HTTP GET), `hn_digest` (periodic HN fetch), `pipeline` (one-shot Timer → HTTP POST to backend `/api/mcp/tools/call`). `Store` с `sync.RWMutex` + debounced JSON save. Горутины привязаны к `rootCtx` (не к request ctx). Восстановление активных задач при старте. Сборка: `go build -o mcp-servers/scheduler/scheduler ./mcp-servers/scheduler/`. `dev.sh` собирает автоматически.
- **MCP сервер (Pipeline)**: `mcp-servers/pipeline/` — 3 файла (main.go, store.go, runner.go), stdio транспорт. 7 tools: `pipe_search`, `pipe_summarize`, `pipe_save`, `pipe_run`, `pipe_status`, `pipe_list`, `pipe_delete`. 3 шага: search (HN Algolia) → summarize (Claude Haiku) → save (файл). Async execution в горутинах. `pipe_delete` запрещает удаление running runs. Сборка: `go build -o mcp-servers/pipeline/pipeline ./mcp-servers/pipeline/`. `dev.sh` собирает автоматически.
- **Pipeline UI**: Полноэкранный layout `PipelineView.vue` вместо sidebar таба. `activeView` в ui store переключает между chat и pipeline view. Layout: header (query + run + back), left column (runs list + delete), center (horizontal flow diagram + output panel). `usePipelineStore` — MCP tool calls, polling, delete.
- **Scheduler → Pipeline мост**: Тип задачи `pipeline` в scheduler. `runPipeline()` — one-shot Timer → HTTP POST на backend `/api/mcp/tools/call` с `pipe_run`. Позволяет отложенный запуск pipeline из чата через агента.
- **RAG Document Indexing**: `backend/docs.go` — `DocumentStore` + REST `/api/docs/*` (upload, list, get, delete, chunks). `runIndexPipeline()` запускает все 4 стратегии (`ChunkSize`, `ChunkSentence`, `ChunkStructure`, `ChunkSemantic`), сохраняет `CombinedIndex { size, sentence, structure, semantic }` + embeddings через Ollama (`nomic-embed-text`). `backend/similarity.go` — `cosineSimilarity()`, `buildChunkResponse()` (strips embeddings, returns similarity stats). `backend/chunker.go` — `ChunkSemantic()` определяет смысловые границы через cosine similarity соседних предложений (boundary при sim < mean−1σ), `splitSentenceSpans()` сохраняет оригинальное форматирование через byte offsets. `backend/pdf.go` — `ExtractPDFText()` с `recover()` panic guard. Frontend: `DocsView.vue` (полноэкранный, shadcn Tooltip/Badge/Input/Button, click-to-expand chunks) + `useDocsStore` (polling каждые 2s). `activeView` расширен до `'chat' | 'pipeline' | 'docs'`.
- **RAG Query**: `chatRequest` расширен полями `ragDocIds`, `ragStrategy`, `ragTopK`, `ragThreshold`, `ragQueryRewrite`. RAG пайплайн вынесен в `backend/rag.go` → `performRAGSearch()`: query rewrite (Haiku) → embed → search → threshold filter → inject XML. SSE события: `rag_step` (промежуточные шаги: rewrite/embed/search/filter/inject) + `rag_context` (enriched: results, all_results, rejected, rewritten_query, threshold). Frontend: `RAGPipelineSteps.vue` (thinking-steps блок с анимацией в потоке сообщений), `RAGChunkSplitView.vue` (split view: до/после фильтра, 2 колонки). `SendSettingsPopover` — секция RAG Settings (threshold slider, top-K, rewrite checkbox, strategy select). Ограничение: nomic-embed-text слабо работает кросс-лингвально.
- **RAG Citations & Anti-Hallucination**: Citation rules prompt заставляет модель использовать `[N]` инлайн-цитаты + `<!-- sources [...] -->` скрытый JSON-блок. `parseRAGSources()` извлекает блок, `renderCitations()` заменяет `[N]` на `.rag-cite` бейджи (с защитой `<a>`/`<code>`/`<pre>`). `RAGSourcesBlock.vue` — сворачиваемый блок источников (ref, filename, chunk_id, score%). Anti-hallucination: `<rag_no_context>` инструкция при пустых результатах или threshold-фильтрации, SSE `no_context` флаг, amber-баннер в UI.
- **Railway Ollama Deploy**: `deploy/ollama-railway/` — Ollama + tinyllama на Railway, Go auth reverse proxy (`main.go`) с Bearer-токеном и per-IP rate limiter (30 req/min). `providerSettings.LocalKey` — API key для удалённых LLM. `streamChatOpenAI()` отправляет `Authorization: Bearer` при наличии ключа. Три провайдера в UI: `claude | local | railway`. Railway = local с пресетом URL/модели. Домен: `https://ollama-proxy-production-5097.up.railway.app`.
- **Developer Assistant**: `docs/` содержит 5 md-файлов с документацией проекта (API, схемы данных, архитектура, MCP, frontend). `autoIndexProjectDocs()` в `backend/docs.go` сканирует `docs/*.md` при старте сервера и автоматически индексирует через существующий RAG pipeline (Ollama nomic-embed-text). Команда `/help` в `handleChat()` перехватывает сообщения с префиксом `/help`, автоматически подставляет все проиндексированные документы в RAG, добавляет system prompt ассистента. Frontend: `isHelpCommand` computed в ChatInput.vue показывает Badge при вводе `/help`. MCP-сервер `devtools` (5 tools: git branch/status/log, file listing, file read) предоставляет контекст проекта через MCP.
- **Code Review Automation**: `backend/review.go` — AI-ревью PR из UI. `listOpenPRs()` / `getPRDiff()` / `postPRComment()` через `gh` CLI (`exec.Command`). `buildReviewRAGContext()` обогащает diff контекстом из проектной документации через RAG (graceful degradation если Ollama недоступен). `performCodeReview()` — 4-step SSE pipeline: diff → rag → analyze (streaming Claude Sonnet, temperature 0) → comment (постит в PR на GitHub). API: `GET /api/review/prs`, `POST /api/review/run`. Frontend: `ReviewView.vue` (полноэкранный layout с списком PR, pipeline-индикатором, markdown ревью), `useReviewStore` (SSE streaming, abort). Навигация: `activeView = 'review'`, кнопка в SessionPanel.
- **Support Assistant Widget**: `backend/support.go` — `handleSupportChat()` отдельный endpoint `POST /api/support/chat`. Фильтрует FAQ docs (OriginalName содержит "faq-"), получает контекст тикета через MCP `ticket_get`, вызывает `performRAGSearch()` с FAQ doc IDs, стримит ответ Claude Haiku. `autoIndexProjectDocs()` расширен для `docs/support/*.md`. MCP-сервер `mcp-servers/tickets/` (2 файла: main.go, store.go) — 5 tools: `ticket_create`, `ticket_list`, `ticket_get`, `ticket_close`, `ticket_add_message`. Frontend: `SupportWidget.vue` — floating виджет внутри chat main area (absolute bottom-right), circle "?" → chat panel (w-96 h-500px). `useSupportStore` — Pinia store с SSE streaming, history (last 10), abort. Markdown через `marked`.
- **File Assistant**: `backend/files.go` — `/files` команда с agentic tool-use loop (до 10 turns). `handleFilesChat()` собирает devtools MCP tools, стрипает `devtools__` prefix, вызывает Claude Sonnet с tools, обрабатывает tool_use → MCP call → tool_result циклически. SSE events: text_delta, tool_call, tool_result, usage, done. `filesCallClaude()` — non-streaming вызов Claude API. Devtools MCP расширен: `dev_grep` (pattern search, glob filter, max 200 results), `dev_write_file` (создание файлов, блокирует .env/.git). Frontend: `sendFilesMessage()` в chat store, `/files` Badge в ChatInput. API: `POST /api/files/chat`.
- **Динамические фазы**: вместо хардкоженного pipeline (planning→executing→validating→done) задача начинается с `PhaseProposing` — агент анализирует goal и предлагает кастомный pipeline через `submit_phases`. Пользователь может одобрить (approve) или отправить feedback для корректировки. `PhaseSpec` (Name/Type/Description/Status) маппится на существующих агентов по Type: `planning`→`newPlanningAgent`, `executing`→`newExecutingAgent`, `validating`→`newValidatingAgent`. Constraints: planning before executing, must end with validating, 2-5 phases. `validatePhases()` проверяет перед принятием.

## Рабочий процесс

Dev-серверы должны работать на протяжении всей сессии. Запускать один раз в начале, Go перезапускать только после изменений `.go` файлов.

Каждая задача ОБЯЗАНА проходить через стадии. Перескакивать стадии или переходить по неразрешённым путям ЗАПРЕЩЕНО.

### Стадии и переходы

1. **Research** — исследование задачи, кодовой базы, зависимостей (консилиум агентов)
2. **Plan** — формирование плана реализации
3. **Executing** — написание кода
4. **Validation** — проверка результата
5. **Done** — задача завершена

Разрешённые переходы:

```
Research   -> Plan
Research   -> Executing
Plan       -> Executing
Executing  -> Validation
Executing  -> Plan
Executing  -> Research
Validation -> Executing
Validation -> Plan
Validation -> Research
Validation -> Done
```

Все остальные переходы ЗАПРЕЩЕНЫ. Перед сменой стадии — явно указывать текущую и следующую стадию.

### Субагенты

Каждая стадия выполняется ОТДЕЛЬНЫМ субагентом если имеется подходящий. Главный контекст — оркестратор, он НЕ выполняет работу стадий напрямую, а только:

- управляет переходами между стадиями
- передаёт контекст между субагентами
- показывает пользователю краткие итоги каждой стадии

Субагенты по типу задач:

| Стадия     | Agent tool params                             | Инструкции из             |
| ---------- | --------------------------------------------- | ------------------------- |
| Research   | КОНСИЛИУМ (см. ниже)                          | Роль-специфичный prompt   |
| Plan       | subagent_type: Plan, model: opus              | —                         |
| Executing  | subagent_type: general-purpose, model: sonnet | .claude/agents/go или vue |
| Validation | subagent_type: general-purpose, model: haiku  | .claude/agents/qa         |
| Done       | Главный чат (без делегирования)               | —                         |

#### Механизм делегирования

При делегировании стадии субагенту оркестратор ОБЯЗАН:

1. **Прочитать** файл `.claude/agents/<name>.md` через Read tool
2. **Извлечь `model`** из frontmatter агента (`sonnet` / `haiku`) — передать в параметр `model` Agent tool
3. **Вставить полное содержимое** файла (без frontmatter) в `prompt` агента
4. **Добавить в prompt контекст стадии** (см. «Передача контекста» ниже)
5. Вызвать `Agent tool` с `subagent_type: "general-purpose"`

Пример вызова:

```
Agent(
  subagent_type: "general-purpose",
  model: "sonnet",
  prompt: "<содержимое .claude/agents/go.md без frontmatter>\n\n---\n\nЗадача: ...\nКонтекст: ..."
)
```

#### Передача контекста между стадиями

При запуске субагента в prompt ОБЯЗАТЕЛЬНО передавать:

1. Исходный запрос пользователя
2. Краткий итог предыдущей стадии (результат субагента)
3. Если откат — причину отката

Результат каждого субагента сохраняется как краткое резюме в оркестраторе для передачи на следующую стадию.

### Research — Консилиум агентов

Research выполняется НЕ одним агентом, а консилиумом. Все агенты запускаются **параллельно** через Agent tool и каждый анализирует задачу со своей экспертизы:

| Роль             | subagent_type | model | Зона ответственности                        |
| ---------------- | ------------- | ----- | ------------------------------------------- |
| Go-архитектор    | Explore       | opus  | Go архитектура, модули, зависимости, API    |
| Frontend-эксперт | Explore       | opus  | Vue компоненты, UI/UX, Pinia, маршрутизация |

**Порядок работы Research:**

1. Оркестратор запускает обоих агентов **параллельно** с описанием задачи
2. Собирает результаты от каждого агента
3. Формирует сводное резюме консилиума
4. Использует `AskUserQuestion` для уточнения, пока весь контекст не собран
5. Только после полного сбора контекста — переход на следующую стадию

### Plan — субагент

Plan выполняется встроенным субагентом `Plan` (subagent_type: Plan, model: opus). Оркестратор передаёт ему:

1. Исходный запрос пользователя
2. Сводное резюме консилиума Research
3. Конкретные рекомендации/ограничения от каждого эксперта

Plan-агент формирует пошаговый план реализации, определяет затронутые файлы и архитектурные решения. Результат возвращается оркестратору для передачи на Executing.

### Validation

**Шаг 1:** Сформировать E2E сценарий и сохранить в файл:

```
./swarm-report/<slug>-e2e-scenario.md
```

Файл содержит чеклист всех шагов пользовательского сценария в формате:

```markdown
# E2E Scenario: <название>

## Шаги

- [ ] 1. Открыть экран X
- [ ] 2. Нажать кнопку Y
- [ ] 3. Проверить что отображается Z
- [ ] 4. ...
```

Каждый шаг — конкретное действие пользователя + ожидаемый результат.
Сценарий формируется ОДИН РАЗ перед началом проверок и является источником правды.

**Шаг 2:** Запустить сборку. Использовать `dev.sh`.

**Шаг 3:** UI/E2E проверки. Использовать Playwright MCP и субагента qa.

**Шаг 4:** После прохождения каждого шага — обновить файл сценария, пометив шаг как выполненный:

```markdown
- [x] 1. Открыть экран X ✅ (проверено)
- [ ] 2. Нажать кнопку Y
```

**ВАЖНО — устойчивость к компактизации контекста:**

- Файл `*-e2e-scenario.md` является персистентным состоянием валидации
- Перед каждым действием в Validation — ПЕРЕЧИТАТЬ файл сценария через Read tool
- Выполненные шаги (`[x]`) — НЕ проверять повторно
- Продолжать с первого невыполненного шага (`[ ]`)
- Это гарантирует что после компактизации тесты не начнутся заново и не будут плавать

**Шаг 5:** Если есть ошибки — откат с описанием проблем. Файл сценария сохраняется, невыполненные шаги остаются.

### Отчёты

Отчёт каждой задачи сохраняется в `./swarm-report/`. Формат имени:

```
./swarm-report/<slug>-<YYYY-MM-DD>.md
```

Пример: `./swarm-report/telegram-notifications-2026-02-20.md`

Содержимое отчёта:

- Название фичи и дата
- Краткое описание задачи
- Итоги Research (сводка консилиума)
- Результаты Plan (план реализации)
- Что реализовано (файлы, модули)
- Результаты Validation (тесты, платформы)
- Проблемы и откаты (если были)
- Статус: Done / Частично

## Инструменты разработки

### Dev servers (`dev.sh`)

```bash
./dev.sh start        # запустить Go + Vite
./dev.sh stop         # остановить всё (Go + Vite + Playwright)
./dev.sh restart-go   # после изменений .go файлов
./dev.sh status       # проверить что запущено
```

Дополнительные команды: `start-go`, `stop-go`, `start-vite`, `stop-vite`, `stop-playwright`.

- **Изменён Go код** → `./dev.sh restart-go`
- **Изменён frontend код** → ничего, Vite HMR подхватит
- **Vite dev server** → не перезапускать, только если изменился config

### Тестовый режим (локальная LLM)

Для UI-тестирования использовать локальную LLM вместо Claude API:

```bash
./dev.sh start-test   # LM Studio + Go (с локальной LLM) + Vite
./dev.sh stop-test    # остановить всё
```

Модель: qwen2.5-0.5b-instruct-mlx (0.5B, мгновенные ответы, 0 токенов).

### Сборка и проверка типов

Запускать перед браузерным тестированием:

```bash
go build -o challenge ./backend      # компиляция Go
cd frontend && npx vue-tsc --noEmit  # проверка TypeScript
```

### Браузерная верификация (Playwright MCP)

После реализации фичи — **всегда проверять в браузере** перед отчётом:

1. Открыть Vite dev URL (обычно `http://localhost:5173` или `:5174`)
2. Протестировать фичу end-to-end: взаимодействие с UI, отправка сообщений, проверка ответов
3. При проблемах: проверить `browser_console_messages` и `browser_network_requests`
