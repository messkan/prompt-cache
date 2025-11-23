package cache

import (
	"context"
	"time"

	"github.com/messkan/PromptCache/internal/storage"
)

type Cache struct {
	store storage.Storage
}

func NewCache(store storage.Storage) *Cache {
	return &Cache{store: store}
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	return c.store.Get(ctx, key)
}

func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.store.Set(ctx, key, value, ttl)
}

func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	val, err := c.store.Get(ctx, key)
	if err != nil {
		return false, err
	}
	return val != nil, nil
}
