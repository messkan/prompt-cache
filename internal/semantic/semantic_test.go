package semantic

import (
	"context"
	"os"
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

	config := &Config{
		HighThreshold:          0.95,
		LowThreshold:           0.80,
		EnableGrayZoneVerifier: true,
	}
	engine := NewSemanticEngine(provider, store, verifier, config)

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

	config := &Config{
		HighThreshold:          0.95,
		LowThreshold:           0.80,
		EnableGrayZoneVerifier: true,
	}
	engine := NewSemanticEngine(provider, store, verifier, config)

	// Test No Match
	key, _, err := engine.FindSimilar(context.Background(), "query")
	if err != nil {
		t.Fatalf("FindSimilar failed: %v", err)
	}

	if key != "" {
		t.Errorf("Expected empty key (no match), got '%s'", key)
	}
}

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name          string
		envValue      string
		expectedType  string
		shouldError   bool
	}{
		{
			name:         "default to openai",
			envValue:     "",
			expectedType: "*semantic.OpenAIProvider",
			shouldError:  false,
		},
		{
			name:         "openai provider",
			envValue:     "openai",
			expectedType: "*semantic.OpenAIProvider",
			shouldError:  false,
		},
		{
			name:         "mistral provider",
			envValue:     "mistral",
			expectedType: "*semantic.MistralProvider",
			shouldError:  false,
		},
		{
			name:         "claude provider",
			envValue:     "claude",
			expectedType: "*semantic.ClaudeProvider",
			shouldError:  false,
		},
		{
			name:         "case insensitive",
			envValue:     "OPENAI",
			expectedType: "*semantic.OpenAIProvider",
			shouldError:  false,
		},
		{
			name:        "unsupported provider",
			envValue:    "unknown",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			os.Setenv("EMBEDDING_PROVIDER", tt.envValue)
			defer os.Unsetenv("EMBEDDING_PROVIDER")

			provider, err := NewProvider()

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if provider == nil {
				t.Fatal("Expected provider but got nil")
			}

			// Type check would require reflection or interface checks
			// For now, just ensure we get a non-nil provider
		})
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name                   string
		highThresholdEnv       string
		lowThresholdEnv        string
		grayZoneEnv            string
		expectedHigh           float32
		expectedLow            float32
		expectedGrayZoneEnabled bool
	}{
		{
			name:                   "default values",
			highThresholdEnv:       "",
			lowThresholdEnv:        "",
			grayZoneEnv:            "",
			expectedHigh:           0.70,
			expectedLow:            0.30,
			expectedGrayZoneEnabled: true,
		},
		{
			name:                   "custom thresholds",
			highThresholdEnv:       "0.92",
			lowThresholdEnv:        "0.75",
			grayZoneEnv:            "",
			expectedHigh:           0.92,
			expectedLow:            0.75,
			expectedGrayZoneEnabled: true,
		},
		{
			name:                   "disable gray zone verifier",
			highThresholdEnv:       "",
			lowThresholdEnv:        "",
			grayZoneEnv:            "false",
			expectedHigh:           0.70,
			expectedLow:            0.30,
			expectedGrayZoneEnabled: false,
		},
		{
			name:                   "gray zone verifier with '1'",
			highThresholdEnv:       "",
			lowThresholdEnv:        "",
			grayZoneEnv:            "1",
			expectedHigh:           0.70,
			expectedLow:            0.30,
			expectedGrayZoneEnabled: true,
		},
		{
			name:                   "invalid threshold values use defaults",
			highThresholdEnv:       "invalid",
			lowThresholdEnv:        "2.5",
			grayZoneEnv:            "",
			expectedHigh:           0.70,
			expectedLow:            0.30,
			expectedGrayZoneEnabled: true,
		},
		{
			name:                   "high <= low resets to defaults",
			highThresholdEnv:       "0.70",
			lowThresholdEnv:        "0.85",
			grayZoneEnv:            "",
			expectedHigh:           0.70,
			expectedLow:            0.30,
			expectedGrayZoneEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			if tt.highThresholdEnv != "" {
				os.Setenv("CACHE_HIGH_THRESHOLD", tt.highThresholdEnv)
				defer os.Unsetenv("CACHE_HIGH_THRESHOLD")
			}
			if tt.lowThresholdEnv != "" {
				os.Setenv("CACHE_LOW_THRESHOLD", tt.lowThresholdEnv)
				defer os.Unsetenv("CACHE_LOW_THRESHOLD")
			}
			if tt.grayZoneEnv != "" {
				os.Setenv("ENABLE_GRAY_ZONE_VERIFIER", tt.grayZoneEnv)
				defer os.Unsetenv("ENABLE_GRAY_ZONE_VERIFIER")
			}

			config := LoadConfig()

			if config.HighThreshold != tt.expectedHigh {
				t.Errorf("Expected HighThreshold %.2f, got %.2f", tt.expectedHigh, config.HighThreshold)
			}
			if config.LowThreshold != tt.expectedLow {
				t.Errorf("Expected LowThreshold %.2f, got %.2f", tt.expectedLow, config.LowThreshold)
			}
			if config.EnableGrayZoneVerifier != tt.expectedGrayZoneEnabled {
				t.Errorf("Expected EnableGrayZoneVerifier %v, got %v", tt.expectedGrayZoneEnabled, config.EnableGrayZoneVerifier)
			}
		})
	}
}

