package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/IfoRRR/vk-feed/internal/counter"
	"github.com/IfoRRR/vk-feed/internal/feed"
	"github.com/IfoRRR/vk-feed/internal/platform/httpx"
)

type Handler struct {
	feed    *feed.Service
	counter *counter.Service
	static  string
}

func New(feedSvc *feed.Service, counterSvc *counter.Service, staticDir string) *Handler {
	return &Handler{feed: feedSvc, counter: counterSvc, static: staticDir}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/v1/users", h.listUsers)
	mux.HandleFunc("GET /api/v1/feed", h.getFeed)
	mux.HandleFunc("POST /api/v1/posts", h.createPost)
	mux.HandleFunc("POST /api/v1/follow", h.follow)
	mux.HandleFunc("POST /api/v1/posts/{id}/like", h.like)
	mux.HandleFunc("GET /api/v1/posts/{id}/likes", h.likesCount)

	if h.static != "" {
		fs := http.FileServer(http.Dir(h.static))
		mux.Handle("GET /", fs)
		mux.Handle("GET /{$}", fs)
	}
	return mux
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.feed.Users(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "list users failed")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"users": users})
}

func (h *Handler) getFeed(w http.ResponseWriter, r *http.Request) {
	userID, err := queryInt64(r, "user_id", 1)
	if err != nil || userID <= 0 {
		httpx.Error(w, http.StatusBadRequest, "user_id required")
		return
	}
	limit, _ := queryInt64(r, "limit", 20)
	page, err := h.feed.Feed(r.Context(), userID, r.URL.Query().Get("cursor"), int(limit))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "feed failed")
		return
	}
	httpx.JSON(w, http.StatusOK, page)
}

type createPostReq struct {
	UserID int64  `json:"user_id"`
	Body   string `json:"body"`
}

func (h *Handler) createPost(w http.ResponseWriter, r *http.Request) {
	var req createPostReq
	if err := httpx.Decode(r, &req); err != nil || req.UserID <= 0 {
		httpx.Error(w, http.StatusBadRequest, "user_id required")
		return
	}
	req.Body = strings.TrimSpace(req.Body)
	if req.Body == "" || len(req.Body) > 500 {
		httpx.Error(w, http.StatusBadRequest, "body 1..500 chars")
		return
	}
	p, err := h.feed.CreatePost(r.Context(), req.UserID, req.Body)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, p)
}

type followReq struct {
	UserID   int64 `json:"user_id"`
	TargetID int64 `json:"target_id"`
}

func (h *Handler) follow(w http.ResponseWriter, r *http.Request) {
	var req followReq
	if err := httpx.Decode(r, &req); err != nil || req.UserID <= 0 || req.TargetID <= 0 {
		httpx.Error(w, http.StatusBadRequest, "user_id and target_id required")
		return
	}
	if err := h.feed.Follow(r.Context(), req.UserID, req.TargetID); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type likeReq struct {
	UserID int64 `json:"user_id"`
}

func (h *Handler) like(w http.ResponseWriter, r *http.Request) {
	postID, err := pathID(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid post id")
		return
	}
	var req likeReq
	if err := httpx.Decode(r, &req); err != nil || req.UserID <= 0 {
		httpx.Error(w, http.StatusBadRequest, "user_id required")
		return
	}
	res, err := h.counter.Like(r.Context(), postID, req.UserID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) likesCount(w http.ResponseWriter, r *http.Request) {
	postID, err := pathID(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid post id")
		return
	}
	n, err := h.counter.Count(r.Context(), postID)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "post not found")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"post_id": postID, "likes_count": n})
}

func pathID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

func queryInt64(r *http.Request, name string, fallback int64) (int64, error) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return fallback, nil
	}
	return strconv.ParseInt(v, 10, 64)
}
