# VK Feed

Backend-сервис социальной ленты с подписками, cursor-пагинацией и идемпотентными лайками.

**Live demo:** `/feed/`  
**API:** `/api/feed/`

## Что демонстрирует проект

- лента собственных постов и постов подписок;
- cursor-пагинация по `(created_at, id)` без пропусков и дублей;
- идемпотентные лайки через `UNIQUE(post_id, user_id)`;
- Redis-кэш первой страницы и hot-счётчики лайков;
- PostgreSQL для постоянного хранения данных;
- Docker Compose и Nginx для production-развёртывания.

## Архитектура

```text
Browser → Nginx → Go API → PostgreSQL
                    └────→ Redis
```

## Стек

- Go
- PostgreSQL
- Redis
- Vue через CDN
- Docker Compose
- Nginx

## Локальный запуск

Для запуска через общий monorepo:

```bash
docker compose --profile heavy up -d --build feed nginx
```

Локальный сервер API:

```bash
go run ./cmd/server
```

Для локального запуска нужны PostgreSQL и Redis. Секреты и настройки подключения задаются через переменные окружения; `.env` не хранится в репозитории.

## API

Основные маршруты:

| Метод | Путь | Назначение |
|---|---|---|
| `GET` | `/api/feed/health` | healthcheck сервиса |
| `GET` | `/api/feed/` | получение страницы ленты |
| `POST` | `/api/feed/posts/{id}/like` | поставить лайк |
| `DELETE` | `/api/feed/posts/{id}/like` | убрать лайк |

## Почему cursor pagination

Offset-пагинация нестабильна при добавлении новых постов: элементы могут дублироваться или пропускаться между запросами. Cursor `(created_at, id)` сохраняет стабильный порядок и лучше подходит для ленты.

## Что можно развить

- авторизацию и refresh tokens;
- комментарии и подписки через полноценные API;
- очередь событий для обновления счётчиков;
- OpenTelemetry и rate limiting;
- нагрузочные тесты ленты и лайков.

## Статус

Учебный pet-project для портфолио backend-разработчика. Проект не является официальным продуктом VK.
