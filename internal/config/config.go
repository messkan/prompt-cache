package config

import (
	"os"
	"strconv"
)

// Config holds the application configuration
type Config struct {
	// Gray zone fallback model for similarity verification
	GrayZoneFallbackModel string
	
	// High similarity threshold - above this, cache hit is guaranteed
	HighSimilarityThreshold float32
	
	// Low similarity threshold - below this, cache miss is guaranteed
	LowSimilarityThreshold float32
}

// Load reads configuration from environment variables with sensible defaults
func Load() *Config {
	highThreshold := float32(0.70)
	if value := os.Getenv("HIGH_SIMILARITY_THRESHOLD"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 32); err == nil {
			highThreshold = float32(parsed)
		}
	}
	
	lowThreshold := float32(0.30)
	if value := os.Getenv("LOW_SIMILARITY_THRESHOLD"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 32); err == nil {
			lowThreshold = float32(parsed)
		}
	}
	
	cfg := &Config{
		GrayZoneFallbackModel:   getEnv("GRAY_ZONE_FALLBACK_MODEL", "gpt-4o-mini"),
		HighSimilarityThreshold: highThreshold,
		LowSimilarityThreshold:  lowThreshold,
	}
	
	// Validate thresholds
	if cfg.HighSimilarityThreshold <= cfg.LowSimilarityThreshold {
		// If invalid, reset to sensible defaults
		cfg.HighSimilarityThreshold = 0.70
		cfg.LowSimilarityThreshold = 0.30
	}
	
	return cfg
}

// getEnv reads an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
