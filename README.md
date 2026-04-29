# SberScribe

**Telegram-бот** для умного конспектирования голосовых сообщений и аудиофайлов.  
Транскрипция через SaluteSpeech + краткая выжимка и чат с GigaChat. Всё хранится в PostgreSQL с полнотекстовым поиском.

## ✨ Возможности
- Принимает голосовые и аудио-сообщения
- Автоматическая транскрипция + генерация заголовка и summary
- Поиск по всем записям (`/find`)
- Просмотр списка и деталей записей (`/list`, `/get`)
- Чат с GigaChat (`/chat`)
- Асинхронная обработка без блокировок пользователя

## 🚀 Технические сильные стороны

- **Надёжная Postgres-очередь** — state machine на `status` (even/odd) + `FOR UPDATE SKIP LOCKED`. Масштабируется горизонтально без внешних брокеров.
- **Чистая layered архитектура** — bot → channels → service workers → repository. Полная изоляция слоёв.
- **TokenManager** с кэшированием для OAuth-токенов Sber.
- **Graceful shutdown** через `context` + `errgroup` + корректный release stale-задач.
- **Полнотекстовый поиск** на `TSVECTOR` + `GIN`-индекс с `websearch_to_tsquery`.
- **Кастомный TLS** + собственный CA для безопасного подключения к закрытым API Сбера.
- **Retry-механизм** и защита от зацикливания через `attempts` + автоматическое удаление «мёртвых» записей.
- Минимальные зависимости, современный Go (pgx/v5, telebot, slog).

## 🛠 Стек
- **Go** + `pgx/v5`, `telebot.v3`
- **PostgreSQL** 15+ (миграции через golang-migrate)
- **SaluteSpeech** + **GigaChat** (gRPC)
- **Docker-ready** (один бинарник)

## Быстрый старт

```bash
cp .env.example .env
# заполните TELEGRAM_TOKEN, SALUTE_SPEECH_CLIENT_SECRET, GIGA_CHAT_CLIENT_SECRET, CA_CERT_PATH и DATABASE_DSN

go run ./cmd/sberscribe