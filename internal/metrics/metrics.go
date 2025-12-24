package metrics

import (
	"context"
	"sync/atomic"

	"github.com/messkan/PromptCache/internal/storage"
)

// Metrics tracks cache statistics
type Metrics struct {
	cacheHits     atomic.Int64
	cacheMisses   atomic.Int64
	evictionCount atomic.Int64
	store         storage.Storage
}

// MetricsResponse represents the JSON response for the metrics endpoint
type MetricsResponse struct {
	CacheHits          int64   `json:"cache_hits"`
	CacheMisses        int64   `json:"cache_misses"`
	EvictionCount      int64   `json:"eviction_count"`
	StoredVectorsCount int64   `json:"stored_vectors_count"`
	HitRate            float64 `json:"hit_rate"`
}

// NewMetrics creates a new Metrics instance
func NewMetrics(store storage.Storage) *Metrics {
	return &Metrics{
		store: store,
	}
}

// IncrementCacheHits increments the cache hits counter
func (m *Metrics) IncrementCacheHits() {
	m.cacheHits.Add(1)
}

// IncrementCacheMisses increments the cache misses counter
func (m *Metrics) IncrementCacheMisses() {
	m.cacheMisses.Add(1)
}

// IncrementEvictionCount increments the eviction counter
func (m *Metrics) IncrementEvictionCount() {
	m.evictionCount.Add(1)
}

// GetMetrics returns the current metrics as a MetricsResponse
func (m *Metrics) GetMetrics(ctx context.Context) (MetricsResponse, error) {
	hits := m.cacheHits.Load()
	misses := m.cacheMisses.Load()
	evictions := m.evictionCount.Load()

	// Calculate hit rate
	var hitRate float64
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	// Get stored vectors count
	storedVectorsCount, err := m.store.CountEmbeddings(ctx)
	if err != nil {
		return MetricsResponse{}, err
	}

	return MetricsResponse{
		CacheHits:          hits,
		CacheMisses:        misses,
		EvictionCount:      evictions,
		StoredVectorsCount: storedVectorsCount,
		HitRate:            hitRate,
	}, nil
}
