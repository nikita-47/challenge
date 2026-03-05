# Task FSM Refactoring: Deterministic Phase Transitions

**Дата:** 2026-03-05

## Описание задачи

Рефакторинг Task State Machine: замена LLM-управляемых переходов на детерминированные, код-управляемые. Каждая фаза = отдельный агентский вызов с изолированным контекстом. Между фазами — автоматическая пауза, пользователь жмёт "Continue".

## Результаты Plan

Архитектура с 3 фазами (planning → executing → validating), каждая с изолированным system prompt и phase-specific тулами:
- `submit_plan` — завершает planning
- `report_step` — отчёт о шаге (не прерывает)
- `submit_validation` — завершает validating с passed/failed

## Что реализовано

### Backend (4 файла)

**`backend/taskstate.go`** — полностью переписан:
- Новый `TaskState` с `Paused bool`, `StepResults`, `Artifacts`, `Feedback`, `ValidationCount`
- Удалены: `applyAction()`, `taskStateAction`, `canTransition()`, `validTransitions`, `PhasePaused`, `SystemPromptSection()`
- Добавлены: `buildPlanningPrompt()`, `buildExecutingPrompt()`, `buildValidatingPrompt()` — изолированные промпты по фазам

**`backend/agent.go`** — рефакторинг:
- Удалены: `newAgentWithTaskState()`, `newAgentWithTaskStateFiltered()`, `taskStateTool()`, `TaskState` поле
- Добавлены: `PhaseResult`, `StepResults` поля, `newPlanningAgent()`, `newExecutingAgent()`, `newValidatingAgent()`
- 3 phase-specific тула: `submitPlanTool()`, `reportStepTool()`, `submitValidationTool()`
- `Run()`: submit_plan/submit_validation → PhaseResult + break; report_step → accumulate + emit

**`backend/server.go`** — оркестратор:
- Новая функция `runTaskPhase()` — switch по фазе, запуск соответствующего агента, обработка результата, переходы
- `handleChat()` — task mode вызывает `runTaskPhase()` напрямую, обработка Continue (paused → unpause)

**`backend/chat.go`** — CLI обновлён:
- `/task <goal>` → `runTaskPhase()` с новой TaskState
- `/resume` → проверка `Paused`, `runTaskPhase()`
- `cliAgentEmit`: обработка `step_result` event, убраны ссылки на `ExpectedAction`

### Frontend (5 файлов)

**`frontend/src/lib/types.ts`**: убран `'paused'` из `TaskPhase`, добавлены `StepResult`, `StepResultEvent`, обновлён `TaskState`

**`frontend/src/stores/chat.ts`**: `continueTask()` + `cancelTask()` вместо `pauseTask()` + `resumeTask()`, `isTaskContinue` логика в `sendMessage`

**`frontend/src/components/TaskStatePanel.vue`**: полная перезапись — Continue/Cancel кнопки, артефакты, feedback секция

**`frontend/src/components/ChatInput.vue`**: disabled при paused, placeholder "Task paused — click Continue above"

**`frontend/src/components/ChatInfoPanel.vue`**: убран `expected_action`, добавлены `paused`, `validation_count`, `step_results`

## Результаты Validation

- `go build` — OK
- `vue-tsc --noEmit` — OK (0 ошибок)
- UI: TaskStatePanel рендерится корректно с 4 фазами
- Cancel: очищает taskState, возвращает SendSettings
- Console: 0 ошибок
- Full flow (planning → executing → validating → done): **не проверен** — API credit balance insufficient

## Проблемы

- Полный E2E flow через Claude API не протестирован из-за недостатка кредитов. Компиляция, типы и UI-рендеринг валидны.

## Статус: Done (частично — без полного E2E)
