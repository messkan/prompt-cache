package storage

import (
	"context"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/messkan/PromptCache/internal/cachecontext"
)

type BadgerStore struct {
	db *badger.DB
}

func NewBadgerStore(path string) (*BadgerStore, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &BadgerStore{db: db}, nil
}

func (s *BadgerStore) SetWithTTL(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	key = cachecontext.TypedKey(ctx, key)
	return s.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(key), value).WithTTL(ttl)
		return txn.SetEntry(e)
	})
}

func (s *BadgerStore) Count(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			count++
		}
		return nil
	})
	return count, err
}

func (s *BadgerStore) GetAllKeys(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			keys = append(keys, string(it.Item().Key()))
		}
		return nil
	})
	return keys, err
}

func (s *BadgerStore) DeleteByPrefix(ctx context.Context, prefix string) (int, error) {
	var keysToDelete [][]byte
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			keyCopy := append([]byte(nil), it.Item().Key()...)
			keysToDelete = append(keysToDelete, keyCopy)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	for _, key := range keysToDelete {
		if err := s.db.Update(func(txn *badger.Txn) error { return txn.Delete(key) }); err != nil {
			return 0, err
		}
	}
	return len(keysToDelete), nil
}

func (s *BadgerStore) Sync() error { return s.db.Sync() }
func (s *BadgerStore) RunGC() error { return s.db.RunValueLogGC(0.5) }

func (s *BadgerStore) Set(ctx context.Context, key string, value []byte) error {
	key = cachecontext.TypedKey(ctx, key)
	return s.db.Update(func(txn *badger.Txn) error { return txn.Set([]byte(key), value) })
}

func (s *BadgerStore) Get(ctx context.Context, key string) ([]byte, error) {
	var valCopy []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil
			}
			return err
		}
		valCopy, err = item.ValueCopy(nil)
		return err
	})
	return valCopy, err
}

func (s *BadgerStore) Delete(ctx context.Context, key string) error {
	key = cachecontext.TypedKey(ctx, key)
	return s.db.Update(func(txn *badger.Txn) error { return txn.Delete([]byte(key)) })
}

func (s *BadgerStore) GetAllEmbeddings(ctx context.Context) (map[string][]byte, error) {
	results := make(map[string][]byte)
	prefixText := cachecontext.EmbeddingPrefix(ctx)
	prefix := []byte(prefixText)
	defaultNamespace := cachecontext.NamespaceID(ctx) == ""

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			physicalKey := string(item.Key())
			if defaultNamespace && cachecontext.IsNamespacedEmbedding(physicalKey) {
				continue
			}
			logicalKey := cachecontext.NormalizeEmbeddingKey(ctx, physicalKey)
			if err := item.Value(func(v []byte) error {
				results[logicalKey] = append([]byte{}, v...)
				return nil
			}); err != nil {
				return err
			}
		}
		return nil
	})
	return results, err
}

func (s *BadgerStore) GetPrompt(ctx context.Context, key string) (string, error) {
	physicalKey := cachecontext.TypedKey(ctx, "prompt:"+strings.TrimPrefix(key, "prompt:"))
	var valCopy []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(physicalKey))
		if err != nil {
			return err
		}
		valCopy, err = item.ValueCopy(nil)
		return err
	})
	return string(valCopy), err
}

func (s *BadgerStore) Close() { _ = s.db.Close() }
