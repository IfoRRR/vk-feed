# vk-feed

Лента подписок с cursor-пагинацией и идемпотентными лайками.

Live: `/feed/` · API: `/api/feed/`

## Стек

Go · PostgreSQL · Redis · Vue (CDN)

## Что внутри

- лента: свои посты + подписки, cursor `(created_at,id)`
- лайки: `UNIQUE(post_id, user_id)` — повтор не накручивает
- Redis: hot-счётчик лайков + короткий кэш первой страницы ленты

## Запуск в monorepo

```bash
docker compose --profile heavy up -d --build feed nginx
```

Локально (нужны Postgres `feed` и Redis):

```bash
go run ./cmd/server
```
