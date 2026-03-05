# E2E Scenario: Task FSM Refactoring — Deterministic Phase Transitions

## Шаги

- [x] 1. Открыть http://localhost:5173, убедиться что UI загружается без ошибок ✅ (проверено)
- [x] 2. Проверить что Send Settings Popover с task mode toggle видна (до активации таска) ✅ (проверено)
- [x] 3. Включить task mode, ввести goal "List files in the current directory", отправить — увидеть TaskStatePanel с фазой planning ✅ (TaskStatePanel появился с корректными фазами, cancel кнопкой)
- [ ] 4. Дождаться завершения planning фазы — увидеть план (steps) и кнопку "Continue" (phase=executing, paused=true) ⚠️ API credit error — невозможно проверить
- [ ] 5. Нажать Continue — executing фаза запускается, степы обновляются ⚠️ API credit error
- [ ] 6. Дождаться завершения executing — увидеть кнопку "Continue" (phase=validating, paused=true) ⚠️ API credit error
- [ ] 7. Нажать Continue — validating фаза запускается ⚠️ API credit error
- [ ] 8. Дождаться завершения — таск переходит в done или откат ⚠️ API credit error
- [x] 9. Проверить что кнопка Cancel работает (начать новый таск, нажать Cancel — taskState очищается) ✅ (проверено)
- [x] 10. Проверить что ChatInput: SendSettings скрывается при активном таске, возвращается после cancel ✅ (проверено)
- [x] 11. Проверить консоль браузера на отсутствие ошибок ✅ (0 errors)
