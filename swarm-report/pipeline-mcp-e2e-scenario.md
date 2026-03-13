# E2E Scenario: MCP Pipeline Composition

## Шаги

- [x] 1. Pipeline MCP сервер запускается и подключается (3/3 серверов connected, 6 tools)
- [x] 2. pipe_run запускает async pipeline, возвращает ID сразу
- [x] 3. search шаг — HN Algolia API, результаты получены (3 hits, ~0.7s)
- [x] 4. summarize шаг — Claude Haiku 4.5 API, summary сгенерирован (~2.8s)
- [x] 5. save шаг — файл pipeline_output/<id>.md создан (~2ms)
- [x] 6. pipe_status возвращает JSON со всеми шагами и статусами
- [x] 7. pipe_list возвращает JSON массив всех runs
- [x] 8. Frontend: таб "pipeline" добавлен в левый сайдбар
- [x] 9. Frontend: PipelinePanel с input, flow-схемой, списком runs
- [x] 10. Frontend: TypeScript компилируется без ошибок
- [x] 11. Frontend: pipeline store с polling каждые 2s
