package core

import (
	"context"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *MemoryCacheStore {
	t.Helper()
	m := NewMemoryCacheStore()
	t.Cleanup(m.Close)
	return m
}

func TestExists_StringKey(t *testing.T) {
	m := newTestStore(t)
	ctx := context.Background()

	ok, err := m.Exists(ctx, "k")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected false for missing key")
	}

	if err := m.Set(ctx, "k", "v", 0); err != nil {
		t.Fatal(err)
	}

	ok, err = m.Exists(ctx, "k")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected true for string key")
	}
}

func TestExists_HashKey(t *testing.T) {
	m := newTestStore(t)
	ctx := context.Background()

	if err := m.HSet(ctx, "h", map[string]any{"f1": "v1"}); err != nil {
		t.Fatal(err)
	}

	ok, err := m.Exists(ctx, "h")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected true for hash key")
	}
}

func TestExists_SetKey(t *testing.T) {
	m := newTestStore(t)
	ctx := context.Background()

	if err := m.SAdd(ctx, "s", "m1", "m2"); err != nil {
		t.Fatal(err)
	}

	ok, err := m.Exists(ctx, "s")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected true for set key")
	}
}

func TestExists_ExpiredStringKey(t *testing.T) {
	m := newTestStore(t)
	ctx := context.Background()

	if err := m.Set(ctx, "k", "v", time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)

	ok, err := m.Exists(ctx, "k")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected false for expired key")
	}
}

func TestDelete_StringKey(t *testing.T) {
	m := newTestStore(t)
	ctx := context.Background()

	if err := m.Set(ctx, "k", "v", 0); err != nil {
		t.Fatal(err)
	}
	if err := m.Delete(ctx, "k"); err != nil {
		t.Fatal(err)
	}

	ok, err := m.Exists(ctx, "k")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected false after delete")
	}
}

func TestDelete_HashKey(t *testing.T) {
	m := newTestStore(t)
	ctx := context.Background()

	if err := m.HSet(ctx, "h", map[string]any{"f1": "v1"}); err != nil {
		t.Fatal(err)
	}
	if err := m.Delete(ctx, "h"); err != nil {
		t.Fatal(err)
	}

	ok, err := m.Exists(ctx, "h")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected false after delete")
	}

	got, err := m.HGetAll(ctx, "h")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty hash after delete, got %v", got)
	}
}

func TestDelete_SetKey(t *testing.T) {
	m := newTestStore(t)
	ctx := context.Background()

	if err := m.SAdd(ctx, "s", "m1"); err != nil {
		t.Fatal(err)
	}
	if err := m.Delete(ctx, "s"); err != nil {
		t.Fatal(err)
	}

	ok, err := m.Exists(ctx, "s")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected false after delete")
	}

	members, err := m.SMembers(ctx, "s")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 0 {
		t.Fatalf("expected empty set after delete, got %v", members)
	}
}

func TestDelete_MultipleKeyTypes(t *testing.T) {
	m := newTestStore(t)
	ctx := context.Background()

	if err := m.Set(ctx, "str", "v", 0); err != nil {
		t.Fatal(err)
	}
	if err := m.HSet(ctx, "hash", map[string]any{"f": "v"}); err != nil {
		t.Fatal(err)
	}
	if err := m.SAdd(ctx, "set", "m"); err != nil {
		t.Fatal(err)
	}

	if err := m.Delete(ctx, "str", "hash", "set"); err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"str", "hash", "set"} {
		ok, err := m.Exists(ctx, key)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Fatalf("expected %q gone after bulk delete", key)
		}
	}
}
