package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds cache and request metrics
type Metrics struct {
	// Cache metrics
	CacheHits      uint64
	CacheMisses    uint64
	CacheGrayZone  uint64
	StoredVectors  uint64

	// Latency tracking
	totalLatencyNs uint64
	requestCount   uint64

	// Request metrics
	TotalRequests  uint64
	FailedRequests uint64

	// Provider metrics
	ProviderCalls  uint64
	ProviderErrors uint64

	mu sync.RWMutex
}

var globalMetrics = &Metrics{}

// Get returns the global metrics instance
func Get() *Metrics {
	return globalMetrics
}

// RecordCacheHit increments the cache hit counter
func (m *Metrics) RecordCacheHit() {
	atomic.AddUint64(&m.CacheHits, 1)
}

// RecordCacheMiss increments the cache miss counter
func (m *Metrics) RecordCacheMiss() {
	atomic.AddUint64(&m.CacheMisses, 1)
}

// RecordCacheGrayZone increments the gray zone counter
func (m *Metrics) RecordCacheGrayZone() {
	atomic.AddUint64(&m.CacheGrayZone, 1)
}

// SetStoredVectors sets the stored vectors count
func (m *Metrics) SetStoredVectors(count uint64) {
	atomic.StoreUint64(&m.StoredVectors, count)
}

// RecordLatency records a request latency
func (m *Metrics) RecordLatency(d time.Duration) {
	atomic.AddUint64(&m.totalLatencyNs, uint64(d.Nanoseconds()))
	atomic.AddUint64(&m.requestCount, 1)
}

// RecordRequest increments the total request counter
func (m *Metrics) RecordRequest() {
	atomic.AddUint64(&m.TotalRequests, 1)
}

// RecordFailedRequest increments the failed request counter
func (m *Metrics) RecordFailedRequest() {
	atomic.AddUint64(&m.FailedRequests, 1)
}

// RecordProviderCall increments the provider call counter
func (m *Metrics) RecordProviderCall() {
	atomic.AddUint64(&m.ProviderCalls, 1)
}

// RecordProviderError increments the provider error counter
func (m *Metrics) RecordProviderError() {
	atomic.AddUint64(&m.ProviderErrors, 1)
}

// GetStats returns current metrics as a map
func (m *Metrics) GetStats() map[string]interface{} {
	hits := atomic.LoadUint64(&m.CacheHits)
	misses := atomic.LoadUint64(&m.CacheMisses)
	grayZone := atomic.LoadUint64(&m.CacheGrayZone)
	vectors := atomic.LoadUint64(&m.StoredVectors)
	totalLatency := atomic.LoadUint64(&m.totalLatencyNs)
	reqCount := atomic.LoadUint64(&m.requestCount)
	totalReqs := atomic.LoadUint64(&m.TotalRequests)
	failedReqs := atomic.LoadUint64(&m.FailedRequests)
	providerCalls := atomic.LoadUint64(&m.ProviderCalls)
	providerErrors := atomic.LoadUint64(&m.ProviderErrors)

	total := hits + misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	avgLatencyMs := float64(0)
	if reqCount > 0 {
		avgLatencyMs = float64(totalLatency) / float64(reqCount) / 1e6
	}

	return map[string]interface{}{
		"cache": map[string]interface{}{
			"hits":           hits,
			"misses":         misses,
			"gray_zone":      grayZone,
			"hit_rate":       hitRate,
			"stored_vectors": vectors,
		},
		"latency": map[string]interface{}{
			"avg_ms":        avgLatencyMs,
			"request_count": reqCount,
		},
		"requests": map[string]interface{}{
			"total":  totalReqs,
			"failed": failedReqs,
		},
		"provider": map[string]interface{}{
			"calls":  providerCalls,
			"errors": providerErrors,
		},
	}
}

