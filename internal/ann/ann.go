package ann

import (
	"math"
	"sort"
	"sync"
)

// Index is a simple in-memory approximate nearest neighbor index
// using a brute-force approach with optimizations for small-medium datasets
type Index struct {
	vectors  map[string][]float32
	keys     []string
	dim      int
	mu       sync.RWMutex
}

// New creates a new ANN index with the given embedding dimension
func New(dim int) *Index {
	return &Index{
		vectors: make(map[string][]float32),
		keys:    []string{},
		dim:     dim,
	}
}

// Add adds a vector to the index
func (i *Index) Add(key string, vector []float32) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if _, ok := i.vectors[key]; ok {
		// Already exists, update vector
		i.vectors[key] = vector
		return
	}

	i.vectors[key] = vector
	i.keys = append(i.keys, key)
}

// Remove removes a vector from the index
func (i *Index) Remove(key string) {
	i.mu.Lock()
	defer i.mu.Unlock()

	delete(i.vectors, key)
	
	// Remove from keys slice
	for idx, k := range i.keys {
		if k == key {
			i.keys = append(i.keys[:idx], i.keys[idx+1:]...)
			break
		}
	}
}

// searchResult holds a search result with key and distance
type searchResult struct {
	key      string
	distance float32
}

// Search finds the k nearest neighbors to the query vector
// Returns keys and distances (lower distance = more similar)
func (i *Index) Search(vector []float32, k int) ([]string, []float32) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if len(i.vectors) == 0 {
		return nil, nil
	}

	// Calculate distances for all vectors
	results := make([]searchResult, 0, len(i.vectors))
	for key, v := range i.vectors {
		dist := cosineDistance(vector, v)
		results = append(results, searchResult{key: key, distance: dist})
	}

	// Sort by distance (ascending)
	sort.Slice(results, func(a, b int) bool {
		return results[a].distance < results[b].distance
	})

	// Take top k
	if k > len(results) {
		k = len(results)
	}

	keys := make([]string, k)
	distances := make([]float32, k)
	for idx := 0; idx < k; idx++ {
		keys[idx] = results[idx].key
		distances[idx] = results[idx].distance
	}

	return keys, distances
}

// Size returns the number of vectors in the index
func (i *Index) Size() int {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return len(i.vectors)
}

// cosineDistance calculates 1 - cosine similarity between two vectors
func cosineDistance(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 1.0 // Maximum distance
	}

	var dot, normA, normB float32
	for idx := range a {
		dot += a[idx] * b[idx]
		normA += a[idx] * a[idx]
		normB += b[idx] * b[idx]
	}

	if normA == 0 || normB == 0 {
		return 1.0
	}

	similarity := dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
	return 1.0 - similarity
}