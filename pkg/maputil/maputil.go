package maputil

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

// ShallowClone creates a shallow clone of the given map.
func ShallowClone[K comparable, V any](inputMap map[K]V) map[K]V {
	cloned := make(map[K]V, len(inputMap))
	for k, v := range inputMap {
		cloned[k] = v
	}
	return cloned
}

// MapsIntersect returns true if there is at least one key-value pair that is present in both maps
// If both, or either maps are empty, it returns false
func MapsIntersect[K, V comparable](m1 map[K]V, m2 map[K]V) bool {
	if len(m2) == 0 {
		return false
	}
	if len(m1) > len(m2) {
		// Range over smaller map
		m1, m2 = m2, m1
	}
	for k, v := range m1 {
		if val, exists := m2[k]; exists {
			if v == val {
				return true
			}
		}
	}
	return false
}

// FastRMap is a thread-safe map from K to V that is optimized for read-heavy access patterns.
// Writes are expensive because it clones, mutates and replaces the map instead of an in-place addition.
// Use NewFastRMap to instantiate.
type FastRMap[K comparable, V any] struct {
	lock sync.RWMutex
	m    *map[K]V
}

// NewFastRMap returns an empty, ready-to-use, KeyTypeValueTypeFastRMap.
func NewFastRMap[K comparable, V any]() *FastRMap[K, V] {
	initialMap := make(map[K]V)
	return &FastRMap[K, V]{m: &initialMap}
}

func (m *FastRMap[K, V]) getCurrentMapPtr() *map[K]V {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.m
}

// GetMap returns a snapshot of the current map.
// Please don't hold on to it for too long because the map can be out-of-date.
// Further, do not mutate its contents UNLESS you know that you are the only
// user who will mutate the map.
func (m *FastRMap[K, V]) GetMap() map[K]V {
	currentPtr := m.getCurrentMapPtr()
	return *currentPtr
}

// DeleteMany deletes the specified keys.
func (m *FastRMap[K, V]) DeleteMany(keys ...K) {
	m.cloneAndMutate(func(clonedMap map[K]V) {
		for _, k := range keys {
			delete(clonedMap, k)
		}
	})
}

// SetMany merges the passed map into the current map.
// If there are key collisions, the passed-in map's elements take precedence.
func (m *FastRMap[K, V]) SetMany(elements map[K]V) {
	m.cloneAndMutate(func(clonedMap map[K]V) {
		for k, v := range elements {
			clonedMap[k] = v
		}
	})
}

// Set sets the value for the given key.
func (m *FastRMap[K, V]) Set(k K, v V) {
	m.cloneAndMutate(func(clonedMap map[K]V) {
		clonedMap[k] = v
	})
}

// Get retrieves the value for the given key.
func (m *FastRMap[K, V]) Get(k K) (V, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	val, exists := (*m.m)[k]
	return val, exists
}

// Delete deletes the value for the given key.
func (m *FastRMap[K, V]) Delete(k K) {
	m.cloneAndMutate(func(clonedMap map[K]V) {
		delete(clonedMap, k)
	})
}

// In order to block readers for as little time as possible, this implementation serializes writes in a more expensive way.
// We read the current pointer, clone the current map and mutate the cloned map. Then, just before replacing the current map pointer
// with a pointer to the cloned map,
// we acquire the lock, and check whether the current map pointer is the same as the one we started out with.
// If it is not (which means the map was mutated by another goroutine), we go back to the beginning.
// If it is, then we replace the map pointer with our cloned map.
func (m *FastRMap[K, V]) cloneAndMutate(mutateFunc func(clonedMap map[K]V)) {
	m.cloneAndMutateWithInitialPtr(m.getCurrentMapPtr(), mutateFunc)
}

func (m *FastRMap[K, V]) cloneAndMutateWithInitialPtr(initialMapPtr *map[K]V, mutateFunc func(clonedMap map[K]V)) {
	defer m.lock.Unlock()

	for {
		cloned := ShallowClone(*initialMapPtr)
		mutateFunc(cloned)

		m.lock.Lock()
		if m.m == initialMapPtr {
			m.m = &cloned
			return
		}

		// our work was for nothing, another goroutine beat us to the write!
		initialMapPtr = m.m
		concurrency.UnsafeUnlock(&m.lock)
	}
}
