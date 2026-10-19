package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/IfoRRR/vk-feed/internal/model"
)

type Service struct {
	repo  *Repository
	redis *goredis.Client
	ttl   time.Duration
}

func NewService(repo *Repository, redis *goredis.Client, ttl time.Duration) *Service {
	return &Service{repo: repo, redis: redis, ttl: ttl}
}

func (s *Service) Users(ctx context.Context) ([]model.User, error) {
	return s.repo.ListUsers(ctx)
}

func (s *Service) CreatePost(ctx context.Context, authorID int64, body string) (model.Post, error) {
	p, err := s.repo.CreatePost(ctx, authorID, body)
	if err != nil {
		return p, err
	}
	s.bustFeedCaches(ctx)
	return p, nil
}

func (s *Service) Follow(ctx context.Context, userID, targetID int64) error {
	if userID == targetID {
		return fmt.Errorf("cannot follow yourself")
	}
	if err := s.repo.Follow(ctx, userID, targetID); err != nil {
		return err
	}
	s.bustFeedCaches(ctx)
	return nil
}

func (s *Service) Feed(ctx context.Context, userID int64, cursor string, limit int) (model.FeedPage, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	cacheKey := ""
	if cursor == "" {
		cacheKey = fmt.Sprintf("feed:u:%d:l:%d", userID, limit)
		if raw, err := s.redis.Get(ctx, cacheKey).Bytes(); err == nil {
			var page model.FeedPage
			if json.Unmarshal(raw, &page) == nil {
				page.Cached = true
				return page, nil
			}
		}
	}

	posts, next, err := s.repo.Feed(ctx, userID, cursor, limit)
	if err != nil {
		return model.FeedPage{}, err
	}
	if posts == nil {
		posts = []model.Post{}
	}
	page := model.FeedPage{Posts: posts, NextCursor: next, Cached: false}
	if cacheKey != "" {
		if b, err := json.Marshal(page); err == nil {
			_ = s.redis.Set(ctx, cacheKey, b, s.ttl).Err()
		}
	}
	return page, nil
}

func (s *Service) bustFeedCaches(ctx context.Context) {
	iter := s.redis.Scan(ctx, 0, "feed:u:*", 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if len(keys) > 0 {
		_ = s.redis.Del(ctx, keys...).Err()
	}
}
