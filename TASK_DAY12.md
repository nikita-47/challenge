# День 12: Персонализация ассистента

## Что делаем

1. Новый слой памяти «operator» — неизменяемый профиль пользователя, всегда первый в system prompt
2. Auto-update profile/project агентом после каждого обмена (LLM анализирует и решает)
3. Frontend поддержка для operators

## Часть 1: Go Backend — Operator layer + Memory auto-update

**Делегировать агенту `@go`** с задачей:

### 1.1 `memory.go` — добавить operator CRUD (после строки 94)

```go
func memoryOperatorsDir() string                 { return filepath.Join(memoryDir, "operators") }
func listOperators() ([]string, error)           { return listMemoryFiles(memoryOperatorsDir()) }
func getOperator(name string) (string, error)    { return getMemoryFile(memoryOperatorsDir(), name) }
func saveOperator(name, content string) error     { return saveMemoryFile(memoryOperatorsDir(), name, content) }
func deleteOperator(name string) error            { return deleteMemoryFile(memoryOperatorsDir(), name) }
```

### 1.2 `history.go` — поле Operator в sessionSettings

```go
Operator string `json:"operator,omitempty"` // operator profile name
```

Также добавить `Operator` в `sessionInfo` struct и в `listSessions()` маппинг.

### 1.3 `api.go` — обновить `buildFullSystemPrompt()`

Добавить operator ПЕРВЫМ в список секций (перед Profile):

```go
if settings != nil && settings.Operator != "" {
    if content, err := getOperator(settings.Operator); err == nil && content != "" {
        sections = append(sections, "# Operator\n\n"+content)
    }
}
```

### 1.4 `server.go` — эндпоинты + chatRequest

Эндпоинты (после projects, строка ~82):
```go
mux.HandleFunc("/api/memory/operators", func(w http.ResponseWriter, r *http.Request) {
    handleMemoryList(w, r, memoryOperatorsDir(), saveOperator)
})
mux.HandleFunc("/api/memory/operators/", func(w http.ResponseWriter, r *http.Request) {
    name := strings.TrimPrefix(r.URL.Path, "/api/memory/operators/")
    handleMemoryItem(w, r, name, memoryOperatorsDir())
})
```

В `chatRequest` добавить `Operator string \`json:"operator"\``.

В `handleChat` (после строки ~210):
```go
if req.Operator != "" && cw.Settings.Operator == "" {
    cw.Settings.Operator = req.Operator
}
```

### 1.5 Новый файл `memory_update.go` — auto-update

По паттерну `strategy_facts.go` (`maybeExtractFacts` + `extractFacts`).

**`maybeUpdateMemory(apiKey, cw, stats) error`**:
- Проверяет: profile/project в settings, ≥2 сообщения (user + assistant)
- Для каждого непустого (profile, project) вызывает `analyzeMemoryUpdate()`
- Если LLM вернул контент (не `NO_UPDATE`) — сохраняет через `saveProfile()`/`saveProject()`
- Operator НЕ обновляется (immutable)

**`analyzeMemoryUpdate(apiKey, kind, currentContent, conversation) (string, tokenUsage, error)`**:
- Модель: `claude-3-5-haiku-20241022`
- max_tokens: 1024
- Non-streaming API вызов (как `extractFacts`)
- System prompt: вернуть `NO_UPDATE` если ничего нового, или полный обновлённый markdown

### 1.6 `server.go` — подключить в handleChat

После facts extraction (строка ~271), перед сохранением:
```go
if cw.Settings.Profile != "" || cw.Settings.Project != "" {
    if err := maybeUpdateMemory(apiKey, cw, &stats); err != nil {
        // Non-fatal
    } else {
        sseWrite(w, map[string]any{"type": "memory_updated"})
    }
}
```

### Проверка Go части
```bash
go build .
go vet ./...
./dev.sh restart-go
curl -s http://localhost:8080/api/memory/operators | jq .
```

---

## Часть 2: Frontend — Operator UI + memory_updated event

**Делегировать агенту `@frontend`** с задачей:

### 2.1 `frontend/src/lib/types.ts`
- Добавить `operator?: string` в `ChatSettings`
- Добавить `| { type: 'memory_updated' }` в `SSEEvent`

### 2.2 `frontend/src/lib/api.ts`
Добавить (по паттерну profiles):
```ts
fetchOperators(): Promise<string[]>
fetchOperator(name): Promise<{ name: string; content: string }>
createOperator(name, content)
updateOperator(name, content)
deleteOperatorAPI(name)
```

### 2.3 `frontend/src/stores/memory.ts`
- Добавить `operators` ref
- Добавить `loadOperators()`, `addOperator()`, `removeOperator()`
- Обновить `loadAll()` — добавить `loadOperators()`

### 2.4 `frontend/src/stores/chat.ts`
- Добавить `operator: settings.value.operator` в body запроса (строка ~93)
- Добавить `case 'memory_updated': break` в switch event handler

### 2.5 `frontend/src/components/NewChatDialog.vue`
- Добавить `operator` ref (как profile/project, с NONE)
- Добавить Select dropdown ПЕРЕД profile (первый в списке — приоритетный слой)
- Label: "Operator (user identity)"
- Пробросить в confirm() → ChatSettings

### 2.6 `frontend/src/components/MemoryEditorDialog.vue`
- Расширить тип `kind`: `'profile' | 'project' | 'operator'`

### 2.7 `frontend/src/components/SessionPanel.vue`
- Импортировать `fetchOperator`, `updateOperator` из api
- Расширить все типы `'profile' | 'project'` → `'profile' | 'project' | 'operator'`
- В `openEditDialog` добавить fetcher для operator
- В `handleEditorSave` и `handleDelete` добавить ветку для operator
- В template: добавить секцию `// operators` ПЕРЕД profiles (с разделителем)

### 2.8 `frontend/src/components/ChatInfoPanel.vue`
- В условии memory секции: добавить `chat.settings?.operator`
- Добавить отображение operator (перед profile)

### Проверка Frontend части
```bash
cd frontend && npx vue-tsc --noEmit
```

---

## Верификация (можно через `@qa`)

1. Создать operator `.memory/operators/Nikolay.md`
2. Создать чат с operator + profile + project → system prompt в правильном порядке
3. Провести обмены → profile/project обновляются (проверить файлы в `.memory/`)
4. Operator НЕ перезаписывается
5. UI: NewChatDialog показывает operator select, memory editor поддерживает operators
