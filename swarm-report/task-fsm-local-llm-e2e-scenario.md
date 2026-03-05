# E2E Scenario: Task FSM with Local LLM

## Предусловия
- Provider переключён на "local" (LM Studio, qwen2.5-0.5b-instruct-mlx)
- Dev servers running (Go :8080, Vite :5173, LM Studio :1234)

## Шаги

- [x] 1. Открыть http://localhost:5173, убедиться что UI загружается без ошибок ✅ PASS
- [x] 2. Проверить что SendSettings popover видна и содержит task mode toggle ✅ PASS
- [x] 3. Включить task mode, ввести goal "List files in the current directory", отправить ✅ PASS
- [x] 4. Дождаться ответа от LLM — TaskStatePanel показывает planning фазу, затем переход в executing (paused=true) с планом ✅ PASS
- [x] 5. Нажать Continue — executing фаза запускается, степы обновляются ✅ PASS (все 3 шага выполнены, счётчик 3/3)
- [x] 6. Дождаться завершения executing — переход в validating (paused=true) ✅ PASS
- [x] 7. Нажать Continue — validating фаза запускается ✅ PASS
- [x] 8. Дождаться завершения — таск переходит в done (TaskStatePanel скрывается) ✅ PASS (TaskStatePanel скрыта)
- [x] 9. Проверить консоль браузера на отсутствие ошибок ✅ PASS (0 ошибок)
- [x] 10. Проверить network requests на отсутствие failed ✅ PASS (все 200 OK, 3 POST запроса к /api/chat)
