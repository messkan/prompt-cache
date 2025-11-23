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
