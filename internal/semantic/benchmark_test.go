package semantic

import (
	"context"
	"testing"
)

// BenchmarkCosineSimilarity benchmarks the cosine similarity calculation
func BenchmarkCosineSimilarity(b *testing.B) {
	vec1 := make([]float32, 1536) // OpenAI embedding size
	vec2 := make([]float32, 1536)
	
	// Initialize with some values
	for i := range vec1 {
		vec1[i] = float32(i) / 1536.0
		vec2[i] = float32(i+1) / 1536.0
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CosineSimilarity(vec1, vec2)
	}
}

// BenchmarkFloat32ToBytes benchmarks float32 to bytes conversion
func BenchmarkFloat32ToBytes(b *testing.B) {
	vec := make([]float32, 1536)
	for i := range vec {
		vec[i] = float32(i) / 1536.0
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Float32ToBytes(vec)
	}
}

// BenchmarkBytesToFloat32 benchmarks bytes to float32 conversion
func BenchmarkBytesToFloat32(b *testing.B) {
	vec := make([]float32, 1536)
	for i := range vec {
		vec[i] = float32(i) / 1536.0
	}
	bytes := Float32ToBytes(vec)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BytesToFloat32(bytes)
	}
}

// BenchmarkFindSimilar benchmarks the main similarity search
func BenchmarkFindSimilar(b *testing.B) {
	queryVec := make([]float32, 1536)
	storedVec := make([]float32, 1536)
	
	for i := range queryVec {
		queryVec[i] = float32(i) / 1536.0
		storedVec[i] = float32(i+1) / 1536.0
	}
	
	provider := &MockProvider{embedding: queryVec}
	store := &MockStorage{
		embeddings: map[string][]byte{
			"emb:test1": Float32ToBytes(storedVec),
		},
	}
	verifier := &MockVerifier{match: true}
	
	config := &Config{
		HighThreshold:          0.70,
		LowThreshold:           0.30,
		EnableGrayZoneVerifier: true,
	}
	engine := NewSemanticEngine(provider, store, verifier, config)
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = engine.FindSimilar(ctx, "test query")
	}
}

// BenchmarkFindSimilar_MultipleEmbeddings benchmarks similarity search with multiple stored embeddings
func BenchmarkFindSimilar_MultipleEmbeddings(b *testing.B) {
	queryVec := make([]float32, 1536)
	for i := range queryVec {
		queryVec[i] = float32(i) / 1536.0
	}
	
	provider := &MockProvider{embedding: queryVec}
	
	// Create 100 stored embeddings
	embeddings := make(map[string][]byte)
	for j := 0; j < 100; j++ {
		vec := make([]float32, 1536)
		for i := range vec {
			vec[i] = float32(i+j) / 1536.0
		}
		embeddings["emb:test"+string(rune(j))] = Float32ToBytes(vec)
	}
	
	store := &MockStorage{embeddings: embeddings}
	verifier := &MockVerifier{match: true}
	
	config := &Config{
		HighThreshold:          0.70,
		LowThreshold:           0.30,
		EnableGrayZoneVerifier: true,
	}
	engine := NewSemanticEngine(provider, store, verifier, config)
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = engine.FindSimilar(ctx, "test query")
	}
}

// BenchmarkSetProvider benchmarks dynamic provider switching
func BenchmarkSetProvider(b *testing.B) {
	store := &MockStorage{embeddings: make(map[string][]byte)}
	provider := NewOpenAIProvider()
	config := &Config{
		HighThreshold:          0.70,
		LowThreshold:           0.30,
		EnableGrayZoneVerifier: true,
	}
	
	engine := NewSemanticEngine(provider, store, provider, config)
	providers := []string{"openai", "mistral", "claude"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.SetProvider(providers[i%len(providers)])
	}
}

// BenchmarkGetCurrentProvider benchmarks reading the current provider
func BenchmarkGetCurrentProvider(b *testing.B) {
	store := &MockStorage{embeddings: make(map[string][]byte)}
	provider := NewOpenAIProvider()
	config := &Config{
		HighThreshold:          0.70,
		LowThreshold:           0.30,
		EnableGrayZoneVerifier: true,
	}
	
	engine := NewSemanticEngine(provider, store, provider, config)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.GetCurrentProvider()
	}
}
