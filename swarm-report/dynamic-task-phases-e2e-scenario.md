# E2E Scenario: Dynamic Task Phases

## Шаги

- [x] 1. Открыть http://localhost:5173 — убедиться что приложение загружается ✅ (проверено)
- [x] 2. Включить task mode, отправить goal — убедиться что фаза "proposing" запускается ✅ (проверено, LLM вернул фазы, fallback-парсер сработал)
- [x] 3. Дождаться предложенных фаз — UI показывает предложенный pipeline с описаниями и кнопку "approve" ✅ (проверено: Plan→Execute→Validate, описания, кнопка approve)
- [x] 4. Нажать approve — фазы начинают выполняться последовательно (первая фаза становится active) ✅ (проверено: planning→executing, steps появились)
- [x] 5. Проверить что фазы переключаются — после завершения фазы она помечается completed, следующая становится active ✅ (проверено: planning→executing→validating, все completed)
- [x] 6. Проверить ChatInfoPanel — секция task показывает phase, pipeline с динамическими фазами ✅ (проверено: phase=done, pipeline с 3 фазами, progress=3/3)
- [x] 7. Проверить что task завершается корректно (phase = done) ✅ (проверено: все фазы completed, phase=done, validation PASS)
