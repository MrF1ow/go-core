package core

import (
	"context"
	"errors"
	"time"
)

// ErrCacheKeyNotFound is returned when a key does not exist.
var ErrCacheKeyNotFound = errors.New("cache: key not found")

// CacheStore abstracts key-value and hash storage for tokens, sessions, and rate limiting.
type CacheStore interface {
	// Key-value operations
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)

	// Counter operations (for rate limiting, brute force tracking)
	Increment(ctx context.Context, key string, ttl time.Duration) (int64, error)

	// Hash operations (for sessions)
	HSet(ctx context.Context, key string, fields map[string]any) error
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HGet(ctx context.Context, key, field string) (string, error)

	// Set operations (for session indexes)
	SAdd(ctx context.Context, key string, members ...string) error
	SMembers(ctx context.Context, key string) ([]string, error)
	SRem(ctx context.Context, key string, members ...string) error

	// TTL operations
	Expire(ctx context.Context, key string, ttl time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Key scanning (for background jobs like session expiry detection)
	Scan(ctx context.Context, cursor uint64, pattern string, count int64) (keys []string, nextCursor uint64, err error)

	// Ping for health checks
	Ping(ctx context.Context) error
}