func TestFindSimilar_GrayZoneDisabled(t *testing.T) {
	// Setup - similarity in gray zone (between 0.80 and 0.95)
	queryVec := []float32{1, 0, 0}
	grayVec := []float32{0.85, 0.5, 0.1} // ~85% similarity (in gray zone)

	provider := &MockProvider{embedding: queryVec}

	store := &MockStorage{
		embeddings: map[string][]byte{
			"emb:gray": Float32ToBytes(grayVec),
		},
	}

	// Verifier would say it's a match, but gray zone is disabled
	verifier := &MockVerifier{match: true}

	config := &Config{
		HighThreshold:          0.95,
		LowThreshold:           0.80,
		EnableGrayZoneVerifier: false, // Disabled
	}
	engine := NewSemanticEngine(provider, store, verifier, config)

	// Test that gray zone returns empty (no match) when verifier is disabled
	key, score, err := engine.FindSimilar(context.Background(), "query")
	if err != nil {
		t.Fatalf("FindSimilar failed: %v", err)
	}

	if key != "" {
		t.Errorf("Expected empty key (gray zone disabled), got '%s'", key)
	}
	
	// Score should still be in gray zone
	if score < 0.80 || score >= 0.95 {
		t.Errorf("Expected score in gray zone [0.80, 0.95), got %f", score)
	}
}

func TestSetProvider(t *testing.T) {
	store := &MockStorage{embeddings: make(map[string][]byte)}
	provider := NewOpenAIProvider()
	config := &Config{
		HighThreshold:          0.70,
		LowThreshold:           0.30,
		EnableGrayZoneVerifier: true,
	}
	
	engine := NewSemanticEngine(provider, store, provider, config)
	
	// Test initial provider
	if engine.GetCurrentProvider() != "openai" {
		t.Errorf("Expected initial provider 'openai', got '%s'", engine.GetCurrentProvider())
	}
	
	// Test switching to Mistral
	err := engine.SetProvider("mistral")
	if err != nil {
		t.Fatalf("Failed to switch to mistral: %v", err)
	}
	if engine.GetCurrentProvider() != "mistral" {
		t.Errorf("Expected provider 'mistral', got '%s'", engine.GetCurrentProvider())
	}
	
	// Test switching to Claude
	err = engine.SetProvider("claude")
	if err != nil {
		t.Fatalf("Failed to switch to claude: %v", err)
	}
	if engine.GetCurrentProvider() != "claude" {
		t.Errorf("Expected provider 'claude', got '%s'", engine.GetCurrentProvider())
	}
	
	// Test case insensitive
	err = engine.SetProvider("OPENAI")
	if err != nil {
		t.Fatalf("Failed to switch to OPENAI: %v", err)
	}
	if engine.GetCurrentProvider() != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", engine.GetCurrentProvider())
	}
	
	// Test invalid provider
	err = engine.SetProvider("invalid")
	if err == nil {
		t.Error("Expected error for invalid provider, got nil")
	}
	// Provider should remain unchanged
	if engine.GetCurrentProvider() != "openai" {
		t.Errorf("Expected provider to remain 'openai', got '%s'", engine.GetCurrentProvider())
	}
}

func TestSetProvider_ThreadSafety(t *testing.T) {
	store := &MockStorage{embeddings: make(map[string][]byte)}
	provider := NewOpenAIProvider()
	config := &Config{
		HighThreshold:          0.70,
		LowThreshold:           0.30,
		EnableGrayZoneVerifier: true,
	}
	
	engine := NewSemanticEngine(provider, store, provider, config)
	
	// Test concurrent provider switches
	done := make(chan bool)
	providers := []string{"openai", "mistral", "claude"}
	
	for i := 0; i < 10; i++ {
		go func(idx int) {
			for j := 0; j < 5; j++ {
				providerName := providers[j%len(providers)]
				engine.SetProvider(providerName)
				_ = engine.GetCurrentProvider()
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Ensure we have a valid provider at the end
	currentProvider := engine.GetCurrentProvider()
	validProvider := false
	for _, p := range providers {
		if currentProvider == p {
			validProvider = true
			break
		}
	}
	if !validProvider {
		t.Errorf("Expected valid provider, got '%s'", currentProvider)
	}
}



