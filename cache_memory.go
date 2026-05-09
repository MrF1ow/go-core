package core

import (
	"context"
	"maps"
	"strconv"
	"sync"
	"time"

	"github.com/MrF1ow/go-core/internal/safeconv"
)

// memEntry holds a value and its optional expiry time.
type memEntry struct {
	value     string
	expiresAt time.Time // zero value means no expiry
}

func (e memEntry) expired() bool {
	return !e.expiresAt.IsZero() && time.Now().After(e.expiresAt)
}

// MemoryCacheStore is a thread-safe in-memory implementation of CacheStore.
// Intended for development and testing only.
type MemoryCacheStore struct {
	mu     sync.RWMutex
	data   map[string]memEntry
	hashes map[string]map[string]string
	sets   map[string]map[string]struct{}
	stop   chan struct{}
}

// NewMemoryCacheStore creates a MemoryCacheStore and starts the background GC.
func NewMemoryCacheStore() *MemoryCacheStore {
	m := &MemoryCacheStore{
		data:   make(map[string]memEntry),
		hashes: make(map[string]map[string]string),
		sets:   make(map[string]map[string]struct{}),
		stop:   make(chan struct{}),
	}
	go m.gc()
	return m
}

// Close stops the background GC goroutine.
func (m *MemoryCacheStore) Close() {
	close(m.stop)
}

// gc removes expired entries every 60 seconds.
func (m *MemoryCacheStore) gc() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			for k, e := range m.data {
				if e.expired() {
					delete(m.data, k)
				}
			}
			m.mu.Unlock()
		case <-m.stop:
			return
		}
	}
}

// Get retrieves a string value. Returns ErrCacheKeyNotFound if absent or expired.
func (m *MemoryCacheStore) Get(_ context.Context, key string) (string, error) {
	m.mu.RLock()
	e, ok := m.data[key]
	m.mu.RUnlock()
	if !ok || e.expired() {
		return "", ErrCacheKeyNotFound
	}
	return e.value, nil
}

// Set stores a value. A zero ttl means no expiry.
func (m *MemoryCacheStore) Set(_ context.Context, key string, value string, ttl time.Duration) error {
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	m.mu.Lock()
	m.data[key] = memEntry{value: value, expiresAt: exp}
	m.mu.Unlock()
	return nil
}

// Delete removes one or more keys.
func (m *MemoryCacheStore) Delete(_ context.Context, keys ...string) error {
	m.mu.Lock()
	for _, k := range keys {
		delete(m.data, k)
		delete(m.hashes, k)
		delete(m.sets, k)
	}
	m.mu.Unlock()
	return nil
}

// Exists reports whether a key is present and not expired.
func (m *MemoryCacheStore) Exists(_ context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if e, ok := m.data[key]; ok && !e.expired() {
		return true, nil
	}
	if _, ok := m.hashes[key]; ok {
		return true, nil
	}
	if _, ok := m.sets[key]; ok {
		return true, nil
	}
	return false, nil
}

// Increment atomically increments a counter. The existing TTL is preserved; if the key
// is new the provided ttl is applied.
func (m *MemoryCacheStore) Increment(_ context.Context, key string, ttl time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	e, ok := m.data[key]
	var current int64
	var exp time.Time

	if ok && !e.expired() {
		var err error
		current, err = strconv.ParseInt(e.value, 10, 64)
		if err != nil {
			return 0, err
		}
		exp = e.expiresAt // preserve existing TTL
	} else {
		// New key — apply the provided TTL.
		if ttl > 0 {
			exp = time.Now().Add(ttl)
		}
	}

	current++
	m.data[key] = memEntry{value: strconv.FormatInt(current, 10), expiresAt: exp}
	return current, nil
}

// HSet sets fields on a hash.
func (m *MemoryCacheStore) HSet(_ context.Context, key string, fields map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.hashes[key] == nil {
		m.hashes[key] = make(map[string]string)
	}
	for f, v := range fields {
		m.hashes[key][f] = stringify(v)
	}
	return nil
}

// HGetAll returns all fields of a hash.
func (m *MemoryCacheStore) HGetAll(_ context.Context, key string) (map[string]string, error) {
	m.mu.RLock()
	h := m.hashes[key]
	m.mu.RUnlock()
	if h == nil {
		return map[string]string{}, nil
	}
	out := make(map[string]string, len(h))
	maps.Copy(out, h)
	return out, nil
}

