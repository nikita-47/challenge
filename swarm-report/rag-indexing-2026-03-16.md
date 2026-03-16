# Day 21: Document Indexing (RAG) — 2026-03-16

## Описание

Реализована система индексации документов для RAG. Новая вкладка Documents позволяет загружать файлы (.txt, .md, .pdf), запускает пайплайн индексации (chunking → embeddings → JSON index), отображает метаданные и сравнение стратегий чанкинга.

## Research (консилиум)

**Go-архитектор:**
- Стандартный `net/http`, 1 зависимость (`mcp-go`). Паттерн для async — pipeline store.
- Рекомендован `ledongthuc/pdf` (pure Go, no CGO) для PDF.
- Ollama через REST API (`POST /api/embeddings`).
- Хранение: `.documents/uploads/` + `.documents/index/` + `.documents/meta.json`.

**Frontend-эксперт:**
- `PipelineView.vue` — точный blueprint для DocsView.
- `ui.ts` расширяется на `'docs'`. Кнопка nav в `SessionPanel.vue` footer.
- Новые: `DocsView.vue`, `stores/docs.ts`.

## Plan

5 новых Go файлов + модификации server.go. Новый `ChunkIndex` JSON формат. Два агента параллельно.

## Реализовано

### Backend (новые файлы)

| Файл | Назначение |
|------|-----------|
| `backend/docs.go` | `DocumentStore` (RWMutex + debounced JSON save) + HTTP handlers |
| `backend/chunker.go` | `ChunkFixed` + `ChunkStructure` (markdown headings / pages / paragraphs) |
| `backend/embeddings.go` | Ollama REST client (`nomic-embed-text`, 30s timeout) |
| `backend/indexer.go` | Async goroutine pipeline: parse → chunk → embed → save JSON |
| `backend/pdf.go` | `ExtractPDFText` с `recover()` panic guard |

### Backend (модификации)

- `backend/server.go` — маршруты `/api/docs/upload`, `/api/docs`, `/api/docs/`
- `.gitignore` — добавлен `.documents/`
- `go.mod` — добавлен `github.com/ledongthuc/pdf`

### Frontend (новые файлы)

| Файл | Назначение |
|------|-----------|
| `frontend/src/components/DocsView.vue` | Full-screen view: header + список документов + мета + сравнение чанков |
| `frontend/src/stores/docs.ts` | Pinia store: upload, polling, selectDoc, delete |

### Frontend (модификации)

- `frontend/src/lib/types.ts` — `DocumentMeta`, `DocumentChunk`, `ChunkIndex`, `ChunkPreview`, `IndexStatus`, `ChunkStrategy`
- `frontend/src/lib/api.ts` — `fetchDocs`, `uploadDoc` (FormData), `fetchDoc`, `deleteDoc`, `fetchDocChunks`, `previewDocChunks`
- `frontend/src/stores/ui.ts` — расширен `activeView` на `'docs'`
- `frontend/src/App.vue` — добавлен `<DocsView v-if="ui.activeView === 'docs'" />`
- `frontend/src/components/SessionPanel.vue` — добавлена кнопка `▸ documents` в footer

## API контракт

```
POST   /api/docs/upload                                    → DocumentMeta (201)
GET    /api/docs                                           → DocumentMeta[]
GET    /api/docs/{id}                                      → DocumentMeta
DELETE /api/docs/{id}                                      → 204
GET    /api/docs/{id}/chunks                               → ChunkIndex
GET    /api/docs/{id}/preview-chunks?strategy=&chunk_size=&overlap=  → ChunkPreview
```

## Validation

E2E сценарий: `swarm-report/rag-indexing-e2e-scenario.md` — все 15 шагов ✅

| Тест | Результат |
|------|----------|
| API: GET /api/docs (пустой) | ✅ |
| API: POST /api/docs/upload (.txt, fixed) | ✅ |
| API: GET /api/docs/{id}/chunks | ✅ embeddings + metadata |
| API: GET /api/docs/{id}/preview-chunks | ✅ оба стратегии |
| API: DELETE /api/docs/{id} | ✅ 204 |
| UI: кнопка "▸ documents" в сайдбаре | ✅ |
| UI: загрузка .txt (fixed) → статус ready | ✅ |
| UI: загрузка .md (structure) → 4 чанка по заголовкам | ✅ |
| UI: preview с chunk_size=500, overlap=50 → 2 чанка | ✅ |
| UI: удаление документа | ✅ |
| UI: 0 JS ошибок в консоли | ✅ |

## Стратегии чанкинга (сравнение)

| | Fixed-size | Structure-based |
|-|------------|----------------|
| Алгоритм | Скользящее окно по рунам | По заголовкам (md) / страницам (pdf) / параграфам (txt) |
| Контроль размера | chunk_size + overlap | Семантические границы |
| Метаданные section | "chunk_N" | Текст заголовка / "page_N" / "para_N" |
| Тест .txt 845B | 1 чанк (1000) / 2 чанка (500) | 8 чанков (параграфы) |
| Тест .md | зависит от size | 4 чанка (по ## заголовкам) |

## Статус: Done ✅
