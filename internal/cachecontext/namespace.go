package cachecontext

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const Header = "X-Cache-Namespace"

type namespaceKey struct{}

// WithNamespace stores a stable, non-reversible identifier for a caller-provided
// cache partition in the request context. An empty namespace preserves legacy
// single-cache behavior.
func WithNamespace(ctx context.Context, raw string) context.Context {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ctx
	}
	h := sha256.Sum256([]byte(raw))
	id := hex.EncodeToString(h[:])
	return context.WithValue(ctx, namespaceKey{}, id)
}

// NamespaceID returns the hashed namespace identifier attached to ctx.
func NamespaceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := ctx.Value(namespaceKey{}).(string)
	return id
}

// CacheKey scopes a response-cache key to the current namespace.
func CacheKey(ctx context.Context, key string) string {
	id := NamespaceID(ctx)
	if id == "" {
		return key
	}
	return "ns:" + id + ":" + key
}

// TypedKey scopes prompt:/emb: storage keys while preserving their type prefix.
func TypedKey(ctx context.Context, key string) string {
	id := NamespaceID(ctx)
	if id == "" {
		return key
	}
	for _, prefix := range []string{"prompt:", "emb:"} {
		if strings.HasPrefix(key, prefix) {
			return prefix + "ns:" + id + ":" + strings.TrimPrefix(key, prefix)
		}
	}
	return key
}

// EmbeddingPrefix returns the physical Badger prefix for the current namespace.
func EmbeddingPrefix(ctx context.Context) string {
	id := NamespaceID(ctx)
	if id == "" {
		return "emb:"
	}
	return "emb:ns:" + id + ":"
}

// NormalizeEmbeddingKey maps a physical embedding key back to the legacy
// logical shape expected by the semantic engine (emb:<hash>).
func NormalizeEmbeddingKey(ctx context.Context, key string) string {
	id := NamespaceID(ctx)
	if id == "" {
		return key
	}
	prefix := "emb:ns:" + id + ":"
	return "emb:" + strings.TrimPrefix(key, prefix)
}

// IsNamespacedEmbedding reports whether a physical key belongs to any explicit
// namespace rather than the legacy/default partition.
func IsNamespacedEmbedding(key string) bool {
	return strings.HasPrefix(key, "emb:ns:")
}
