package semantic

import (
	"context"
)

type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type Storage interface {
	GetAllEmbeddings(ctx context.Context) (map[string][]byte, error)
}

type SemanticEngine struct {
	Provider  EmbeddingProvider
	Store     Storage
	Threshold float32
}

func NewSemanticEngine(p EmbeddingProvider, s Storage, threshold float32) *SemanticEngine {
	return &SemanticEngine{
		Provider:  p,
		Store:     s,
		Threshold: threshold,
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

	if bestSim < se.Threshold {
		return "", bestSim, nil
	}

	return bestKey, bestSim, nil
}
