package semantic

import (
	"context"
	"strings"
)

type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type Storage interface {
	GetAllEmbeddings(ctx context.Context) (map[string][]byte, error)
	GetPrompt(ctx context.Context, key string) (string, error)
}

type Verifier interface {
	CheckSimilarity(ctx context.Context, prompt1, prompt2 string) (bool, error)
}

type SemanticEngine struct {
	Provider      EmbeddingProvider
	Store         Storage
	Verifier      Verifier
	HighThreshold float32
	LowThreshold  float32
}

func NewSemanticEngine(p EmbeddingProvider, s Storage, v Verifier, highThreshold, lowThreshold float32) *SemanticEngine {
	return &SemanticEngine{
		Provider:      p,
		Store:         s,
		Verifier:      v,
		HighThreshold: highThreshold,
		LowThreshold:  lowThreshold,
	}
}

func (se *SemanticEngine) FindSimilar(ctx context.Context, text string) (string, float32, error) {
	queryEmb, err := se.Provider.Embed(ctx, text)
	if err != nil {
		return "", 0, err
	}

	stored, err := se.Store.GetAllEmbeddings(ctx)
	if err != nil {
		return "", 0, err
	}

	bestKey := ""
	bestSim := float32(0)

	for key, embBytes := range stored {
		embVec := BytesToFloat32(embBytes)
		sim := CosineSimilarity(queryEmb, embVec)

		if sim > bestSim {
			bestSim = sim
			bestKey = key
		}
	}

	// 1. Clear Match
	if bestSim >= se.HighThreshold {
		return bestKey, bestSim, nil
	}

	// 2. Clear Mismatch
	if bestSim < se.LowThreshold {
		return "", bestSim, nil
	}

	// 3. Gray Zone -> Smart Verification
	// The key in storage has "emb:" prefix, we need to strip it to get the hash
	hashKey := strings.TrimPrefix(bestKey, "emb:")

	originalPrompt, err := se.Store.GetPrompt(ctx, hashKey)
	if err != nil {
		// If we can't find the prompt, we can't verify, so we assume miss to be safe
		return "", bestSim, nil
	}

	isMatch, err := se.Verifier.CheckSimilarity(ctx, text, originalPrompt)
	if err != nil {
		return "", bestSim, err
	}

	if isMatch {
		return bestKey, bestSim, nil
	}

	return "", bestSim, nil
}
