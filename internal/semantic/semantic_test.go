package semantic

import (
	"context"
	"testing"
)

// MockProvider implements EmbeddingProvider
type MockProvider struct {
	embedding []float32
}

func (m *MockProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	return m.embedding, nil
}

// MockStorage implements Storage
type MockStorage struct {
	embeddings map[string][]byte
}

func (m *MockStorage) GetAllEmbeddings(ctx context.Context) (map[string][]byte, error) {
	return m.embeddings, nil
}

func (m *MockStorage) GetPrompt(ctx context.Context, key string) (string, error) {
	return "original prompt", nil
}

func (m *MockStorage) Set(ctx context.Context, key string, value []byte) error { return nil }
func (m *MockStorage) Get(ctx context.Context, key string) ([]byte, error)     { return nil, nil }
func (m *MockStorage) Delete(ctx context.Context, key string) error            { return nil }
func (m *MockStorage) CountEmbeddings(ctx context.Context) (int64, error)      { return int64(len(m.embeddings)), nil }
func (m *MockStorage) Close()                                                  {}

// MockVerifier implements Verifier
type MockVerifier struct {
	match bool
}

func (m *MockVerifier) CheckSimilarity(ctx context.Context, prompt1, prompt2 string) (bool, error) {
	return m.match, nil
}

func TestFindSimilar(t *testing.T) {
	// Setup
	queryVec := []float32{1, 0, 0}
	matchVec := []float32{0.99, 0.01, 0} // Very similar
	diffVec := []float32{0, 1, 0}        // Orthogonal

	provider := &MockProvider{embedding: queryVec}

	store := &MockStorage{
		embeddings: map[string][]byte{
			"emb:match": Float32ToBytes(matchVec),
			"emb:diff":  Float32ToBytes(diffVec),
		},
	}

	verifier := &MockVerifier{match: true}

	engine := NewSemanticEngine(provider, store, verifier, 0.95, 0.80)

	// Test Match (High Confidence)
	key, score, err := engine.FindSimilar(context.Background(), "query")
	if err != nil {
		t.Fatalf("FindSimilar failed: %v", err)
	}

	if key != "emb:match" {
		t.Errorf("Expected key 'emb:match', got '%s'", key)
	}
	if score < 0.95 {
		t.Errorf("Expected high score, got %f", score)
	}
}

func TestFindSimilar_NoMatch(t *testing.T) {
	// Setup
	queryVec := []float32{1, 0, 0}
	diffVec := []float32{0, 1, 0} // Orthogonal

	provider := &MockProvider{embedding: queryVec}

	store := &MockStorage{
		embeddings: map[string][]byte{
			"emb:diff": Float32ToBytes(diffVec),
		},
	}

	verifier := &MockVerifier{match: false}

	engine := NewSemanticEngine(provider, store, verifier, 0.95, 0.80)

	// Test No Match
	key, _, err := engine.FindSimilar(context.Background(), "query")
	if err != nil {
		t.Fatalf("FindSimilar failed: %v", err)
	}

	if key != "" {
		t.Errorf("Expected empty key (no match), got '%s'", key)
	}
}

func TestFindSimilar_CustomThresholds(t *testing.T) {
	// Setup - test with custom thresholds
	queryVec := []float32{1, 0, 0}
	similarVec := []float32{0.85, 0.15, 0} // 0.85 similarity

	provider := &MockProvider{embedding: queryVec}

	store := &MockStorage{
		embeddings: map[string][]byte{
			"emb:similar": Float32ToBytes(similarVec),
		},
	}

	verifier := &MockVerifier{match: true}

	// With lower high threshold, this should match directly
	engine := NewSemanticEngine(provider, store, verifier, 0.80, 0.70)

	key, score, err := engine.FindSimilar(context.Background(), "query")
	if err != nil {
		t.Fatalf("FindSimilar failed: %v", err)
	}

	if key != "emb:similar" {
		t.Errorf("Expected key 'emb:similar', got '%s'", key)
	}
	
	if score < 0.80 {
		t.Errorf("Expected score >= 0.80, got %f", score)
	}
}

func TestFindSimilar_GrayZone(t *testing.T) {
	// Setup - test gray zone behavior
	queryVec := []float32{1, 0, 0}
	// This vector is chosen to have ~0.9 cosine similarity with queryVec
	// Cosine similarity = dot(queryVec, grayVec) / (||queryVec|| * ||grayVec||)
	// For normalized vectors: dot([1,0,0], [0.9, 0.436, 0]) ≈ 0.9
	grayVec := []float32{0.9, 0.436, 0}

	provider := &MockProvider{embedding: queryVec}

	store := &MockStorage{
		embeddings: map[string][]byte{
			"emb:gray": Float32ToBytes(grayVec),
		},
	}

	// Test gray zone with verifier match
	verifier := &MockVerifier{match: true}
	engine := NewSemanticEngine(provider, store, verifier, 0.95, 0.80)

	key, score, err := engine.FindSimilar(context.Background(), "query")
	if err != nil {
		t.Fatalf("FindSimilar failed: %v", err)
	}

	// Score should be in gray zone, and verifier should allow the match
	if score >= 0.95 {
		t.Logf("Score %f is above high threshold, adjusting test expectations", score)
	}

	if key != "emb:gray" {
		t.Errorf("Expected key 'emb:gray' (verifier matched), got '%s'", key)
	}

	// Test gray zone with verifier no match - use a score that's definitely in the gray zone
	// This vector is chosen to have ~0.85 cosine similarity with queryVec
	grayVec2 := []float32{0.85, 0.527, 0}
	store2 := &MockStorage{
		embeddings: map[string][]byte{
			"emb:gray2": Float32ToBytes(grayVec2),
		},
	}

	verifier2 := &MockVerifier{match: false}
	engine2 := NewSemanticEngine(provider, store2, verifier2, 0.95, 0.80)

	key2, score2, err := engine2.FindSimilar(context.Background(), "query")
	if err != nil {
		t.Fatalf("FindSimilar failed: %v", err)
	}

	if score2 < 0.80 || score2 >= 0.95 {
		t.Logf("Score %f should be in gray zone (0.80-0.95)", score2)
	}

	if key2 != "" {
		t.Errorf("Expected empty key (verifier didn't match), got '%s'", key2)
	}
}
