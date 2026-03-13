# E2E Scenario: Pipeline Full-Screen Layout

## Шаги

- [x] 1. Открыть приложение (http://localhost:5173 или :5174), убедиться что chat view загружается нормально ✅
- [x] 2. В левом sidebar найти кнопку "▸ pipeline" внизу (перед GlobalSettings), кликнуть — должен открыться полноэкранный Pipeline View ✅
- [x] 3. Проверить layout PipelineView: header bar (// pipeline, input, back to chat), left column (runs), center area (flow diagram + output) ✅
- [x] 4. Кликнуть "← back to chat" — должен вернуться в chat view ✅
- [x] 5. Снова перейти в pipeline view. Проверить что в sidebar нет таба "pipeline" (только sessions, memory, mcp) ✅ (в Pipeline View нет левого sidebar вообще — это правильно)
- [x] 6. Если pipeline MCP сервер подключён — проверить что runs list загружается ✅ (список запусков загружается, детали запуска отображаются, output для шагов работает)
