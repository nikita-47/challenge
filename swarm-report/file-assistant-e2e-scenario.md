# E2E Scenario: File Assistant (/files command)

## Шаги

- [x] 1. Открыть http://localhost:5173 — чат загружается ✅ (проверено)
- [x] 2. Ввести "/files" в поле ввода — появляется Badge "/files — file assistant" и placeholder "Work with project files..." ✅ (проверено)
- [ ] 3. Отправить "/files найди все использования MCPManager в Go коде" — ассистент использует dev_grep, dev_read_file и выдаёт summary
- [ ] 4. Проверить что tool_call и tool_result отображаются в UI (ToolCallCard)
- [ ] 5. Отправить "/files сгенерируй CHANGELOG.md на основе последних 20 коммитов" — ассистент использует dev_git_log, dev_write_file и создаёт файл
- [ ] 6. Проверить что CHANGELOG.md создан на диске с содержимым
