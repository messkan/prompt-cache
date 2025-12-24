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
	cfg := &Config{
		GrayZoneFallbackModel:   getEnv("GRAY_ZONE_FALLBACK_MODEL", "gpt-4o-mini"),
		HighSimilarityThreshold: getEnvFloat32("HIGH_SIMILARITY_THRESHOLD", 0.95),
		LowSimilarityThreshold:  getEnvFloat32("LOW_SIMILARITY_THRESHOLD", 0.80),
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

// getEnvFloat32 reads a float32 environment variable or returns a default value
func getEnvFloat32(key string, defaultValue float32) float32 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 32); err == nil {
			return float32(parsed)
		}
	}
	return defaultValue
}
