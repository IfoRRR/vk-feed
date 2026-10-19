CREATE TABLE IF NOT EXISTS users (
    id          BIGSERIAL PRIMARY KEY,
    username    TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS follows (
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_id   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, target_id),
    CHECK (user_id <> target_id)
);

CREATE TABLE IF NOT EXISTS posts (
    id          BIGSERIAL PRIMARY KEY,
    author_id   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body        TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 500),
    likes_count BIGINT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_posts_feed ON posts (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_posts_author ON posts (author_id, created_at DESC);

-- Идемпотентность лайка: один пользователь — один лайк на пост.
CREATE TABLE IF NOT EXISTS likes (
    post_id     BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (post_id, user_id)
);

INSERT INTO users (id, username, display_name) VALUES
    (1, 'artem', 'Артём'),
    (2, 'lena', 'Лена'),
    (3, 'max', 'Макс')
ON CONFLICT (username) DO NOTHING;

SELECT setval('users_id_seq', (SELECT MAX(id) FROM users));

INSERT INTO follows (user_id, target_id) VALUES
    (1, 2), (1, 3), (2, 1)
ON CONFLICT DO NOTHING;

INSERT INTO posts (author_id, body, likes_count, created_at) VALUES
    (2, 'Собрала ленту: подписки + cursor-пагинация. Классика соц. бэка.', 2, NOW() - INTERVAL '3 hours'),
    (3, 'Лайки через UNIQUE(post_id, user_id) — повторный клик не накручивает счётчик.', 1, NOW() - INTERVAL '2 hours'),
    (1, 'Redis держит hot-счётчик likes; Postgres — source of truth.', 0, NOW() - INTERVAL '1 hour'),
    (2, 'Для VK-стажировки важнее паттерны, чем «ещё один CRUD».', 1, NOW() - INTERVAL '30 minutes'),
    (3, 'Feed cache TTL короткий: свежесть важнее hit-rate на демо.', 0, NOW() - INTERVAL '10 minutes');

INSERT INTO likes (post_id, user_id)
SELECT p.id, u.id
FROM posts p
CROSS JOIN users u
WHERE p.body LIKE 'Собрала ленту%' AND u.id IN (1, 3)
ON CONFLICT DO NOTHING;
