package semantic

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
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
}

// Config holds configuration for the semantic engine
type Config struct {
	HighThreshold          float32
	LowThreshold           float32
	EnableGrayZoneVerifier bool
}

// LoadConfig loads configuration from environment variables with sensible defaults
func LoadConfig() *Config {
	config := &Config{
		HighThreshold:          0.70, // Default: 70% similarity for direct cache hit
		LowThreshold:           0.30, // Default: below 30% is a clear miss
		EnableGrayZoneVerifier: true, // Default: enable smart verification
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

	// Ensure high threshold is greater than low threshold
	if config.HighThreshold <= config.LowThreshold {
		config.HighThreshold = 0.70
		config.LowThreshold = 0.30
	}

	return config
}

type SemanticEngine struct {
	Provider               EmbeddingProvider
	Store                  Storage
	Verifier               Verifier
	HighThreshold          float32
	LowThreshold           float32
	EnableGrayZoneVerifier bool
	mu                     sync.RWMutex // Protects Provider and Verifier
	currentProviderName    string       // Tracks the current provider name
}

func NewSemanticEngine(p EmbeddingProvider, s Storage, v Verifier, config *Config) *SemanticEngine {
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
	
	return &SemanticEngine{
		Provider:               p,
		Store:                  s,
		Verifier:               v,
		HighThreshold:          config.HighThreshold,
		LowThreshold:           config.LowThreshold,
		EnableGrayZoneVerifier: config.EnableGrayZoneVerifier,
		currentProviderName:    providerName,
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
func (se *SemanticEngine) GetProvider() EmbeddingProvider {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.Provider
}

func (se *SemanticEngine) FindSimilar(ctx context.Context, text string) (string, float32, error) {
	se.mu.RLock()
	provider := se.Provider
	verifier := se.Verifier
	se.mu.RUnlock()
	
	queryEmb, err := provider.Embed(ctx, text)
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

	// 3. Gray Zone -> Smart Verification (if enabled)
	if !se.EnableGrayZoneVerifier {
		// Gray zone verification disabled, treat as miss
		return "", bestSim, nil
	}

	// The key in storage has "emb:" prefix, we need to strip it to get the hash
	hashKey := strings.TrimPrefix(bestKey, "emb:")

	originalPrompt, err := se.Store.GetPrompt(ctx, hashKey)
	if err != nil {
		// If we can't find the prompt, we can't verify, so we assume miss to be safe
		return "", bestSim, nil
	}

	isMatch, err := verifier.CheckSimilarity(ctx, text, originalPrompt)
	if err != nil {
		return "", bestSim, err
	}

	if isMatch {
		return bestKey, bestSim, nil
	}

	return "", bestSim, nil
}
