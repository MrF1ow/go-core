package core

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisCacheStore implements CacheStore using a Redis client.
type RedisCacheStore struct {
	client *redis.Client
}

// NewRedisCacheStore creates a Redis client from cfg, pings it, and returns the store.
func NewRedisCacheStore(cfg RedisConfig) (*RedisCacheStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &RedisCacheStore{client: client}, nil
}

// Get retrieves a string value by key. Returns ErrCacheKeyNotFound if key is absent.
func (r *RedisCacheStore) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrCacheKeyNotFound
	}
	return val, err
}

// Set stores a string value with a TTL.
func (r *RedisCacheStore) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// Delete removes one or more keys.
func (r *RedisCacheStore) Delete(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Exists reports whether a key is present.
func (r *RedisCacheStore) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, key).Result()
	return n > 0, err
}

// Increment atomically increments a counter. If the key is new (value becomes 1), the TTL is set.
func (r *RedisCacheStore) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	val, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if val == 1 {
		// New key — apply the expiry.
		if expErr := r.client.Expire(ctx, key, ttl).Err(); expErr != nil {
			return val, expErr
		}
	}
	return val, nil
}

// HSet sets multiple fields on a hash key.
func (r *RedisCacheStore) HSet(ctx context.Context, key string, fields map[string]any) error {
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return r.client.HSet(ctx, key, args...).Err()
}

// HGetAll returns all field-value pairs for a hash key.
func (r *RedisCacheStore) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.client.HGetAll(ctx, key).Result()
}

// HGet returns a single field from a hash. Returns ErrCacheKeyNotFound if absent.
func (r *RedisCacheStore) HGet(ctx context.Context, key, field string) (string, error) {
	val, err := r.client.HGet(ctx, key, field).Result()
	if err == redis.Nil {
		return "", ErrCacheKeyNotFound
	}
	return val, err
}

// SAdd adds members to a set.
func (r *RedisCacheStore) SAdd(ctx context.Context, key string, members ...string) error {
	args := make([]any, len(members))
	for i, m := range members {
		args[i] = m
	}
	return r.client.SAdd(ctx, key, args...).Err()
}

// SMembers returns all members of a set.
func (r *RedisCacheStore) SMembers(ctx context.Context, key string) ([]string, error) {
	return r.client.SMembers(ctx, key).Result()
}

// SRem removes members from a set.
func (r *RedisCacheStore) SRem(ctx context.Context, key string, members ...string) error {
	args := make([]any, len(members))
	for i, m := range members {
		args[i] = m
	}
	return r.client.SRem(ctx, key, args...).Err()
}

// Expire sets the TTL on an existing key.
func (r *RedisCacheStore) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return r.client.Expire(ctx, key, ttl).Err()
}

// TTL returns the remaining time-to-live for a key.
// Returns -2 if the key does not exist, -1 if it has no expiry.
func (r *RedisCacheStore) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

// Scan iterates keys matching a pattern. Returns matching keys, the next cursor, and any error.
func (r *RedisCacheStore) Scan(ctx context.Context, cursor uint64, pattern string, count int64) ([]string, uint64, error) {
	return r.client.Scan(ctx, cursor, pattern, count).Result()
}

// Client returns the underlying *redis.Client for Redis-specific operations
// such as PubSub that cannot be abstracted behind CacheStore.
func (r *RedisCacheStore) Client() *redis.Client {
	return r.client
}

// Ping checks the Redis connection.
func (r *RedisCacheStore) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
