# Changelog

All notable changes to this project will be documented in this file.

---

## [Day 33] — Support Assistant

### ✨ New Features

#### 🧩 MCP Server: Tickets (`mcp-servers/tickets`)
- Новый MCP-сервер для управления тикетами поддержки.
- Инструменты:
  - `ticket_create` — создание нового тикета с заголовком, описанием, приоритетом и email пользователя.
  - `ticket_list` — список тикетов с фильтрацией по статусу (`open` / `closed` / `all`).
  - `ticket_get` — получение тикета по ID вместе с полной историей сообщений.
  - `ticket_close` — закрытие тикета с опциональной резолюцией.
  - `ticket_add_message` — добавление сообщения в историю тикета (роли: `user` / `assistant`).
- Хранилище тикетов (`store.go`) с персистентностью в JSON-файл, поддержкой приоритетов (`low` / `medium` / `high`) и статусов.

#### 💬 Support Chat Backend (`backend/support.go`)
- Новый эндпоинт `POST /api/support/chat` — чат с ассистентом поддержки.
- RAG-обогащение ответов: автоматический поиск по FAQ-документам (файлы с префиксом `faq-`).
- Интеграция с MCP-сервером `tickets`: при передаче `ticketId` контекст тикета подставляется в системный промпт.
- Поддержка истории диалога (последние 10 сообщений).
- Системный промпт настроен на дружелюбный стиль, ответы на языке пользователя, отказ от выдумывания информации.

#### 🪟 Floating Support Widget (`frontend/src/components/SupportWidget.vue`)
- Новый плавающий виджет чата поддержки, доступный из любого экрана приложения (`App.vue`).
- Рендеринг ответов ассистента в Markdown (через `marked`).
- Автоскролл к последнему сообщению.
- Авторесайз поля ввода (до 80px).
- Управление состоянием через Pinia-стор `useSupportStore`.

---

## [Day 32] — Code Review Automation

### ✨ New Features

#### 🔍 AI Code Review Pipeline (`backend/review.go`)
- Новый модуль автоматического ревью Pull Request'ов через GitHub CLI (`gh`).
- Пайплайн из трёх шагов с SSE-стримингом прогресса:
  1. **diff** — получение git-диффа PR (`gh pr diff`), с обрезкой до 100 КБ.
  2. **rag** — RAG-обогащение контекста: поиск релевантных чанков по всем готовым документам через эмбеддинги Ollama (порог 0.3, топ-5 на документ).
  3. **analyze** — стриминг ревью от Claude с разбивкой на секции: Summary, Potential Bugs, Code Quality, Security, Performance, Recommendations.
- Автоматическая публикация результата ревью как комментария к PR (`gh pr comment`).
- Эндпоинты:
  - `GET /api/review/prs` — список открытых PR с метаданными (номер, заголовок, автор, ветка, URL, лейблы).
  - `POST /api/review/run` — запуск ревью для выбранного PR (SSE-стрим).

#### 🖥️ Review UI (`frontend/src/components/ReviewView.vue`)
- Новый экран **Code Review** в боковой панели навигации.
- Список открытых PR с кнопкой запуска ревью.
- Визуализация прогресса пайплайна по шагам (diff → rag → analyze).
- Рендеринг итогового ревью в Markdown.

---

*Формат основан на [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).*
