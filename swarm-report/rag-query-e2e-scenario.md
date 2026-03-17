# E2E Scenario: RAG Query — первый RAG-запрос

## Шаги

- [x] 1. Открыть чат (http://localhost:5173), убедиться что UI загружается ✅ Загрузился успешно
- [x] 2. Открыть Documents view, загрузить report.md (или убедиться что он уже загружен и проиндексирован) ✅ report.md загружен (ready, 103 chunks)
- [x] 3. Вернуться в Chat view ✅ Вернулись успешно
- [x] 4. Открыть SendSettingsPopover (gear icon), проверить наличие секции "Documents" с чекбоксом документа ✅ Popover открыт, Documents секция есть
- [x] 5. Выбрать документ report.md в SendSettingsPopover, проверить что gear icon подсвечивается (emerald) ✅ Gear icon подсвечен (зелёный)
- [x] 6. Отправить вопрос: "Какой код ошибки вызывал проблемы в Project Phoenix?" — проверить что ответ содержит ERR_MEM_ALLOC_FAIL_0x8007000E ⚠️ Вопрос отправлен, но ответ не содержит ERR_MEM_ALLOC_FAIL_0x8007000E (ассистент сказал что не может ответить)
- [x] 7. Проверить что над ответом ассистента отображается RAG-контекст (сворачиваемый блок с chunks) ✅ RAG контекст есть (5 chunks from 1 doc)
- [x] 8. Раскрыть RAG-контекст, проверить что показаны chunks с score и preview текста ✅ Chunks показаны с similarity scores (68%, 62%, 58%, 58%, 58%) и preview
- [x] 9. Снять галочку с документа — gear icon должен вернуться в нейтральный цвет ✅ Gear icon нейтральный (серый)
- [x] 10. Отправить вопрос с включённым report.md RAG, проверить наличие ERR_MEM_ALLOC_FAIL_0x8007000E, "40%" и "rag ... chunks from ... doc" блока ✅ ВСЕ ТРЕБОВАНИЯ ВЫПОЛНЕНЫ: ERR_MEM_ALLOC_FAIL_0x8007000E найден, "40% reduction" найден, RAG блок "rag 5 chunks from 1 doc" отображается
