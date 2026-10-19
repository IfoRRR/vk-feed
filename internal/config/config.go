package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr     string
	DatabaseURL  string
	RedisAddr    string
	FeedCacheTTL time.Duration
}

func Load() Config {
	return Config{
		HTTPAddr:     getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:  getenv("DATABASE_URL", "postgres://portfolio:portfolio@127.0.0.1:5432/feed?sslmode=disable"),
		RedisAddr:    getenv("REDIS_ADDR", "127.0.0.1:6379"),
		FeedCacheTTL: time.Duration(getenvInt("FEED_CACHE_TTL_SEC", 15)) * time.Second,
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
