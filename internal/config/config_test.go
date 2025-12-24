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
	
	if cfg.HighSimilarityThreshold != 0.70 {
		t.Errorf("Expected HighSimilarityThreshold to be 0.70, got %f", cfg.HighSimilarityThreshold)
	}
	
	if cfg.LowSimilarityThreshold != 0.30 {
		t.Errorf("Expected LowSimilarityThreshold to be 0.30, got %f", cfg.LowSimilarityThreshold)
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
	if cfg.HighSimilarityThreshold != 0.70 {
		t.Errorf("Expected HighSimilarityThreshold to fall back to 0.70, got %f", cfg.HighSimilarityThreshold)
	}
	
	if cfg.LowSimilarityThreshold != 0.30 {
		t.Errorf("Expected LowSimilarityThreshold to fall back to 0.30, got %f", cfg.LowSimilarityThreshold)
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

func TestLoad_InvalidThresholdOrdering(t *testing.T) {
	// Set thresholds where high is less than or equal to low (invalid)
	os.Setenv("HIGH_SIMILARITY_THRESHOLD", "0.20")
	os.Setenv("LOW_SIMILARITY_THRESHOLD", "0.90")
	
	// Cleanup after test
	defer func() {
		os.Unsetenv("HIGH_SIMILARITY_THRESHOLD")
		os.Unsetenv("LOW_SIMILARITY_THRESHOLD")
	}()
	
	cfg := Load()
	
	// Should fall back to defaults when ordering is invalid
	if cfg.HighSimilarityThreshold != 0.70 {
		t.Errorf("Expected HighSimilarityThreshold to fall back to 0.70, got %f", cfg.HighSimilarityThreshold)
	}
	
	if cfg.LowSimilarityThreshold != 0.30 {
		t.Errorf("Expected LowSimilarityThreshold to fall back to 0.30, got %f", cfg.LowSimilarityThreshold)
	}
	
	// Test equal thresholds (also invalid)
	os.Setenv("HIGH_SIMILARITY_THRESHOLD", "0.50")
	os.Setenv("LOW_SIMILARITY_THRESHOLD", "0.50")
	
	cfg = Load()
	
	if cfg.HighSimilarityThreshold != 0.70 {
		t.Errorf("Expected HighSimilarityThreshold to fall back to 0.70, got %f", cfg.HighSimilarityThreshold)
	}
	
	if cfg.LowSimilarityThreshold != 0.30 {
		t.Errorf("Expected LowSimilarityThreshold to fall back to 0.30, got %f", cfg.LowSimilarityThreshold)
	}
}
