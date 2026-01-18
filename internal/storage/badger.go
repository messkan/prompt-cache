package storage

import (
	"context"
	"time"

	"github.com/dgraph-io/badger/v4"
)

type BadgerStore struct {
	db *badger.DB
}

func NewBadgerStore(path string) (*BadgerStore, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil // Disable badger's default logging
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &BadgerStore{db: db}, nil
}

// SetWithTTL sets a key-value pair with a time-to-live
func (s *BadgerStore) SetWithTTL(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return s.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(key), value).WithTTL(ttl)
		return txn.SetEntry(e)
	})
}

// Count returns the total number of keys in the store
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

// GetAllKeys returns all keys with the given prefix
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

// DeleteByPrefix deletes all keys with the given prefix
func (s *BadgerStore) DeleteByPrefix(ctx context.Context, prefix string) (int, error) {
	var keysToDelete [][]byte
	
	// First, collect all keys to delete
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			keyCopy := make([]byte, len(it.Item().Key()))
			copy(keyCopy, it.Item().Key())
			keysToDelete = append(keysToDelete, keyCopy)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	// Then delete them
	for _, key := range keysToDelete {
		err := s.db.Update(func(txn *badger.Txn) error {
			return txn.Delete(key)
		})
		if err != nil {
			return 0, err
		}
	}

	return len(keysToDelete), nil
}

// Sync flushes all writes to disk
func (s *BadgerStore) Sync() error {
	return s.db.Sync()
}

// RunGC runs garbage collection on the database
func (s *BadgerStore) RunGC() error {
	return s.db.RunValueLogGC(0.5)
}

func (s *BadgerStore) Set(ctx context.Context, key string, value []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	})
}

func (s *BadgerStore) Get(ctx context.Context, key string) ([]byte, error) {
	var valCopy []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		valCopy, err = item.ValueCopy(nil)
		return err
	})
	return valCopy, err
}

func (s *BadgerStore) Delete(ctx context.Context, key string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

func (s *BadgerStore) GetAllEmbeddings(ctx context.Context) (map[string][]byte, error) {
	results := make(map[string][]byte)
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("emb:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			err := item.Value(func(v []byte) error {
				results[string(k)] = append([]byte{}, v...)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return results, err
}

func (s *BadgerStore) GetPrompt(ctx context.Context, key string) (string, error) {
	var valCopy []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("prompt:" + key))
		if err != nil {
			return err
		}
		valCopy, err = item.ValueCopy(nil)
		return err
	})
	return string(valCopy), err
}

func (s *BadgerStore) Close() {
	s.db.Close()
}
