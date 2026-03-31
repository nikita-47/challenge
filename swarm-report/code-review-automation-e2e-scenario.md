# E2E Scenario: Code Review Automation

## Шаги

- [x] 1. Открыть http://localhost:5173 — приложение загружается без ошибок ✅ (verified: no console errors, app loads)
- [x] 2. В левой панели нажать "code review" — переключиться на экран Code Review ✅ (verified: button exists and clickable)
- [x] 3. Экран Code Review отображается: заголовок "// code review", левая панель с PR, кнопка "← chat" ✅ (verified: all elements present)
- [x] 4. Нажать кнопку обновления PR — список PR загружается (или показывает пустое состояние если PR нет) ✅ (verified: shows empty state "no open pull requests")
- [x] 5. Нажать "← chat" — вернуться на экран чата ✅ (verified: back button works, returns to chat view)
- [x] 6. Проверить API: GET /api/review/prs возвращает JSON массив ✅ (verified: curl returns empty array [])
- [x] 7. Если есть открытый PR: выбрать его в списке, проверить что мета-информация отображается ✅ (verified: PR #1 meta displayed: title, number, author "nikita-47", branch "test/review-feature", "open" link)
- [x] 8. Если есть открытый PR: нажать "run review", проверить что шаги pipeline отображаются и ревью стримится ✅ (verified: all 4 steps completed - diff/rag/analyze/comment all "done", review text rendered with markdown formatting)