// GetPrometheusMetrics returns metrics in Prometheus format
func (m *Metrics) GetPrometheusMetrics() string {
	hits := atomic.LoadUint64(&m.CacheHits)
	misses := atomic.LoadUint64(&m.CacheMisses)
	grayZone := atomic.LoadUint64(&m.CacheGrayZone)
	vectors := atomic.LoadUint64(&m.StoredVectors)
	totalLatency := atomic.LoadUint64(&m.totalLatencyNs)
	reqCount := atomic.LoadUint64(&m.requestCount)
	totalReqs := atomic.LoadUint64(&m.TotalRequests)
	failedReqs := atomic.LoadUint64(&m.FailedRequests)
	providerCalls := atomic.LoadUint64(&m.ProviderCalls)
	providerErrors := atomic.LoadUint64(&m.ProviderErrors)

	avgLatencyMs := float64(0)
	if reqCount > 0 {
		avgLatencyMs = float64(totalLatency) / float64(reqCount) / 1e6
	}

	return `# HELP promptcache_cache_hits_total Total number of cache hits
# TYPE promptcache_cache_hits_total counter
promptcache_cache_hits_total ` + uintToStr(hits) + `

# HELP promptcache_cache_misses_total Total number of cache misses
# TYPE promptcache_cache_misses_total counter
promptcache_cache_misses_total ` + uintToStr(misses) + `

# HELP promptcache_cache_gray_zone_total Total number of gray zone verifications
# TYPE promptcache_cache_gray_zone_total counter
promptcache_cache_gray_zone_total ` + uintToStr(grayZone) + `

# HELP promptcache_stored_vectors Current number of stored vectors
# TYPE promptcache_stored_vectors gauge
promptcache_stored_vectors ` + uintToStr(vectors) + `

# HELP promptcache_latency_avg_ms Average request latency in milliseconds
# TYPE promptcache_latency_avg_ms gauge
promptcache_latency_avg_ms ` + floatToStr(avgLatencyMs) + `

# HELP promptcache_requests_total Total number of requests
# TYPE promptcache_requests_total counter
promptcache_requests_total ` + uintToStr(totalReqs) + `

# HELP promptcache_requests_failed_total Total number of failed requests
# TYPE promptcache_requests_failed_total counter
promptcache_requests_failed_total ` + uintToStr(failedReqs) + `

# HELP promptcache_provider_calls_total Total number of provider API calls
# TYPE promptcache_provider_calls_total counter
promptcache_provider_calls_total ` + uintToStr(providerCalls) + `

# HELP promptcache_provider_errors_total Total number of provider API errors
# TYPE promptcache_provider_errors_total counter
promptcache_provider_errors_total ` + uintToStr(providerErrors) + `
`
}

func uintToStr(n uint64) string {
	return string(formatUint(n))
}

func floatToStr(f float64) string {
	return string(formatFloat(f))
}

func formatUint(n uint64) []byte {
	if n == 0 {
		return []byte("0")
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return buf[i:]
}

func formatFloat(f float64) []byte {
	// Simple float formatting with 3 decimal places
	if f == 0 {
		return []byte("0.000")
	}
	
	intPart := int64(f)
	fracPart := int64((f - float64(intPart)) * 1000)
	if fracPart < 0 {
		fracPart = -fracPart
	}
	
	result := formatUint(uint64(intPart))
	result = append(result, '.')
	
	// Pad with zeros if needed
	if fracPart < 10 {
		result = append(result, '0', '0')
	} else if fracPart < 100 {
		result = append(result, '0')
	}
	result = append(result, formatUint(uint64(fracPart))...)
	
	return result
}

// Reset resets all metrics to zero
func (m *Metrics) Reset() {
	atomic.StoreUint64(&m.CacheHits, 0)
	atomic.StoreUint64(&m.CacheMisses, 0)
	atomic.StoreUint64(&m.CacheGrayZone, 0)
	atomic.StoreUint64(&m.StoredVectors, 0)
	atomic.StoreUint64(&m.totalLatencyNs, 0)
	atomic.StoreUint64(&m.requestCount, 0)
	atomic.StoreUint64(&m.TotalRequests, 0)
	atomic.StoreUint64(&m.FailedRequests, 0)
	atomic.StoreUint64(&m.ProviderCalls, 0)
	atomic.StoreUint64(&m.ProviderErrors, 0)
}
