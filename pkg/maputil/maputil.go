package maputil

import (
	"github.com/mauricelam/genny/generic"
	"github.com/stackrox/stackrox/pkg/sync"
)

//go:generate genny -in=$GOFILE -out=gen-string-$GOFILE gen "KeyType=string ValueType=string"

// KeyType represents a generic type that we want to use as a map key.
type KeyType generic.Type

// ValueType represents a generic type that we want to use as a map value.
type ValueType generic.Type

// CloneKeyTypeValueTypeMap clones a map of the given type.
func CloneKeyTypeValueTypeMap(inputMap map[KeyType]ValueType) map[KeyType]ValueType {
	cloned := make(map[KeyType]ValueType, len(inputMap))
	for k, v := range inputMap {
		cloned[k] = v
	}
	return cloned
}

// KeyTypeValueTypeMapsEqual compares if two maps of the given type are equal.
func KeyTypeValueTypeMapsEqual(a, b map[KeyType]ValueType) bool {
	if len(a) != len(b) {
		return false
	}
	for k, aV := range a {
		if bV, ok := b[k]; !ok || aV != bV {
			return false
		}
	}
	return true
}

// KeyTypeValueTypeFastRMap is a thread-safe map from KeyType to ValueType that is optimized for read-heavy access patterns.
// Writes are expensive because it clones, mutates and replaces the map instead of an in-place addition.
// Use NewKeyTypeValueType to instantiate.
type KeyTypeValueTypeFastRMap struct {
	lock sync.RWMutex
	m    *map[KeyType]ValueType
}

// NewKeyTypeValueTypeFastRMap returns an empty, read-to-use, KeyTypeValueTypeFastRMap.
func NewKeyTypeValueTypeFastRMap() KeyTypeValueTypeFastRMap {
	initialMap := make(map[KeyType]ValueType)
	return KeyTypeValueTypeFastRMap{m: &initialMap}
}

func (m *KeyTypeValueTypeFastRMap) getCurrentMapPtr() *map[KeyType]ValueType {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.m
}

// GetMap returns a snapshot of the current map.
// Please don't hold on to it for too long because the map can be out-of-date.
// Further, do not mutate its contents UNLESS you know that you are the only
// user who will mutate the map.
func (m *KeyTypeValueTypeFastRMap) GetMap() map[KeyType]ValueType {
	currentPtr := m.getCurrentMapPtr()
	return *currentPtr
}

// DeleteMany deletes the specified keys.
func (m *KeyTypeValueTypeFastRMap) DeleteMany(keys ...KeyType) {
	m.cloneAndMutate(func(clonedMap map[KeyType]ValueType) {
		for _, k := range keys {
			delete(clonedMap, k)
		}
	})
}

// SetMany merges the passed map into the current map.
// If there are key collisions, the passed-in map's elements take precedence.
func (m *KeyTypeValueTypeFastRMap) SetMany(elements map[KeyType]ValueType) {
	m.cloneAndMutate(func(clonedMap map[KeyType]ValueType) {
		for k, v := range elements {
			clonedMap[k] = v
		}
	})
}

// Set sets the value for the given key.
func (m *KeyTypeValueTypeFastRMap) Set(k KeyType, v ValueType) {
	m.cloneAndMutate(func(clonedMap map[KeyType]ValueType) {
		clonedMap[k] = v
	})
}

// Get retrieves the value for the given key.
func (m *KeyTypeValueTypeFastRMap) Get(k KeyType) (ValueType, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	val, exists := (*m.m)[k]
	return val, exists
}

// Delete deletes the value for the given key.
func (m *KeyTypeValueTypeFastRMap) Delete(k KeyType) {
	m.cloneAndMutate(func(clonedMap map[KeyType]ValueType) {
		delete(clonedMap, k)
	})
}

// In order to block readers for as little time as possible, this implementation serializes writes in a more expensive way.
// We read the current pointer, clone the current map and mutate the cloned map. Then, just before replacing the current map pointer
// with a pointer to the cloned map,
// we acquire the lock, and check whether the current map pointer is the same as the one we started out with.
// If it is not (which means the map was mutated by another goroutine), we go back to the beginning.
// If it is, then we replace the map pointer with our cloned map.
func (m *KeyTypeValueTypeFastRMap) cloneAndMutate(mutateFunc func(clonedMap map[KeyType]ValueType)) {
	m.cloneAndMutateWithInitialPtr(m.getCurrentMapPtr(), mutateFunc)
}

func (m *KeyTypeValueTypeFastRMap) cloneAndMutateWithInitialPtr(initialMapPtr *map[KeyType]ValueType, mutateFunc func(clonedMap map[KeyType]ValueType)) {
	defer m.lock.Unlock()

	for {
		cloned := CloneKeyTypeValueTypeMap(*initialMapPtr)
		mutateFunc(cloned)

		m.lock.Lock()
		if m.m == initialMapPtr {
			m.m = &cloned
			return
		}

		// our work was for nothing, another goroutine beat us to the write!
		initialMapPtr = m.m
		m.lock.Unlock()
	}
}
