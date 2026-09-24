package semantic

import (
	"context"
	"testing"
)

// A deleted entry must stop shadowing live entries in the ANN index.
func TestFindSimilar_DeletedEntryDoesNotShadowLiveEntry(t *testing.T) {
	queryVec := []float32{1, 0, 0}
	deletedVec := []float32{1, 0, 0}     // nearest neighbour, then deleted
	liveVec := []float32{0.99, 0.01, 0} // still a valid high-confidence match

	provider := &MockProvider{embedding: queryVec}
	store := &MockStorage{embeddings: map[string][]byte{
		"emb:deleted": Float32ToBytes(deletedVec),
		"emb:live":    Float32ToBytes(liveVec),
	}}

	config := &Config{
		HighThreshold:      0.95,
		LowThreshold:       0.80,
		UseANNIndex:        true,
		EmbeddingDimension: 3,
	}
	engine := NewSemanticEngine(provider, store, provider, config)
	engine.AddToIndex("emb:deleted", deletedVec)
	engine.AddToIndex("emb:live", liveVec)

	// Same effect as DELETE /v1/cache/:key on the storage side.
	delete(store.embeddings, "emb:deleted")

	key, _, err := engine.FindSimilar(context.Background(), "query")
	if err != nil {
		t.Fatalf("FindSimilar failed: %v", err)
	}
	if key != "emb:live" {
		t.Fatalf("expected live entry to match, got %q (deleted vector still in ANN index)", key)
	}
}
