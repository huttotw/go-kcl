package kcl

import (
	"fmt"
	"sync"
)

// Store is an interface that defines how we will persist and retrieve
// the shard iterator. It is important to keep track of the shard iterator
// so that we know our position in the stream. The implementation of Store
// must be safe for concurrent use.
type Store interface {
	// GetShardIterator will get the current iterator for the shard. This
	// tells Amazon where we want to start reading records from.
	GetShardIterator(stream, shard string) (string, error)

	// UpdateShardIterator will update the position in the shard so that
	// on the next tick of our listener, we read records from the latest
	// position.
	UpdateShardIterator(stream, shard, iterator string) error
}

// LocalStore implements Store using a local map. This store is not usable
// if your application is running in multiple containers.
type LocalStore struct {
	m     map[string]string
	mutex sync.Mutex
}

// NewLocalStore will create a pointer to a local store that can keep track
// of our shard iterators.
func NewLocalStore() *LocalStore {
	s := LocalStore{
		m: make(map[string]string),
	}

	return &s
}

// UpdateShardIterator will use the stream-shard combination as the key, and store
// the iterator that corresponds to it. Updates require a mutex lock so that two
// goroutines are not trying to update it at the same time.
func (s *LocalStore) UpdateShardIterator(stream, shard, iterator string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	key := fmt.Sprintf("%s-%s", stream, shard)
	s.m[key] = iterator
	return nil
}

// GetShardIterator will get the shard iterator that corresponds to the stream-shard
// combination. We do not require a lock here, because we are simply reading.
func (s *LocalStore) GetShardIterator(stream, shard string) (string, error) {
	key := fmt.Sprintf("%s-%s", stream, shard)
	return s.m[key], nil
}
