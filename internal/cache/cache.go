package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/messkan/PromptCache/internal/storage"
)

type Cache struct {
	store storage.Storage
}

type CacheItem struct {
	Response  []byte        `json:"response"`
	CreatedAt time.Time     `json:"created_at"`
	TTL       time.Duration `json:"ttl"`
}

func NewCache(store storage.Storage) *Cache {
	return &Cache{store: store}
}

func GenerateKey(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])
}

func (c *Cache) Set(ctx context.Context, key string, response []byte, ttl time.Duration) error {
	item := CacheItem{
		Response:  response,
		CreatedAt: time.Now(),
		TTL:       ttl,
	}

	data, err := json.Marshal(item)
	if err != nil {
		return err
	}

	return c.store.Set(ctx, key, data)
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	data, err := c.store.Get(ctx, key)
	if err != nil {
		return nil, false, err
	}
	if data == nil {
		return nil, false, nil
	}

	var item CacheItem
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, false, err
	}

	if item.TTL > 0 && time.Since(item.CreatedAt) > item.TTL {
		return nil, false, nil
	}

	return item.Response, true, nil
}
