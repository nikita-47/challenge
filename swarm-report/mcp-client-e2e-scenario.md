# E2E Scenario: MCP Client Integration

## API Tests (Backend)

- [x] 1. Сервер запускается без `.mcp_servers.json` — не крашится, API отвечает ✅
- [x] 2. GET /api/mcp/servers возвращает пустой массив `[]` ✅
- [x] 3. GET /api/mcp/tools возвращает пустой массив `[]` ✅
- [x] 4. POST /api/mcp/reload возвращает status: "reloaded" с пустыми серверами ✅
- [x] 5. Создать `.mcp_servers.json` с Railway конфигом ✅
- [x] 6. POST /api/mcp/reload — подтянул новый конфиг, railway connected с 14 tools ✅
- [x] 7. GET /api/mcp/servers — railway connected, toolsCount: 14 ✅
- [x] 8. GET /api/mcp/tools — полный список 14 инструментов с inputSchema ✅
- [x] 9. Go build компилируется без ошибок ✅
- [x] 10. `.mcp_servers.json` в `.gitignore` ✅

## UI Tests (Frontend)

- [x] 11. Открыть приложение на http://localhost:5173 ✅
- [x] 12. Найти вкладку "mcp" в левой боковой панели (рядом с "sessions" и "memory") ✅
- [x] 13. Нажать на вкладку "mcp" — должна открыться MCP панель ✅
- [x] 14. Проверить список серверов — должен показать "railway" с бейджем "connected" и "14 tools" ✅
- [x] 15. Развернуть сервер railway — должен показать список всех 14 инструментов с именами и описаниями ✅
- [x] 16. Проверить детали инструмента — каждый инструмент показывает название, описание, и кликабельную ссылку "schema" ✅
- [x] 17. Кликнуть на schema одного из инструментов — должно отображаться JSON с inputSchema ✅
- [x] 18. Нажать кнопку disconnect на railway — бейдж изменился на "disconnected", инструменты исчезли, показ "0 tools" ✅
- [x] 19. Нажать кнопку connect на railway — бейдж вернулся на "connected", все 14 инструментов вернулись ✅
- [x] 20. Нажать кнопку "reload config" — серверы и инструменты обновились (1/1 connected, 14 tools) ✅
- [x] 21. Проверить консоль браузера — нет JavaScript ошибок ✅
- [x] 22. Проверить сетевые запросы — все API endpoints вернули 200 OK ✅
