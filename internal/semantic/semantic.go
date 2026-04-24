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

// Provider combines EmbeddingProvider and Verifier interfaces
type Provider interface {
	EmbeddingProvider
	Verifier
	ForwardChatCompletion(ctx context.Context, requestBody []byte) ([]byte, int, error)
	// ForwardStreamingChatCompletion streams SSE events to w and returns a buffered
	// non-streaming JSON response suitable for caching.
	ForwardStreamingChatCompletion(ctx context.Context, requestBody []byte, w http.ResponseWriter) ([]byte, int, error)
}

// Config holds configuration for the semantic engine
type Config struct {
	HighThreshold          float32
	LowThreshold           float32
	EnableGrayZoneVerifier bool
	EmbeddingDimension     int  // Dimension of embeddings (e.g., 1536 for OpenAI, 1024 for Voyage)
	UseANNIndex            bool // Whether to use ANN index for similarity search
}

// LoadConfig loads configuration from environment variables with sensible defaults
func LoadConfig() *Config {
	config := &Config{
		HighThreshold:          0.70, // Default: 70% similarity for direct cache hit
		LowThreshold:           0.30, // Default: below 30% is a clear miss
		EnableGrayZoneVerifier: true, // Default: enable smart verification
		EmbeddingDimension:     1536, // Default: OpenAI text-embedding-3-small
		UseANNIndex:            true, // Default: use ANN index
	}

	// Load high threshold
	if val := os.Getenv("CACHE_HIGH_THRESHOLD"); val != "" {
		if f, err := strconv.ParseFloat(val, 32); err == nil && f > 0 && f <= 1.0 {
			config.HighThreshold = float32(f)
		}
	}

	// Load low threshold
	if val := os.Getenv("CACHE_LOW_THRESHOLD"); val != "" {
		if f, err := strconv.ParseFloat(val, 32); err == nil && f > 0 && f <= 1.0 {
			config.LowThreshold = float32(f)
		}
	}

	// Load gray zone verifier setting
	if val := os.Getenv("ENABLE_GRAY_ZONE_VERIFIER"); val != "" {
		config.EnableGrayZoneVerifier = val == "true" || val == "1" || val == "yes"
	}

	// Load embedding dimension
	if val := os.Getenv("EMBEDDING_DIMENSION"); val != "" {
		if dim, err := strconv.Atoi(val); err == nil && dim > 0 {
			config.EmbeddingDimension = dim
		}
	}

	// Load ANN index setting
	if val := os.Getenv("USE_ANN_INDEX"); val != "" {
		config.UseANNIndex = val == "true" || val == "1" || val == "yes"
	}

	// Ensure high threshold is greater than low threshold
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
	mu                     sync.RWMutex // Protects Provider and Verifier
	currentProviderName    string       // Tracks the current provider name
	annIndex               *ann.Index   // ANN index for fast similarity search
	useANNIndex            bool         // Whether to use ANN index
	embeddingDimension     int          // Dimension of embeddings
}

