package semantic

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/messkan/PromptCache/internal/ann"
	"github.com/messkan/PromptCache/internal/cachecontext"
	"github.com/messkan/PromptCache/internal/metrics"
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

type Provider interface {
	EmbeddingProvider
	Verifier
	ForwardChatCompletion(ctx context.Context, requestBody []byte) ([]byte, int, error)
	ForwardStreamingChatCompletion(ctx context.Context, requestBody []byte, w http.ResponseWriter) ([]byte, int, error)
}

type Config struct {
	HighThreshold          float32
	LowThreshold           float32
	EnableGrayZoneVerifier bool
	EmbeddingDimension     int
	UseANNIndex            bool
}

func LoadConfig() *Config {
	config := &Config{
		HighThreshold:          0.70,
		LowThreshold:           0.30,
		EnableGrayZoneVerifier: true,
		EmbeddingDimension:     1536,
		UseANNIndex:            true,
	}

	if val := os.Getenv("CACHE_HIGH_THRESHOLD"); val != "" {
		if f, err := strconv.ParseFloat(val, 32); err == nil && f > 0 && f <= 1.0 {
			config.HighThreshold = float32(f)
		}
	}
	if val := os.Getenv("CACHE_LOW_THRESHOLD"); val != "" {
		if f, err := strconv.ParseFloat(val, 32); err == nil && f > 0 && f <= 1.0 {
			config.LowThreshold = float32(f)
		}
	}
	if val := os.Getenv("ENABLE_GRAY_ZONE_VERIFIER"); val != "" {
		config.EnableGrayZoneVerifier = val == "true" || val == "1" || val == "yes"
	}
	if val := os.Getenv("EMBEDDING_DIMENSION"); val != "" {
		if dim, err := strconv.Atoi(val); err == nil && dim > 0 {
			config.EmbeddingDimension = dim
		}
	}
	if val := os.Getenv("USE_ANN_INDEX"); val != "" {
		config.UseANNIndex = val == "true" || val == "1" || val == "yes"
	}
	if config.HighThreshold <= config.LowThreshold {
		config.HighThreshold = 0.70
		config.LowThreshold = 0.30
	}
	return config
}

type SemanticEngine struct {
	Provider               Provider
	Store                  Storage
	Verifier               Verifier
	HighThreshold          float32
	LowThreshold           float32
	EnableGrayZoneVerifier bool
	mu                     sync.RWMutex
	currentProviderName    string
	annIndex               *ann.Index
	useANNIndex            bool
	embeddingDimension     int
}

func NewSemanticEngine(p Provider, s Storage, v Verifier, config *Config) *SemanticEngine {
	if config == nil {
		config = LoadConfig()
	}

	providerName := "openai"
	if val := os.Getenv("EMBEDDING_PROVIDER"); val != "" {
		providerName = strings.ToLower(val)
	}

	se := &SemanticEngine{
		Provider:               p,
		Store:                  s,
		Verifier:               v,
		HighThreshold:          config.HighThreshold,
		LowThreshold:           config.LowThreshold,
		EnableGrayZoneVerifier: config.EnableGrayZoneVerifier,
		currentProviderName:    providerName,
		useANNIndex:            config.UseANNIndex,
		embeddingDimension:     config.EmbeddingDimension,
	}

	if config.UseANNIndex {
		se.annIndex = ann.New(config.EmbeddingDimension)
		go se.rebuildANNIndex()
	}
	return se
}

// rebuildANNIndex indexes only the legacy/default partition. Explicit cache
// namespaces use a filtered linear scan so ANN candidates can never cross a
// partition boundary.
func (se *SemanticEngine) rebuildANNIndex() {
	if se.annIndex == nil {
		return
	}
	stored, err := se.Store.GetAllEmbeddings(context.Background())
	if err != nil {
		return
	}
	count := 0
	for key, embBytes := range stored {
		se.annIndex.Add(key, BytesToFloat32(embBytes))
		count++
	}
	metrics.Get().SetStoredVectors(uint64(count))
}

// AddToIndex adds only default-partition embeddings to the ANN index. A
// namespaced write is hidden from a background/default storage view and is
// therefore deliberately skipped.
func (se *SemanticEngine) AddToIndex(key string, embedding []float32) {
	if se.annIndex == nil {
		return
	}
	stored, err := se.Store.GetAllEmbeddings(context.Background())
	if err != nil {
		return
	}
	if _, ok := stored[key]; !ok {
		return
	}
	se.annIndex.Add(key, embedding)
	metrics.Get().SetStoredVectors(uint64(len(stored)))
}

func NewProvider() (Provider, error) {
	provider := os.Getenv("EMBEDDING_PROVIDER")
	if provider == "" {
		provider = "openai"
	}
	switch strings.ToLower(provider) {
	case "openai":
		return NewOpenAIProvider(), nil
	case "mistral":
		return NewMistralProvider(), nil
	case "claude":
		return NewClaudeProvider(), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s (supported: openai, mistral, claude)", provider)
	}
}

