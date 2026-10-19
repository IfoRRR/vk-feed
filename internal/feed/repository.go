package feed

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/IfoRRR/vk-feed/internal/model"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ListUsers(ctx context.Context) ([]model.User, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, username, display_name FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *Repository) CreatePost(ctx context.Context, authorID int64, body string) (model.Post, error) {
	var p model.Post
	err := r.pool.QueryRow(ctx, `
		INSERT INTO posts (author_id, body)
		VALUES ($1, $2)
		RETURNING id, author_id, body, likes_count, created_at
	`, authorID, body).Scan(&p.ID, &p.AuthorID, &p.Body, &p.LikesCount, &p.CreatedAt)
	if err != nil {
		return p, err
	}
	_ = r.pool.QueryRow(ctx, `SELECT username, display_name FROM users WHERE id=$1`, authorID).
		Scan(&p.AuthorUser, &p.AuthorName)
	return p, nil
}

func (r *Repository) Follow(ctx context.Context, userID, targetID int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO follows (user_id, target_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, userID, targetID)
	return err
}

func (r *Repository) Feed(ctx context.Context, userID int64, cursor string, limit int) ([]model.Post, string, error) {
	var (
		cursorTime time.Time
		cursorID   int64
		hasCursor  bool
	)
	if cursor != "" {
		parts := strings.SplitN(cursor, ",", 2)
		if len(parts) == 2 {
			if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
				if id, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					cursorTime, cursorID, hasCursor = t, id, true
				}
			}
		}
	}

	q := `
		SELECT p.id, p.author_id, u.username, u.display_name, p.body, p.likes_count, p.created_at,
		       EXISTS(SELECT 1 FROM likes l WHERE l.post_id = p.id AND l.user_id = $1) AS liked
		FROM posts p
		JOIN users u ON u.id = p.author_id
		WHERE (p.author_id = $1 OR p.author_id IN (
			SELECT target_id FROM follows WHERE user_id = $1
		))
	`
	args := []any{userID}
	if hasCursor {
		q += ` AND (p.created_at, p.id) < ($2::timestamptz, $3::bigint)`
		args = append(args, cursorTime, cursorID)
		q += ` ORDER BY p.created_at DESC, p.id DESC LIMIT $4`
		args = append(args, limit)
	} else {
		q += ` ORDER BY p.created_at DESC, p.id DESC LIMIT $2`
		args = append(args, limit)
	}

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var posts []model.Post
	for rows.Next() {
		var p model.Post
		if err := rows.Scan(&p.ID, &p.AuthorID, &p.AuthorUser, &p.AuthorName, &p.Body, &p.LikesCount, &p.CreatedAt, &p.LikedByMe); err != nil {
			return nil, "", err
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	var next string
	if len(posts) == limit {
		last := posts[len(posts)-1]
		next = fmt.Sprintf("%s,%d", last.CreatedAt.UTC().Format(time.RFC3339Nano), last.ID)
	}
	return posts, next, nil
}