func NewSemanticEngine(p Provider, s Storage, v Verifier, config *Config) *SemanticEngine {
	if config == nil {
		config = LoadConfig()
	}
	
	// Detect provider name
	providerName := "unknown"
	if val := os.Getenv("EMBEDDING_PROVIDER"); val != "" {
		providerName = strings.ToLower(val)
	} else {
		providerName = "openai" // default
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

	// Initialize ANN index if enabled
	if config.UseANNIndex {
		se.annIndex = ann.New(config.EmbeddingDimension)
		// Load existing embeddings into index
		go se.rebuildANNIndex()
	}

	return se
}

// rebuildANNIndex loads all existing embeddings into the ANN index
func (se *SemanticEngine) rebuildANNIndex() {
	if se.annIndex == nil {
		return
	}

	ctx := context.Background()
	stored, err := se.Store.GetAllEmbeddings(ctx)
	if err != nil {
		return
	}

	count := 0
	for key, embBytes := range stored {
		embVec := BytesToFloat32(embBytes)
		se.annIndex.Add(key, embVec)
		count++
	}

	m := metrics.Get()
	m.SetStoredVectors(uint64(count))
}

// AddToIndex adds an embedding to the ANN index
func (se *SemanticEngine) AddToIndex(key string, embedding []float32) {
	if se.annIndex != nil {
		se.annIndex.Add(key, embedding)
		m := metrics.Get()
		// Increment stored vectors count
		stored, _ := se.Store.GetAllEmbeddings(context.Background())
		m.SetStoredVectors(uint64(len(stored)))
	}
}

// NewProvider creates an embedding provider based on the EMBEDDING_PROVIDER environment variable
// Supported providers: openai (default), mistral, claude
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

// SetProvider dynamically changes the embedding provider at runtime
func (se *SemanticEngine) SetProvider(providerName string) error {
	providerName = strings.ToLower(providerName)
	
	var newProvider Provider
	var err error
	
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
	
	return err
}

// GetCurrentProvider returns the name of the currently active provider
func (se *SemanticEngine) GetCurrentProvider() string {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.currentProviderName
}

// GetProvider returns the current provider instance (thread-safe)
func (se *SemanticEngine) GetProvider() Provider {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Provider
}

// ForwardChatCompletion forwards the request to the current provider
func (se *SemanticEngine) ForwardChatCompletion(ctx context.Context, requestBody []byte) ([]byte, int, error) {
	se.mu.RLock()
	provider := se.Provider
	se.mu.RUnlock()
	return provider.ForwardChatCompletion(ctx, requestBody)
}

// ForwardStreamingChatCompletion streams SSE events to w via the current provider.
// It returns a buffered full JSON response for caching and the HTTP status code.
func (se *SemanticEngine) ForwardStreamingChatCompletion(ctx context.Context, requestBody []byte, w http.ResponseWriter) ([]byte, int, error) {
	se.mu.RLock()
	provider := se.Provider
	se.mu.RUnlock()
	return provider.ForwardStreamingChatCompletion(ctx, requestBody, w)
}

// GetConfig returns the current semantic configuration (thread-safe).
func (se *SemanticEngine) GetConfig() map[string]interface{} {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return map[string]interface{}{
		"high_threshold":            se.HighThreshold,
		"low_threshold":             se.LowThreshold,
		"enable_gray_zone_verifier": se.EnableGrayZoneVerifier,
	}
}

// UpdateThresholds atomically updates the similarity thresholds and optionally the
// gray-zone verifier toggle. Returns an error if high <= low or values are out of range.
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
	useANN := se.useANNIndex && se.annIndex != nil
	se.mu.RUnlock()
	
	m := metrics.Get()
	
	queryEmb, err := provider.Embed(ctx, text)
	if err != nil {
		return "", 0, err
	}

	var bestKey string
	var bestSim float32

	if useANN {
		// Use ANN index for O(log n) similarity search
		keys, distances := se.annIndex.Search(queryEmb, 1)
		if len(keys) > 0 {
			bestKey = keys[0]
			// HNSW returns distance, convert to similarity
			// For cosine distance: similarity = 1 - distance
			bestSim = 1.0 - distances[0]
			
			// Verify with exact cosine similarity for accuracy
			embBytes, err := se.Store.GetAllEmbeddings(ctx)
			if err == nil {
				if emb, ok := embBytes[bestKey]; ok {
					embVec := BytesToFloat32(emb)
					bestSim = CosineSimilarity(queryEmb, embVec)
				}
			}
		}
	} else {
		// Fallback to linear scan
		stored, err := se.Store.GetAllEmbeddings(ctx)
		if err != nil {
			return "", 0, err
		}

		for key, embBytes := range stored {
			embVec := BytesToFloat32(embBytes)
			sim := CosineSimilarity(queryEmb, embVec)

			if sim > bestSim {
				bestSim = sim
				bestKey = key
			}
		}
	}

	// 1. Clear Match
	if bestSim >= se.HighThreshold {
		m.RecordCacheHit()
		return bestKey, bestSim, nil
	}

	// 2. Clear Mismatch
	if bestSim < se.LowThreshold {
		m.RecordCacheMiss()
		return "", bestSim, nil
	}

	// 3. Gray Zone -> Smart Verification (if enabled)
	m.RecordCacheGrayZone()
	
	if !se.EnableGrayZoneVerifier {
		// Gray zone verification disabled, treat as miss
		m.RecordCacheMiss()
		return "", bestSim, nil
	}

	// The key in storage has "emb:" prefix, we need to strip it to get the hash
	hashKey := strings.TrimPrefix(bestKey, "emb:")

	originalPrompt, err := se.Store.GetPrompt(ctx, hashKey)
	if err != nil {
		// If we can't find the prompt, we can't verify, so we assume miss to be safe
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