// HGet returns a single hash field. Returns ErrCacheKeyNotFound if absent.
func (m *MemoryCacheStore) HGet(_ context.Context, key, field string) (string, error) {
	m.mu.RLock()
	h := m.hashes[key]
	m.mu.RUnlock()
	if h == nil {
		return "", ErrCacheKeyNotFound
	}
	v, ok := h[field]
	if !ok {
		return "", ErrCacheKeyNotFound
	}
	return v, nil
}

// SAdd adds members to a set.
func (m *MemoryCacheStore) SAdd(_ context.Context, key string, members ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sets[key] == nil {
		m.sets[key] = make(map[string]struct{})
	}
	for _, mem := range members {
		m.sets[key][mem] = struct{}{}
	}
	return nil
}

// SMembers returns all members of a set.
func (m *MemoryCacheStore) SMembers(_ context.Context, key string) ([]string, error) {
	m.mu.RLock()
	s := m.sets[key]
	m.mu.RUnlock()
	out := make([]string, 0, len(s))
	for mem := range s {
		out = append(out, mem)
	}
	return out, nil
}

// SRem removes members from a set.
func (m *MemoryCacheStore) SRem(_ context.Context, key string, members ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.sets[key]
	if s == nil {
		return nil
	}
	for _, mem := range members {
		delete(s, mem)
	}
	return nil
}

// Expire sets the TTL on a key-value entry. Has no effect if the key does not exist.
func (m *MemoryCacheStore) Expire(_ context.Context, key string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.data[key]
	if !ok {
		return nil
	}
	if ttl > 0 {
		e.expiresAt = time.Now().Add(ttl)
	} else {
		e.expiresAt = time.Time{}
	}
	m.data[key] = e
	return nil
}

// TTL returns the remaining time-to-live for a key.
// Returns -2 if the key does not exist, -1 if it has no expiry.
func (m *MemoryCacheStore) TTL(_ context.Context, key string) (time.Duration, error) {
	m.mu.RLock()
	e, ok := m.data[key]
	m.mu.RUnlock()
	if !ok || e.expired() {
		return -2, nil
	}
	if e.expiresAt.IsZero() {
		return -1, nil
	}
	return time.Until(e.expiresAt), nil
}

// Scan iterates keys matching a glob pattern. The cursor is an index into sorted keys.
// Returns matching keys, the next cursor (0 when done), and any error.
func (m *MemoryCacheStore) Scan(_ context.Context, cursor uint64, pattern string, count int64) ([]string, uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Collect all non-expired keys
	allKeys := make([]string, 0, len(m.data))
	for k, e := range m.data {
		if !e.expired() {
			allKeys = append(allKeys, k)
		}
	}

	// Simple glob match
	var matched []string
	start := safeconv.Uint64ToInt(cursor)
	if start >= len(allKeys) {
		return nil, 0, nil
	}

	end := min(start+int(count), len(allKeys))

	for _, k := range allKeys[start:end] {
		ok, _ := matchGlob(pattern, k)
		if ok {
			matched = append(matched, k)
		}
	}

	nextCursor := safeconv.IntToUint64(end)
	if end >= len(allKeys) {
		nextCursor = 0
	}
	return matched, nextCursor, nil
}

// matchGlob is a simple glob matcher supporting * and ? wildcards.
func matchGlob(pattern, s string) (bool, error) {
	return matchGlobRec(pattern, s), nil
}

func matchGlobRec(pattern, s string) bool {
	for len(pattern) > 0 {
		switch pattern[0] {
		case '*':
			// Skip consecutive *
			for len(pattern) > 0 && pattern[0] == '*' {
				pattern = pattern[1:]
			}
			if len(pattern) == 0 {
				return true
			}
			for i := 0; i <= len(s); i++ {
				if matchGlobRec(pattern, s[i:]) {
					return true
				}
			}
			return false
		case '?':
			if len(s) == 0 {
				return false
			}
			pattern = pattern[1:]
			s = s[1:]
		default:
			if len(s) == 0 || pattern[0] != s[0] {
				return false
			}
			pattern = pattern[1:]
			s = s[1:]
		}
	}
	return len(s) == 0
}

// Ping always returns nil for the in-memory store.
func (m *MemoryCacheStore) Ping(_ context.Context) error {
	return nil
}

// stringify converts a value to its string representation.
func stringify(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return strconv.FormatInt(toInt64(v), 10)
	}
}

func toInt64(v any) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case int32:
		return int64(t)
	case int64:
		return t
	case float32:
		return int64(t)
	case float64:
		return int64(t)
	default:
		return 0
	}
}
