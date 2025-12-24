package metrics

import (
	"context"
	"testing"
)

// MockStorage implements storage.Storage for testing
type MockStorage struct {
	embeddings map[string][]byte
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		embeddings: make(map[string][]byte),
	}
}

func (m *MockStorage) Set(ctx context.Context, key string, value []byte) error {
	m.embeddings[key] = value
	return nil
}

func (m *MockStorage) Get(ctx context.Context, key string) ([]byte, error) {
	val, ok := m.embeddings[key]
	if !ok {
		return nil, nil
	}
	return val, nil
}

func (m *MockStorage) Delete(ctx context.Context, key string) error {
	delete(m.embeddings, key)
	return nil
}

func (m *MockStorage) GetAllEmbeddings(ctx context.Context) (map[string][]byte, error) {
	return m.embeddings, nil
}

func (m *MockStorage) GetPrompt(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (m *MockStorage) CountEmbeddings(ctx context.Context) (int64, error) {
	var count int64
	for k := range m.embeddings {
		if len(k) >= 4 && k[:4] == "emb:" {
			count++
		}
	}
	return count, nil
}

func (m *MockStorage) Close() {}

func TestMetrics_IncrementCacheHits(t *testing.T) {
	store := NewMockStorage()
	m := NewMetrics(store)

	m.IncrementCacheHits()
	m.IncrementCacheHits()
	m.IncrementCacheHits()

	if m.cacheHits.Load() != 3 {
		t.Errorf("Expected 3 cache hits, got %d", m.cacheHits.Load())
	}
}

func TestMetrics_IncrementCacheMisses(t *testing.T) {
	store := NewMockStorage()
	m := NewMetrics(store)

	m.IncrementCacheMisses()
	m.IncrementCacheMisses()

	if m.cacheMisses.Load() != 2 {
		t.Errorf("Expected 2 cache misses, got %d", m.cacheMisses.Load())
	}
}

func TestMetrics_IncrementEvictionCount(t *testing.T) {
	store := NewMockStorage()
	m := NewMetrics(store)

	m.IncrementEvictionCount()

	if m.evictionCount.Load() != 1 {
		t.Errorf("Expected 1 eviction, got %d", m.evictionCount.Load())
	}
}

func TestMetrics_GetMetrics(t *testing.T) {
	store := NewMockStorage()
	m := NewMetrics(store)
	ctx := context.Background()

	// Add some embeddings to the mock storage
	store.Set(ctx, "emb:key1", []byte("embedding1"))
	store.Set(ctx, "emb:key2", []byte("embedding2"))
	store.Set(ctx, "prompt:key1", []byte("prompt1"))

	// Increment counters
	m.IncrementCacheHits()
	m.IncrementCacheHits()
	m.IncrementCacheHits()
	m.IncrementCacheMisses()
	m.IncrementEvictionCount()

	// Get metrics
	metricsResp, err := m.GetMetrics(ctx)
	if err != nil {
		t.Fatalf("GetMetrics failed: %v", err)
	}

	// Verify metrics
	if metricsResp.CacheHits != 3 {
		t.Errorf("Expected 3 cache hits, got %d", metricsResp.CacheHits)
	}

	if metricsResp.CacheMisses != 1 {
		t.Errorf("Expected 1 cache miss, got %d", metricsResp.CacheMisses)
	}

	if metricsResp.EvictionCount != 1 {
		t.Errorf("Expected 1 eviction, got %d", metricsResp.EvictionCount)
	}

	if metricsResp.StoredVectorsCount != 2 {
		t.Errorf("Expected 2 stored vectors, got %d", metricsResp.StoredVectorsCount)
	}

	expectedHitRate := 3.0 / 4.0 // 3 hits out of 4 total (3 hits + 1 miss)
	if metricsResp.HitRate != expectedHitRate {
		t.Errorf("Expected hit rate %f, got %f", expectedHitRate, metricsResp.HitRate)
	}
}

func TestMetrics_GetMetrics_NoActivity(t *testing.T) {
	store := NewMockStorage()
	m := NewMetrics(store)
	ctx := context.Background()

	// Get metrics without any activity
	metricsResp, err := m.GetMetrics(ctx)
	if err != nil {
		t.Fatalf("GetMetrics failed: %v", err)
	}

	// Verify all metrics are zero
	if metricsResp.CacheHits != 0 {
		t.Errorf("Expected 0 cache hits, got %d", metricsResp.CacheHits)
	}

	if metricsResp.CacheMisses != 0 {
		t.Errorf("Expected 0 cache misses, got %d", metricsResp.CacheMisses)
	}

	if metricsResp.EvictionCount != 0 {
		t.Errorf("Expected 0 evictions, got %d", metricsResp.EvictionCount)
	}

	if metricsResp.StoredVectorsCount != 0 {
		t.Errorf("Expected 0 stored vectors, got %d", metricsResp.StoredVectorsCount)
	}

	if metricsResp.HitRate != 0.0 {
		t.Errorf("Expected hit rate 0.0, got %f", metricsResp.HitRate)
	}
}
