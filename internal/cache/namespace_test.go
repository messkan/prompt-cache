package cache

import (
	"context"
	"testing"
	"time"

	"github.com/messkan/PromptCache/internal/cachecontext"
	"github.com/messkan/PromptCache/internal/storage"
)

func TestCacheNamespaceIsolation(t *testing.T) {
	store, err := storage.NewBadgerStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewBadgerStore: %v", err)
	}
	defer store.Close()

	c := NewCacheWithConfig(store, &Config{TTL: time.Hour, MaxEntries: 100, CleanupInterval: time.Hour})
	defer c.Stop()

	ctxA := cachecontext.WithNamespace(context.Background(), "tenant-a")
	ctxB := cachecontext.WithNamespace(context.Background(), "tenant-b")
	key := GenerateKey("same prompt")

	if err := c.Set(ctxA, key, []byte("response-a"), time.Hour); err != nil {
		t.Fatalf("Set tenant A: %v", err)
	}
	if err := c.Set(ctxB, key, []byte("response-b"), time.Hour); err != nil {
		t.Fatalf("Set tenant B: %v", err)
	}

	gotA, found, err := c.Get(ctxA, key)
	if err != nil || !found || string(gotA) != "response-a" {
		t.Fatalf("tenant A got %q, found=%v, err=%v", gotA, found, err)
	}
	gotB, found, err := c.Get(ctxB, key)
	if err != nil || !found || string(gotB) != "response-b" {
		t.Fatalf("tenant B got %q, found=%v, err=%v", gotB, found, err)
	}

	if _, found, err := c.Get(context.Background(), key); err != nil || found {
		t.Fatalf("default namespace unexpectedly saw namespaced entry: found=%v err=%v", found, err)
	}
}
