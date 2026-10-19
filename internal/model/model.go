package model

import "time"

type User struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type Post struct {
	ID          int64     `json:"id"`
	AuthorID    int64     `json:"author_id"`
	AuthorName  string    `json:"author_name"`
	AuthorUser  string    `json:"author_username"`
	Body        string    `json:"body"`
	LikesCount  int64     `json:"likes_count"`
	LikedByMe   bool      `json:"liked_by_me"`
	CreatedAt   time.Time `json:"created_at"`
}

type FeedPage struct {
	Posts      []Post `json:"posts"`
	NextCursor string `json:"next_cursor,omitempty"`
	Cached     bool   `json:"cached"`
}

type LikeResult struct {
	PostID     int64 `json:"post_id"`
	LikesCount int64 `json:"likes_count"`
	Created    bool  `json:"created"` // false = идемпотентный повтор
}
