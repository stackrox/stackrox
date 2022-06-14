package concurrency

import (
	"hash/fnv"

	"github.com/stackrox/rox/pkg/sync"
)

// A KeyedRWMutex allows callers to synchronize a block of code based on a key.
// For a given key, only one goroutine will be allowed to obtain the mutex at a time.
// Goroutines with different keys will be allowed to run in parallel;
// (although it is not guaranteed, since there might be a hash collision).
// Callers can specify the poolSize when creating a KeyedRWMutex, depending on how frequently
// they are okay with dealing with hash collisions.
// The zero value is NOT ready-to-use. Please use the NewKeyedRWMutex function.
type KeyedRWMutex struct {
	mutexPool []sync.RWMutex
}

// NewKeyedRWMutex returns a ready-to-use KeyedRWMutex.
// The chance of two random keys colliding can be assumed
// to be 1/poolSize; set poolSize accordingly.
func NewKeyedRWMutex(poolSize uint32) *KeyedRWMutex {
	mutexPool := make([]sync.RWMutex, poolSize)
	return &KeyedRWMutex{
		mutexPool: mutexPool,
	}
}

func (k *KeyedRWMutex) indexFromKey(key string) uint32 {
	h := fnv.New32()
	// Write never returns an error.
	_, _ = h.Write([]byte(key))
	return h.Sum32() % uint32(len(k.mutexPool))
}

// Lock locks the mutex corresponding to the given key.
func (k *KeyedRWMutex) Lock(key string) {
	k.mutexPool[k.indexFromKey(key)].Lock()
}

// RLock acquires a read lock for the mutex corresponding to the given key
func (k *KeyedRWMutex) RLock(key string) {
	k.mutexPool[k.indexFromKey(key)].RLock()
}

// Unlock unlocks the mutex with the given key.
// It is the caller's responsibility to ensure that
// the mutex was locked first; it is a runtime error otherwise.
func (k *KeyedRWMutex) Unlock(key string) {
	k.mutexPool[k.indexFromKey(key)].Unlock()
}

// RUnlock releases a read lock for the mutex corresponding to the given key.
// It is the caller's responsibility to ensure that
// the mutex was locked first; it is a runtime error otherwise.
func (k *KeyedRWMutex) RUnlock(key string) {
	k.mutexPool[k.indexFromKey(key)].RUnlock()
}

// DoWithLock calls the given function while holding the lock. The lock is acquired in a safe manner, making sure
// it's released even if `do` panics.
func (k *KeyedRWMutex) DoWithLock(key string, do func()) {
	m := &k.mutexPool[k.indexFromKey(key)]
	m.Lock()
	defer m.Unlock()
	do()
}

// DoWithRLock calls the given function while holding a read lock. The lock is acquired in a safe manner, making sure
// it's released even if `do` panics.
func (k *KeyedRWMutex) DoWithRLock(key string, do func()) {
	m := &k.mutexPool[k.indexFromKey(key)]
	m.RLock()
	defer m.RUnlock()
	do()
}

// DoStatusWithLock calls the given function while holding the lock, and passes through any error returned by it. The
// lock is acquired in a safe manner, making sure it's released even if `do` panics.
func (k *KeyedRWMutex) DoStatusWithLock(key string, do func() error) error {
	m := &k.mutexPool[k.indexFromKey(key)]
	m.Lock()
	defer m.Unlock()

	return do()
}

// DoStatusWithRLock calls the given function while holding a read lock, and passes through any error returned by it. The
// lock is acquired in a safe manner, making sure it's released even if `do` panics.
func (k *KeyedRWMutex) DoStatusWithRLock(key string, do func() error) error {
	m := &k.mutexPool[k.indexFromKey(key)]
	m.RLock()
	defer m.RUnlock()

	return do()
}
