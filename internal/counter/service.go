package counter

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"github.com/IfoRRR/vk-feed/internal/model"
)

// Service — кусок B: идемпотентные лайки + hot-счётчик в Redis.
type Service struct {
	pool  *pgxpool.Pool
	redis *goredis.Client
}

func New(pool *pgxpool.Pool, redis *goredis.Client) *Service {
	return &Service{pool: pool, redis: redis}
}

func likesKey(postID int64) string {
	return fmt.Sprintf("likes:p:%d", postID)
}

func (s *Service) Like(ctx context.Context, postID, userID int64) (model.LikeResult, error) {
	var res model.LikeResult
	res.PostID = postID

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		INSERT INTO likes (post_id, user_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, postID, userID)
	if err != nil {
		return res, err
	}
	res.Created = tag.RowsAffected() > 0

	if res.Created {
		if err := tx.QueryRow(ctx, `
			UPDATE posts SET likes_count = likes_count + 1
			WHERE id = $1
			RETURNING likes_count
		`, postID).Scan(&res.LikesCount); err != nil {
			return res, err
		}
	} else {
		if err := tx.QueryRow(ctx, `SELECT likes_count FROM posts WHERE id = $1`, postID).Scan(&res.LikesCount); err != nil {
			return res, fmt.Errorf("post not found")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return res, err
	}

	if res.Created {
		_ = s.redis.Incr(ctx, likesKey(postID)).Err()
	} else {
		// прогрев кэша актуальным значением
		_ = s.redis.Set(ctx, likesKey(postID), res.LikesCount, 0).Err()
	}
	return res, nil
}

func (s *Service) Count(ctx context.Context, postID int64) (int64, error) {
	if n, err := s.redis.Get(ctx, likesKey(postID)).Result(); err == nil {
		if v, err := strconv.ParseInt(n, 10, 64); err == nil {
			return v, nil
		}
	}
	var count int64
	if err := s.pool.QueryRow(ctx, `SELECT likes_count FROM posts WHERE id=$1`, postID).Scan(&count); err != nil {
		return 0, err
	}
	_ = s.redis.Set(ctx, likesKey(postID), count, 0).Err()
	return count, nil
}
