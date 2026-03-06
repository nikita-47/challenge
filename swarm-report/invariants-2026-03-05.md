# Invariants and State Constraints — 2026-03-05

## Описание задачи

День 14: добавить систему инвариантов — правил, которые ассистент не имеет права нарушать. Инварианты хранятся отдельно от диалога, учитываются в рассуждениях, ассистент отказывается предлагать решения нарушающие их.

## Итоги Research

**Go-архитектор:** исследовал buildFullSystemPrompt(), chatRequest, sessionSettings, TaskState, memory CRUD. Определил точки интеграции: TaskState для хранения, phase prompts для инжекции, chatRequest для передачи с фронта.

**Frontend-эксперт:** исследовал SendSettingsPopover (task mode + tools), ChatInput, chat store, TaskStatePanel, ChatInfoPanel. Определил UI: inline input в popover, отображение в панелях состояния.

## План реализации

Инварианты — массив текстовых строк, задаваемых per-task через SendSettingsPopover. Хранятся в TaskState (персистентно в рамках задачи через JSON-сериализацию сессии). Инжектируются в system prompt всех фаз с жёсткой формулировкой отказа при нарушении.

## Что реализовано

### Backend (Go)
- `backend/taskstate.go`: поле `Invariants []string` в TaskState, функция `formatInvariantsBlock()`, инжекция в 6 prompt builders (planning, executing, validating + local варианты)
- `backend/server.go`: поле `Invariants` в chatRequest, применение при создании TaskState в handleChat()

### Frontend (Vue/TS)
- `frontend/src/lib/types.ts`: `invariants?: string[]` в TaskState
- `frontend/src/stores/chat.ts`: расширение startTask() и sendMessage() body
- `frontend/src/components/SendSettingsPopover.vue`: UI ввода инвариантов (список + inline input + add/remove)
- `frontend/src/components/ChatInput.vue`: ref invariants, проброс в popover и store
- `frontend/src/components/TaskStatePanel.vue`: отображение инвариантов с ! иконкой
- `frontend/src/components/ChatInfoPanel.vue`: отображение в секции Task

## Результаты Validation

10/10 E2E шагов пройдены:
- Popover открывается, Task mode показывает секцию Invariants
- Ввод, добавление, удаление инвариантов работает
- Отправка задачи с инвариантами активирует task mode
- TaskStatePanel отображает инварианты
- ChatInfoPanel отображает инварианты
- Нет ошибок в консоли, все API вызовы 200 OK

## Проблемы и откаты

Нет.

## Статус: Done
