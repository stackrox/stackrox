package concurrency

import (
	"hash/fnv"

	"github.com/stackrox/stackrox/pkg/sync"
)

// A KeyedMutex allows callers to synchronize a block of code based on a key.
// For a given key, only one goroutine will be allowed to obtain the mutex at a time.
// Goroutines with different keys will be allowed to run in parallel;
// (although it is not guaranteed, since there might be a hash collision).
// Callers can specify the poolSize when creating a KeyedMutex, depending on how frequently
// they are okay with dealing with hash collisions.
// The zero value is NOT ready-to-use. Please use the NewKeyedMutex function.
type KeyedMutex struct {
	mutexPool []sync.Mutex
}

// NewKeyedMutex returns a ready-to-use KeyedMutex.
// The chance of two random keys colliding can be assumed
// to be 1/poolSize; set poolSize accordingly.
func NewKeyedMutex(poolSize uint32) *KeyedMutex {
	mutexPool := make([]sync.Mutex, poolSize)
	return &KeyedMutex{
		mutexPool: mutexPool,
	}
}

func (k *KeyedMutex) indexFromKey(key string) uint32 {
	h := fnv.New32()
	// Write never returns an error.
	_, _ = h.Write([]byte(key))
	return h.Sum32() % uint32(len(k.mutexPool))
}

// Lock locks the mutex corresponding to the given key.
func (k *KeyedMutex) Lock(key string) {
	k.mutexPool[k.indexFromKey(key)].Lock()
}

// Unlock unlocks the mutex with the given key.
// It is the caller's responsibility to ensure that
// the mutex was locked first; it is a runtime error otherwise.
func (k *KeyedMutex) Unlock(key string) {
	k.mutexPool[k.indexFromKey(key)].Unlock()
}

// DoWithLock calls the given function while holding the lock. The lock is acquired in a safe manner, making sure
// it's released even if `do` panics.
func (k *KeyedMutex) DoWithLock(key string, do func()) {
	m := &k.mutexPool[k.indexFromKey(key)]
	m.Lock()
	defer m.Unlock()
	do()
}

// DoStatusWithLock calls the given function while holding the lock, and passes through any error returned by it. The
// lock is acquired in a safe manner, making sure it's released even if `do` panics.
func (k *KeyedMutex) DoStatusWithLock(key string, do func() error) error {
	m := &k.mutexPool[k.indexFromKey(key)]
	m.Lock()
	defer m.Unlock()

	return do()
}
