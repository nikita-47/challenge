# MCP Client Integration — 2026-03-10

## Задача
День 16: Подключение MCP. Реализовать MCP-клиент, который устанавливает соединение с MCP-серверами и получает список инструментов. UI для управления. Первая интеграция — Railway MCP.

## Research (консилиум)

**Go-архитектор:**
- Backend — flat `package main`, 17 файлов, zero deps
- Рекомендация: файлы в `backend/`, mcp-go библиотека (JSON-RPC 2.0 с нуля = overengineering)
- Ключевые точки интеграции: `startServer()` для роутов, MCPManager singleton

**Frontend-эксперт:**
- SPA без роутера, навигация через табы в сайдбаре
- shadcn-vue + Tailwind v4, Pinia stores, чистый fetch
- Рекомендация: третий таб "mcp" в SessionPanel, новый store + компонент

## Plan
1. Добавить `mcp-go` v0.45.0 (первая внешняя зависимость)
2. `backend/mcp.go` — MCPManager + HTTP API
3. Wiring в `server.go`
4. Frontend: types → api → store → MCPPanel → SessionPanel интеграция

## Реализовано

### Backend
- **`backend/mcp.go`** (418 строк) — MCPServerConfig, MCPConnection, MCPManager (sync.RWMutex), 5 HTTP-хэндлеров
- **`backend/server.go`** — инициализация MCPManager + 5 роутов `/api/mcp/*`
- **`go.mod`** — добавлен `github.com/mark3labs/mcp-go v0.45.0`
- **`.mcp_servers.example.json`** — пример конфига
- **`.gitignore`** — добавлен `.mcp_servers.json`

### Frontend
- **`frontend/src/lib/types.ts`** — MCPServerStatus, MCPToolInfo
- **`frontend/src/lib/api.ts`** — 5 MCP API функций
- **`frontend/src/stores/mcp.ts`** — Pinia store (servers, tools, loading, error)
- **`frontend/src/components/MCPPanel.vue`** — панель: серверы, статусы, collapsible tools, schema viewer
- **`frontend/src/components/SessionPanel.vue`** — третий таб "mcp"

### API endpoints

| Метод | Путь | Назначение |
|-------|------|-----------|
| GET | `/api/mcp/servers` | Статус серверов |
| POST | `/api/mcp/servers/{name}/connect` | Подключить |
| POST | `/api/mcp/servers/{name}/disconnect` | Отключить |
| GET | `/api/mcp/tools?server=name` | Список инструментов |
| POST | `/api/mcp/tools/call` | Вызов инструмента |
| POST | `/api/mcp/reload` | Перечитать конфиг |

## Validation

22/22 тестов пройдено (100%):
- 10 API тестов: пустой конфиг, reload, connect, tools list, build
- 12 UI тестов: таб, серверы, tools, schema, connect/disconnect, reload, console clean

Подробности: `mcp-client-e2e-scenario.md`, `mcp-ui-verification-2026-03-10.md`

## Проблемы и откаты
Нет.

## Статус: Done
