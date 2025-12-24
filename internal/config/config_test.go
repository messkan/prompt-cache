package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("GRAY_ZONE_FALLBACK_MODEL")
	os.Unsetenv("HIGH_SIMILARITY_THRESHOLD")
	os.Unsetenv("LOW_SIMILARITY_THRESHOLD")
	
	cfg := Load()
	
	if cfg.GrayZoneFallbackModel != "gpt-4o-mini" {
		t.Errorf("Expected GrayZoneFallbackModel to be 'gpt-4o-mini', got '%s'", cfg.GrayZoneFallbackModel)
	}
	
	if cfg.HighSimilarityThreshold != 0.95 {
		t.Errorf("Expected HighSimilarityThreshold to be 0.95, got %f", cfg.HighSimilarityThreshold)
	}
	
	if cfg.LowSimilarityThreshold != 0.80 {
		t.Errorf("Expected LowSimilarityThreshold to be 0.80, got %f", cfg.LowSimilarityThreshold)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	// Set custom environment variables
	os.Setenv("GRAY_ZONE_FALLBACK_MODEL", "gpt-4")
	os.Setenv("HIGH_SIMILARITY_THRESHOLD", "0.98")
	os.Setenv("LOW_SIMILARITY_THRESHOLD", "0.75")
	
	// Cleanup after test
	defer func() {
		os.Unsetenv("GRAY_ZONE_FALLBACK_MODEL")
		os.Unsetenv("HIGH_SIMILARITY_THRESHOLD")
		os.Unsetenv("LOW_SIMILARITY_THRESHOLD")
	}()
	
	cfg := Load()
	
	if cfg.GrayZoneFallbackModel != "gpt-4" {
		t.Errorf("Expected GrayZoneFallbackModel to be 'gpt-4', got '%s'", cfg.GrayZoneFallbackModel)
	}
	
	if cfg.HighSimilarityThreshold != 0.98 {
		t.Errorf("Expected HighSimilarityThreshold to be 0.98, got %f", cfg.HighSimilarityThreshold)
	}
	
	if cfg.LowSimilarityThreshold != 0.75 {
		t.Errorf("Expected LowSimilarityThreshold to be 0.75, got %f", cfg.LowSimilarityThreshold)
	}
}

func TestLoad_InvalidThresholds(t *testing.T) {
	// Set invalid threshold values (should fall back to defaults)
	os.Setenv("HIGH_SIMILARITY_THRESHOLD", "invalid")
	os.Setenv("LOW_SIMILARITY_THRESHOLD", "not_a_number")
	
	// Cleanup after test
	defer func() {
		os.Unsetenv("HIGH_SIMILARITY_THRESHOLD")
		os.Unsetenv("LOW_SIMILARITY_THRESHOLD")
	}()
	
	cfg := Load()
	
	// Should fall back to defaults when parsing fails
	if cfg.HighSimilarityThreshold != 0.95 {
		t.Errorf("Expected HighSimilarityThreshold to fall back to 0.95, got %f", cfg.HighSimilarityThreshold)
	}
	
	if cfg.LowSimilarityThreshold != 0.80 {
		t.Errorf("Expected LowSimilarityThreshold to fall back to 0.80, got %f", cfg.LowSimilarityThreshold)
	}
}

func TestGetEnv(t *testing.T) {
	key := "TEST_ENV_VAR"
	defaultVal := "default_value"
	customVal := "custom_value"
	
	// Test with no env var set
	os.Unsetenv(key)
	result := getEnv(key, defaultVal)
	if result != defaultVal {
		t.Errorf("Expected '%s', got '%s'", defaultVal, result)
	}
	
	// Test with env var set
	os.Setenv(key, customVal)
	defer os.Unsetenv(key)
	
	result = getEnv(key, defaultVal)
	if result != customVal {
		t.Errorf("Expected '%s', got '%s'", customVal, result)
	}
}

func TestGetEnvFloat32(t *testing.T) {
	key := "TEST_FLOAT_VAR"
	defaultVal := float32(0.5)
	customVal := float32(0.75)
	
	// Test with no env var set
	os.Unsetenv(key)
	result := getEnvFloat32(key, defaultVal)
	if result != defaultVal {
		t.Errorf("Expected %f, got %f", defaultVal, result)
	}
	
	// Test with valid env var set
	os.Setenv(key, "0.75")
	defer os.Unsetenv(key)
	
	result = getEnvFloat32(key, defaultVal)
	if result != customVal {
		t.Errorf("Expected %f, got %f", customVal, result)
	}
	
	// Test with invalid env var (should return default)
	os.Setenv(key, "invalid")
	result = getEnvFloat32(key, defaultVal)
	if result != defaultVal {
		t.Errorf("Expected %f, got %f", defaultVal, result)
	}
}
