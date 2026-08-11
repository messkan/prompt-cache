package semantic

import (
	"context"
	"testing"

	"github.com/messkan/PromptCache/internal/cachecontext"
	"github.com/messkan/PromptCache/internal/storage"
)

func TestFindSimilarDoesNotCrossNamespaces(t *testing.T) {
	store, err := storage.NewBadgerStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewBadgerStore: %v", err)
	}
	defer store.Close()

	ctxA := cachecontext.WithNamespace(context.Background(), "tenant-a")
	ctxB := cachecontext.WithNamespace(context.Background(), "tenant-b")

	if err := store.Set(ctxA, "emb:key-a", Float32ToBytes([]float32{1, 0, 0})); err != nil {
		t.Fatalf("store tenant A embedding: %v", err)
	}
	if err := store.Set(ctxA, "prompt:key-a", []byte("tenant A prompt")); err != nil {
		t.Fatalf("store tenant A prompt: %v", err)
	}
	if err := store.Set(ctxB, "emb:key-b", Float32ToBytes([]float32{0, 1, 0})); err != nil {
		t.Fatalf("store tenant B embedding: %v", err)
	}
	if err := store.Set(ctxB, "prompt:key-b", []byte("tenant B prompt")); err != nil {
		t.Fatalf("store tenant B prompt: %v", err)
	}

	provider := &MockProvider{embedding: []float32{1, 0, 0}, similarity: true}
	engine := NewSemanticEngine(provider, store, provider, &Config{
		HighThreshold:          0.95,
		LowThreshold:           0.30,
		EnableGrayZoneVerifier: true,
		EmbeddingDimension:     3,
		UseANNIndex:            false,
	})

	key, _, err := engine.FindSimilar(ctxA, "query")
	if err != nil {
		t.Fatalf("FindSimilar tenant A: %v", err)
	}
	if key != "emb:key-a" {
		t.Fatalf("tenant A expected emb:key-a, got %q", key)
	}

	key, _, err = engine.FindSimilar(ctxB, "query")
	if err != nil {
		t.Fatalf("FindSimilar tenant B: %v", err)
	}
	if key != "" {
		t.Fatalf("tenant B unexpectedly matched tenant A data: %q", key)
	}
}
