# E2E Scenario: Scheduler MCP Server

## Шаги

- [x] 1. Go backend компилируется без ошибок (`go build -o challenge ./backend`) ✅ (проверено)
- [x] 2. Scheduler MCP-сервер компилируется (`go build -o mcp-servers/scheduler/scheduler ./mcp-servers/scheduler/`) ✅ (проверено)
- [x] 3. Dev-серверы запущены (`./dev.sh restart-go`) ✅ (проверено, scheduler собран)
- [x] 4. Открыть UI (http://localhost:5173 или 5174), чат загружается ✅ (проверено)
- [x] 5. В sidebar MCP panel: подключить scheduler сервер (connect) ✅ (уже подключен, видна кнопка "off")
- [x] 6. Scheduler tools (6 штук) отображаются в MCP panel ✅ (все 6 tools видны: sched_create, sched_delete, sched_list, sched_pause, sched_resume, sched_status)
- [x] 7. В SendSettingsPopover: включить scheduler tools (checkboxes) ✅ (все 6 tools выбраны, 6/6)
- [x] 8. API: sched_create работает ✅ (создана задача ID 16e40aeb с type=url_monitor)
- [x] 9. API: sched_list показывает созданную задачу ✅ (задача видна в списке с статусом active)
- [x] 10. API: После 35 сек ожидания sched_status показывает результаты ✅ (1 successful check, 100% uptime, 200 OK)
- [x] 11. API: sched_pause работает ✅ (статус изменён на paused)
- [x] 12. API: sched_resume работает ✅ (статус вернулся на active)
- [x] 13. API: sched_delete работает ✅ (задача удалена)
- [x] 14. API: sched_list показывает пустой список ✅ (No scheduled tasks)
- [x] 15. /api/mcp/servers показывает scheduler connected ✅ (scheduler в списке с toolsCount=6)
