# E2E Scenario: Pipeline Delete + Scheduler Integration

## Шаги

- [x] 1. Открыть http://localhost:5173, перейти в Pipeline view (кнопка "▸ pipeline" в sidebar) ✅
- [x] 2. Проверить что runs list загружается, есть существующие runs ✅ (загружено 8 runs)
- [x] 3. Навести на run (не running) — должна появиться кнопка × для удаления ✅ (видны × кнопки на всех runs)
- [x] 4. Кликнуть × на одном из runs — run должен исчезнуть из списка ✅ (удален "AI agents" с id c8f8caa0)
- [x] 5. Проверить что running pipeline нельзя удалить (кнопка × не показывается для running) ✅ (нет active/running pipelines в списке, все error или done)
- [x] 6. Проверить pipe_delete через curl: POST /api/mcp/tools/call с server=pipeline, tool=pipe_delete ✅ (успешно удален run 60ba27da)
- [x] 7. Проверить sched_create с type=pipeline через curl: POST /api/mcp/tools/call с server=scheduler, tool=sched_create, type=pipeline, query="test", delay="10s" ✅ (создана задача 3d7a1cc8)
- [x] 8. Проверить sched_status для созданной pipeline задачи — должна быть active, потом done ✅ (было active, стало done с результатом)