func (se *SemanticEngine) SetProvider(providerName string) error {
	providerName = strings.ToLower(providerName)
	var newProvider Provider
	switch providerName {
	case "openai":
		newProvider = NewOpenAIProvider()
	case "mistral":
		newProvider = NewMistralProvider()
	case "claude":
		newProvider = NewClaudeProvider()
	default:
		return fmt.Errorf("unsupported provider: %s (supported: openai, mistral, claude)", providerName)
	}
	se.mu.Lock()
	se.Provider = newProvider
	se.Verifier = newProvider
	se.currentProviderName = providerName
	se.mu.Unlock()
	return nil
}

func (se *SemanticEngine) GetCurrentProvider() string {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.currentProviderName
}

func (se *SemanticEngine) GetProvider() Provider {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Provider
}

func (se *SemanticEngine) ForwardChatCompletion(ctx context.Context, requestBody []byte) ([]byte, int, error) {
	se.mu.RLock()
	provider := se.Provider
	se.mu.RUnlock()
	return provider.ForwardChatCompletion(ctx, requestBody)
}

func (se *SemanticEngine) ForwardStreamingChatCompletion(ctx context.Context, requestBody []byte, w http.ResponseWriter) ([]byte, int, error) {
	se.mu.RLock()
	provider := se.Provider
	se.mu.RUnlock()
	return provider.ForwardStreamingChatCompletion(ctx, requestBody, w)
}

func (se *SemanticEngine) GetConfig() map[string]interface{} {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return map[string]interface{}{
		"high_threshold":            se.HighThreshold,
		"low_threshold":             se.LowThreshold,
		"enable_gray_zone_verifier": se.EnableGrayZoneVerifier,
	}
}

func (se *SemanticEngine) UpdateThresholds(high, low float32, enableGrayZone *bool) error {
	if high < 0 || high > 1.0 {
		return fmt.Errorf("high_threshold must be between 0 and 1.0, got %.4f", high)
	}
	if low < 0 || low > 1.0 {
		return fmt.Errorf("low_threshold must be between 0 and 1.0, got %.4f", low)
	}
	if high <= low {
		return fmt.Errorf("high_threshold (%.4f) must be greater than low_threshold (%.4f)", high, low)
	}
	se.mu.Lock()
	se.HighThreshold = high
	se.LowThreshold = low
	if enableGrayZone != nil {
		se.EnableGrayZoneVerifier = *enableGrayZone
	}
	se.mu.Unlock()
	return nil
}

func (se *SemanticEngine) FindSimilar(ctx context.Context, text string) (string, float32, error) {
	se.mu.RLock()
	provider := se.Provider
	verifier := se.Verifier
	// The ANN index intentionally contains only the default partition. Explicit
	// namespaces must search the storage view already filtered to that namespace.
	useANN := se.useANNIndex && se.annIndex != nil && cachecontext.NamespaceID(ctx) == ""
	se.mu.RUnlock()

	m := metrics.Get()
	queryEmb, err := provider.Embed(ctx, text)
	if err != nil {
		return "", 0, err
	}

	var bestKey string
	var bestSim float32

	if useANN {
		keys, distances := se.annIndex.Search(queryEmb, 1)
		if len(keys) > 0 {
			bestKey = keys[0]
			bestSim = 1.0 - distances[0]
			embBytes, err := se.Store.GetAllEmbeddings(ctx)
			if err == nil {
				if emb, ok := embBytes[bestKey]; ok {
					bestSim = CosineSimilarity(queryEmb, BytesToFloat32(emb))
				} else {
					bestKey = ""
					bestSim = 0
				}
			}
		}
	} else {
		stored, err := se.Store.GetAllEmbeddings(ctx)
		if err != nil {
			return "", 0, err
		}
		for key, embBytes := range stored {
			sim := CosineSimilarity(queryEmb, BytesToFloat32(embBytes))
			if sim > bestSim {
				bestSim = sim
				bestKey = key
			}
		}
	}

	if bestSim >= se.HighThreshold {
		m.RecordCacheHit()
		return bestKey, bestSim, nil
	}
	if bestSim < se.LowThreshold {
		m.RecordCacheMiss()
		return "", bestSim, nil
	}

	m.RecordCacheGrayZone()
	if !se.EnableGrayZoneVerifier {
		m.RecordCacheMiss()
		return "", bestSim, nil
	}

	hashKey := strings.TrimPrefix(bestKey, "emb:")
	originalPrompt, err := se.Store.GetPrompt(ctx, hashKey)
	if err != nil {
		m.RecordCacheMiss()
		return "", bestSim, nil
	}

	isMatch, err := verifier.CheckSimilarity(ctx, text, originalPrompt)
	if err != nil {
		m.RecordCacheMiss()
		return "", bestSim, err
	}
	if isMatch {
		m.RecordCacheHit()
		return bestKey, bestSim, nil
	}
	m.RecordCacheMiss()
	return "", bestSim, nil
}
