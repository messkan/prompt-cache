package storage

import (
	"context"
)

type Storage interface {
	Set(ctx context.Context, key string, value []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	GetAllEmbeddings(ctx context.Context) (map[string][]byte, error)
	GetPrompt(ctx context.Context, key string) (string, error)
	Close()
}
