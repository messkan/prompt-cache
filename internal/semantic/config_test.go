package semantic

import (
	"testing"
)

func newTestEngine() *SemanticEngine {
	provider := &MockProvider{}
	store := &MockStorage{embeddings: map[string][]byte{}}
	cfg := &Config{
		HighThreshold:          0.9,
		LowThreshold:           0.7,
		EnableGrayZoneVerifier: false,
	}
	return NewSemanticEngine(provider, store, provider, cfg)
}

func TestUpdateThresholds_Valid(t *testing.T) {
	se := newTestEngine()
	enable := true
	if err := se.UpdateThresholds(0.95, 0.80, &enable); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cfg := se.GetConfig()
	if cfg["high_threshold"].(float32) != 0.95 {
		t.Errorf("high not updated: %v", cfg["high_threshold"])
	}
	if cfg["low_threshold"].(float32) != 0.80 {
		t.Errorf("low not updated: %v", cfg["low_threshold"])
	}
	if cfg["enable_gray_zone_verifier"].(bool) != true {
		t.Errorf("verifier flag not updated")
	}
}

func TestUpdateThresholds_HighNotGreaterThanLow(t *testing.T) {
	se := newTestEngine()
	if err := se.UpdateThresholds(0.5, 0.6, nil); err == nil {
		t.Fatal("expected error when high <= low")
	}
}

func TestUpdateThresholds_OutOfRange(t *testing.T) {
	se := newTestEngine()
	if err := se.UpdateThresholds(1.5, 0.5, nil); err == nil {
		t.Fatal("expected error when high > 1.0")
	}
	if err := se.UpdateThresholds(0.9, -0.1, nil); err == nil {
		t.Fatal("expected error when low < 0")
	}
}

func TestUpdateThresholds_NilGrayZonePreserves(t *testing.T) {
	se := newTestEngine()
	enable := true
	_ = se.UpdateThresholds(0.95, 0.80, &enable)
	// Now update without changing the flag
	if err := se.UpdateThresholds(0.96, 0.81, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cfg := se.GetConfig()
	if cfg["enable_gray_zone_verifier"].(bool) != true {
		t.Errorf("expected verifier flag to be preserved as true")
	}
}

func TestGetConfig_ReturnsCurrentValues(t *testing.T) {
	se := newTestEngine()
	cfg := se.GetConfig()
	if cfg["high_threshold"].(float32) != 0.9 {
		t.Errorf("unexpected high: %v", cfg["high_threshold"])
	}
	if cfg["low_threshold"].(float32) != 0.7 {
		t.Errorf("unexpected low: %v", cfg["low_threshold"])
	}
}
