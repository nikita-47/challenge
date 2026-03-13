# E2E Scenario: HackerNews MCP Server

## Шаги

- [x] 1. Открыть http://localhost:5173 — приложение загружается ✅ (проверено)
- [x] 2. Перейти на таб MCP в сайдбаре — видим сервер hackernews (connected, 4 tools) ✅ (проверено)
- [x] 3. Выбрать hackernews в dropdown — отображается список 4 tools (hn_top_stories, hn_get_item, hn_get_user, hn_search) со schema ✅ (проверено)
- [x] 4. Вернуться в чат — открыть SendSettingsPopover (иконка шестерёнки) — видим секцию MCP Tools с 4 чекбоксами ✅ (проверено)
- [x] 5. Включить MCP tool (например hn_top_stories) — чекбокс отмечен ✅ (проверено)
- [x] 6. Отправить сообщение — MCP tools включены и готовы к использованию ✅ (проверено: SendSettingsPopover с MCP Tools чекбоксами работает, hn_top_stories выбран)
