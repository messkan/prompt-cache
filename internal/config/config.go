package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Server settings
	Port            string
	RequestMaxBytes int64

	// Storage settings
	StoragePath string

	// Cache settings
	CacheTTL   time.Duration
	MaxEntries int

	// Semantic thresholds
	HighThreshold          float32
	LowThreshold           float32
	EnableGrayZoneVerifier bool

	// HTTP client settings
	HTTPTimeout       time.Duration
	HTTPMaxRetries    int
	HTTPRetryBaseWait time.Duration

	// Provider settings
	EmbeddingProvider string

	// Model overrides (optional)
	OpenAIEmbedModel    string
	OpenAIVerifyModel   string
	MistralEmbedModel   string
	MistralVerifyModel  string
	ClaudeModel         string
	ClaudeVerifyModel   string
	VoyageEmbedModel    string

	// Auth settings
	AuthToken string
}

// Load loads configuration from environment variables with sensible defaults
func Load() *Config {
	cfg := &Config{
		// Server defaults
		Port:            getEnvOrDefault("PORT", "8080"),
		RequestMaxBytes: getEnvInt64OrDefault("REQUEST_MAX_BYTES", 1*1024*1024), // 1MB

		// Storage defaults
		StoragePath: getEnvOrDefault("STORAGE_PATH", "./badger_data"),

		// Cache defaults
		CacheTTL:   getEnvDurationOrDefault("CACHE_TTL_HOURS", 24) * time.Hour,
		MaxEntries: getEnvIntOrDefault("CACHE_MAX_ENTRIES", 100000),

		// Semantic thresholds
		HighThreshold:          getEnvFloat32OrDefault("CACHE_HIGH_THRESHOLD", 0.70),
		LowThreshold:           getEnvFloat32OrDefault("CACHE_LOW_THRESHOLD", 0.30),
		EnableGrayZoneVerifier: getEnvBoolOrDefault("ENABLE_GRAY_ZONE_VERIFIER", true),

		// HTTP client settings
		HTTPTimeout:       getEnvDurationOrDefault("HTTP_TIMEOUT_SECONDS", 30) * time.Second,
		HTTPMaxRetries:    getEnvIntOrDefault("HTTP_MAX_RETRIES", 3),
		HTTPRetryBaseWait: getEnvDurationOrDefault("HTTP_RETRY_BASE_WAIT_MS", 500) * time.Millisecond,

		// Provider settings
		EmbeddingProvider: getEnvOrDefault("EMBEDDING_PROVIDER", "openai"),

		// Auth settings
		AuthToken: getEnvOrDefault("API_AUTH_TOKEN", ""),

		// Model overrides
		OpenAIEmbedModel:   getEnvOrDefault("OPENAI_EMBED_MODEL", "text-embedding-3-small"),
		OpenAIVerifyModel:  getEnvOrDefault("OPENAI_VERIFY_MODEL", "gpt-4o-mini"),
		MistralEmbedModel:  getEnvOrDefault("MISTRAL_EMBED_MODEL", "mistral-embed"),
		MistralVerifyModel: getEnvOrDefault("MISTRAL_VERIFY_MODEL", "mistral-small-latest"),
		ClaudeModel:        getEnvOrDefault("CLAUDE_MODEL", "claude-3-opus-20240229"),
		ClaudeVerifyModel:  getEnvOrDefault("CLAUDE_VERIFY_MODEL", "claude-3-haiku-20240307"),
		VoyageEmbedModel:   getEnvOrDefault("VOYAGE_EMBED_MODEL", "voyage-3"),
	}

	// Ensure high threshold is greater than low threshold
	if cfg.HighThreshold <= cfg.LowThreshold {
		cfg.HighThreshold = 0.70
		cfg.LowThreshold = 0.30
	}

	return cfg
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvIntOrDefault(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvInt64OrDefault(key string, defaultVal int64) int64 {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvFloat32OrDefault(key string, defaultVal float32) float32 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 32); err == nil && f > 0 && f <= 1.0 {
			return float32(f)
		}
	}
	return defaultVal
}

func getEnvBoolOrDefault(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		return val == "true" || val == "1" || val == "yes"
	}
	return defaultVal
}

func getEnvDurationOrDefault(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			return time.Duration(i)
		}
	}
	return defaultVal
}
