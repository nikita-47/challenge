# E2E Scenario: Инварианты при запросе к LLM

## Тест 1: UI инвариантов — все шаги пройдены ✅

## Тест 2: Local LLM (qwen 0.5B) — все шаги пройдены ✅

- [x] 1. Открыть чат, создать новую сессию, включить Task Mode, задать 2 инварианта ✅
- [x] 2. Отправить задачу и проверить что инварианты отображаются в TaskStatePanel ✅
- [x] 3. Проверить network-запросы: 3 POST /api/chat за полный цикл ✅
- [x] 4. LLM упоминает инварианты в ответах ✅
- [x] 5. Нет спама запросов ✅
- [x] 6. Консоль без ошибок ✅

## Тест 3: Claude API (prod) — ДО фикса sandbox

- [x] 1. Planning: Claude явно проверяет инварианты ✅ (920 in / 1 turn / $0.009)
- [x] 2. Executing: 5/5 шагов, инварианты проверены ✅ (15 turns / $0.17)
- [x] 3. Validating: FAIL — файлы не найдены из-за sandbox isolation ❌
- [x] 4. Re-executing + Re-validating: повторный FAIL, max turns ❌
- Итого: $0.39 / 39 turns — 50%+ потрачено впустую на sandbox-баг

## Тест 4: Claude API (prod) — ПОСЛЕ фикса sandbox ✅

- [x] 1. Planning: инварианты проверены, план из 5 шагов ✅ (920 in / 1 turn / $0.009)
- [x] 2. Executing: 5/5 шагов выполнены ✅ (13 turns / $0.16)
- [x] 3. Validating: PASS — файлы найдены сразу, код проверен ✅ (7 turns / $0.08)
- [x] 4. Done: задача завершена успешно ✅
- Итого: $0.24 / 20 turns — экономия 38% токенов vs до фикса

## Фикс: Shared sandbox между фазами task mode

Файлы: taskstate.go (+SandboxDir, EnsureSandbox, CleanupSandbox), agent.go (конструкторы принимают sandboxDir), server.go (убраны defer Cleanup, sandbox из TaskState), chat.go (cleanup при /task)
