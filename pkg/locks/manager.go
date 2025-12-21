package locks

import (
	"sync"
)

// KeyedMutex allows locking based on a string key (e.g. "task:123")
type KeyedMutex struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

// New creates a new KeyedMutex
func New() *KeyedMutex {
	return &KeyedMutex{
		locks: make(map[string]*sync.Mutex),
	}
}

// Lock acquires the lock for stepID
func (k *KeyedMutex) Lock(key string) {
	k.mu.Lock()
	if k.locks[key] == nil {
		k.locks[key] = &sync.Mutex{}
	}
	l := k.locks[key]
	k.mu.Unlock()
	l.Lock()
}

// Unlock releases the lock for stepID
func (k *KeyedMutex) Unlock(key string) {
	k.mu.Lock()
	if l, ok := k.locks[key]; ok {
		l.Unlock()
		// Optional: Clean up if no one is waiting?
		// For simplicity and avoiding races with pure deletion, we often leave it.
		// If memory is a concern with infinite IDs, we'd need ref counting.
		// Given task IDs are finite integers usually, this is fine.
	}
	k.mu.Unlock()
}
