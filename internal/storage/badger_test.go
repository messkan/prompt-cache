package storage

import (
	"context"
	"testing"
	"os"
	"time"
)

func TestBadgerStore(t *testing.T) {
	// Create a temporary directory for the test database
	tmpDir, err := os.MkdirTemp("", "badger_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewBadgerStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create BadgerStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	t.Run("Set and Get", func(t *testing.T) {
		key := "test_key"
		value := []byte("test_value")

		err := store.Set(ctx, key, value)
		if err != nil {
			t.Fatalf("Failed to set value: %v", err)
		}

		got, err := store.Get(ctx, key)
		if err != nil {
			t.Fatalf("Failed to get value: %v", err)
		}

		if string(got) != string(value) {
			t.Errorf("Got %s, want %s", string(got), string(value))
		}
	})

	t.Run("Get non-existent key", func(t *testing.T) {
		_, err := store.Get(ctx, "non_existent_key")
		if err == nil {
			t.Error("Expected error for non-existent key, got nil")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		key := "delete_test_key"
		value := []byte("delete_test_value")

		err := store.Set(ctx, key, value)
		if err != nil {
			t.Fatalf("Failed to set value: %v", err)
		}

		err = store.Delete(ctx, key)
		if err != nil {
			t.Fatalf("Failed to delete value: %v", err)
		}

		_, err = store.Get(ctx, key)
		if err == nil {
			t.Error("Expected error after delete, got nil")
		}
	})

	t.Run("GetAllEmbeddings", func(t *testing.T) {
		// Set some embeddings
		embKeys := []string{"emb:key1", "emb:key2", "emb:key3"}
		for _, k := range embKeys {
			err := store.Set(ctx, k, []byte("embedding_data"))
			if err != nil {
				t.Fatalf("Failed to set embedding: %v", err)
			}
		}

		embeddings, err := store.GetAllEmbeddings(ctx)
		if err != nil {
			t.Fatalf("Failed to get all embeddings: %v", err)
		}

		if len(embeddings) < 3 {
			t.Errorf("Expected at least 3 embeddings, got %d", len(embeddings))
		}

		for _, k := range embKeys {
			if _, ok := embeddings[k]; !ok {
				t.Errorf("Expected embedding %s not found", k)
			}
		}
	})

	t.Run("GetPrompt", func(t *testing.T) {
		key := "test_prompt_key"
		prompt := "What is the meaning of life?"

		err := store.Set(ctx, "prompt:"+key, []byte(prompt))
		if err != nil {
			t.Fatalf("Failed to set prompt: %v", err)
		}

		got, err := store.GetPrompt(ctx, key)
		if err != nil {
			t.Fatalf("Failed to get prompt: %v", err)
		}

		if got != prompt {
			t.Errorf("Got %s, want %s", got, prompt)
		}
	})

	t.Run("Count", func(t *testing.T) {
		count, err := store.Count(ctx)
		if err != nil {
			t.Fatalf("Failed to count: %v", err)
		}

		if count == 0 {
			t.Error("Expected count > 0")
		}
	})

	t.Run("GetAllKeys", func(t *testing.T) {
		keys, err := store.GetAllKeys(ctx, "emb:")
		if err != nil {
			t.Fatalf("Failed to get all keys: %v", err)
		}

		if len(keys) < 3 {
			t.Errorf("Expected at least 3 keys, got %d", len(keys))
		}
	})
}

func TestBadgerStore_SetWithTTL(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "badger_ttl_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewBadgerStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create BadgerStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	t.Run("SetWithTTL", func(t *testing.T) {
		key := "ttl_key"
		value := []byte("ttl_value")

		err := store.SetWithTTL(ctx, key, value, 1*time.Second)
		if err != nil {
			t.Fatalf("Failed to set with TTL: %v", err)
		}

		// Should exist immediately
		got, err := store.Get(ctx, key)
		if err != nil {
			t.Fatalf("Failed to get value: %v", err)
		}

		if string(got) != string(value) {
			t.Errorf("Got %s, want %s", string(got), string(value))
		}

		// Wait for TTL to expire
		time.Sleep(2 * time.Second)

		// Should not exist after TTL
		_, err = store.Get(ctx, key)
		if err == nil {
			t.Error("Expected error after TTL expiry, got nil")
		}
	})
}

func TestBadgerStore_DeleteByPrefix(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "badger_prefix_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewBadgerStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create BadgerStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	t.Run("DeleteByPrefix", func(t *testing.T) {
		// Set some keys with prefix
		for i := 0; i < 5; i++ {
			key := "prefix:key" + string(rune('0'+i))
			err := store.Set(ctx, key, []byte("value"))
			if err != nil {
				t.Fatalf("Failed to set value: %v", err)
			}
		}

		// Delete by prefix
		count, err := store.DeleteByPrefix(ctx, "prefix:")
		if err != nil {
			t.Fatalf("Failed to delete by prefix: %v", err)
		}

		if count != 5 {
			t.Errorf("Expected 5 deletions, got %d", count)
		}

		// Verify keys are deleted
		keys, err := store.GetAllKeys(ctx, "prefix:")
		if err != nil {
			t.Fatalf("Failed to get all keys: %v", err)
		}

		if len(keys) != 0 {
			t.Errorf("Expected 0 keys, got %d", len(keys))
		}
	})
}
