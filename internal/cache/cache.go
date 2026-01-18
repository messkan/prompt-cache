package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/messkan/PromptCache/internal/logging"
	"github.com/messkan/PromptCache/internal/storage"
)

// Config holds cache configuration
type Config struct {
	TTL             time.Duration
	MaxEntries      int
	CleanupInterval time.Duration
}

// DefaultConfig returns default cache configuration
func DefaultConfig() *Config {
	return &Config{
		TTL:             24 * time.Hour,
		MaxEntries:      100000,
		CleanupInterval: 1 * time.Hour,
	}
}

type Cache struct {
	store           storage.Storage
	config          *Config
	stopCleanup     chan struct{}
	cleanupDone     chan struct{}
	mu              sync.RWMutex
	accessOrder     []string // LRU tracking
	accessOrderMap  map[string]int
}

type CacheItem struct {
	Response  []byte        `json:"response"`
	CreatedAt time.Time     `json:"created_at"`
	TTL       time.Duration `json:"ttl"`
}

func NewCache(store storage.Storage) *Cache {
	return NewCacheWithConfig(store, DefaultConfig())
}

func NewCacheWithConfig(store storage.Storage, config *Config) *Cache {
	if config == nil {
		config = DefaultConfig()
	}

	c := &Cache{
		store:          store,
		config:         config,
		stopCleanup:    make(chan struct{}),
		cleanupDone:    make(chan struct{}),
		accessOrder:    make([]string, 0),
		accessOrderMap: make(map[string]int),
	}

	// Start background cleanup goroutine
	go c.cleanupLoop()

	return c
}

func GenerateKey(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])
}

func (c *Cache) Set(ctx context.Context, key string, response []byte, ttl time.Duration) error {
	if ttl == 0 {
		ttl = c.config.TTL
	}

	item := CacheItem{
		Response:  response,
		CreatedAt: time.Now(),
		TTL:       ttl,
	}

	data, err := json.Marshal(item)
	if err != nil {
		return err
	}

	// Check if we need to evict entries
	c.mu.Lock()
	if len(c.accessOrder) >= c.config.MaxEntries {
		// Evict oldest entry (LRU)
		if len(c.accessOrder) > 0 {
			oldestKey := c.accessOrder[0]
			c.accessOrder = c.accessOrder[1:]
			delete(c.accessOrderMap, oldestKey)
			
			// Delete from storage
			go func(k string) {
				if err := c.store.Delete(context.Background(), k); err != nil {
					logging.Debug().Str("key", k).Err(err).Msg("Failed to delete evicted cache entry")
				}
			}(oldestKey)
		}
	}

	// Update access order
	if idx, exists := c.accessOrderMap[key]; exists {
		// Move to end
		c.accessOrder = append(c.accessOrder[:idx], c.accessOrder[idx+1:]...)
	}
	c.accessOrder = append(c.accessOrder, key)
	c.accessOrderMap[key] = len(c.accessOrder) - 1
	c.mu.Unlock()

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

	// Check TTL
	if item.TTL != 0 && time.Since(item.CreatedAt) > item.TTL {
		// Async delete expired entry
		go func(k string) {
			if err := c.store.Delete(context.Background(), k); err != nil {
				logging.Debug().Str("key", k).Err(err).Msg("Failed to delete expired cache entry")
			}
		}(key)
		return nil, false, nil
	}

	// Update access order (move to end for LRU)
	c.mu.Lock()
	if idx, exists := c.accessOrderMap[key]; exists {
		c.accessOrder = append(c.accessOrder[:idx], c.accessOrder[idx+1:]...)
	}
	c.accessOrder = append(c.accessOrder, key)
	c.accessOrderMap[key] = len(c.accessOrder) - 1
	c.mu.Unlock()

	return item.Response, true, nil
}

// Delete removes an entry from the cache
func (c *Cache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	if idx, exists := c.accessOrderMap[key]; exists {
		c.accessOrder = append(c.accessOrder[:idx], c.accessOrder[idx+1:]...)
		delete(c.accessOrderMap, key)
	}
	c.mu.Unlock()

	return c.store.Delete(ctx, key)
}

// Clear removes all entries from the cache
func (c *Cache) Clear(ctx context.Context) error {
	c.mu.Lock()
	c.accessOrder = make([]string, 0)
	c.accessOrderMap = make(map[string]int)
	c.mu.Unlock()

	// Delete all cache entries from storage
	badgerStore, ok := c.store.(*storage.BadgerStore)
	if ok {
		_, err := badgerStore.DeleteByPrefix(ctx, "")
		return err
	}
	return nil
}

// Count returns the number of entries in the cache
func (c *Cache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.accessOrder)
}

// GetConfig returns the cache configuration
func (c *Cache) GetConfig() *Config {
	return c.config
}

// cleanupLoop runs periodic cleanup of expired entries
func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()
	defer close(c.cleanupDone)

	for {
		select {
		case <-ticker.C:
			c.cleanupExpired()
		case <-c.stopCleanup:
			return
		}
	}
}

// cleanupExpired removes expired entries from storage
func (c *Cache) cleanupExpired() {
	ctx := context.Background()
	
	badgerStore, ok := c.store.(*storage.BadgerStore)
	if !ok {
		return
	}

	// Get all cache keys
	keys, err := badgerStore.GetAllKeys(ctx, "")
	if err != nil {
		logging.Error().Err(err).Msg("Failed to get cache keys for cleanup")
		return
	}

	expiredCount := 0
	for _, key := range keys {
		// Skip non-cache keys (embeddings, prompts)
		if len(key) > 4 && (key[:4] == "emb:" || key[:7] == "prompt:") {
			continue
		}

		data, err := c.store.Get(ctx, key)
		if err != nil {
			continue
		}

		var item CacheItem
		if err := json.Unmarshal(data, &item); err != nil {
			continue
		}

		if item.TTL != 0 && time.Since(item.CreatedAt) > item.TTL {
			if err := c.store.Delete(ctx, key); err == nil {
				expiredCount++
				
				c.mu.Lock()
				if idx, exists := c.accessOrderMap[key]; exists {
					c.accessOrder = append(c.accessOrder[:idx], c.accessOrder[idx+1:]...)
					delete(c.accessOrderMap, key)
				}
				c.mu.Unlock()
			}
		}
	}

	if expiredCount > 0 {
		logging.Info().Int("count", expiredCount).Msg("Cleaned up expired cache entries")
	}
}

// Stop stops the background cleanup goroutine
func (c *Cache) Stop() {
	close(c.stopCleanup)
	<-c.cleanupDone
}
