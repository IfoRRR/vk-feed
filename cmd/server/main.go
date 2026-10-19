package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IfoRRR/vk-feed/internal/api"
	"github.com/IfoRRR/vk-feed/internal/config"
	"github.com/IfoRRR/vk-feed/internal/counter"
	"github.com/IfoRRR/vk-feed/internal/feed"
	"github.com/IfoRRR/vk-feed/internal/platform/postgres"
	redisx "github.com/IfoRRR/vk-feed/internal/platform/redis"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := postgres.EnsureDatabase(ctx, cfg.DatabaseURL); err != nil {
		slog.Error("ensure database", "err", err)
		os.Exit(1)
	}

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := postgres.MigrateDir(ctx, pool, "migrations"); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}

	rdb, err := redisx.NewClient(ctx, cfg.RedisAddr)
	if err != nil {
		slog.Error("redis", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()

	feedSvc := feed.NewService(feed.NewRepository(pool), rdb, cfg.FeedCacheTTL)
	counterSvc := counter.New(pool, rdb)
	handler := api.New(feedSvc, counterSvc, "static")

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      handler.Routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	slog.Info("stopped")
}
