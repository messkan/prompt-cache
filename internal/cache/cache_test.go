package cache

import (
	"context"
	"testing"
	"time"
)

// MockStorage implements storage.Storage for testing
type MockStorage struct {
	data map[string][]byte
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		data: make(map[string][]byte),
	}
}

func (m *MockStorage) Set(ctx context.Context, key string, value []byte) error {
	m.data[key] = value
	return nil
}

func (m *MockStorage) Get(ctx context.Context, key string) ([]byte, error) {
	val, ok := m.data[key]
	if !ok {
		return nil, nil // Simulate miss
	}
	return val, nil
}

func (m *MockStorage) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *MockStorage) GetAllEmbeddings(ctx context.Context) (map[string][]byte, error) {
	return nil, nil
}

func (m *MockStorage) GetPrompt(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (m *MockStorage) CountEmbeddings(ctx context.Context) (int64, error) {
	var count int64
	for k := range m.data {
		if len(k) >= 4 && k[:4] == "emb:" {
			count++
		}
	}
	return count, nil
}

func (m *MockStorage) Close() {}

func TestCache_SetAndGet(t *testing.T) {
	store := NewMockStorage()
	c := NewCache(store)
	ctx := context.Background()

	key := "test-key"
	response := []byte("test-response")
	ttl := 1 * time.Hour

	// Test Set
	err := c.Set(ctx, key, response, ttl)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test Get (Hit)
	got, found, err := c.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("Expected cache hit, got miss")
	}
	if string(got) != string(response) {
		t.Errorf("Get = %s, want %s", got, response)
	}
}

func TestCache_Expiration(t *testing.T) {
	store := NewMockStorage()
	c := NewCache(store)
	ctx := context.Background()

	key := "expired-key"
	response := []byte("expired-response")
	ttl := -1 * time.Hour // Already expired

	err := c.Set(ctx, key, response, ttl)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test Get (Expired)
	_, found, err := c.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatal("Expected cache miss (expired), got hit")
	}
}

func TestCache_Miss(t *testing.T) {
	store := NewMockStorage()
	c := NewCache(store)
	ctx := context.Background()

	_, found, err := c.Get(ctx, "non-existent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatal("Expected cache miss, got hit")
	}
}
