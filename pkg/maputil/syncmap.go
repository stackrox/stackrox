package maputil

import "github.com/stackrox/rox/pkg/sync"

// SyncMap interface provides methods to access a map container.
type SyncMap[K comparable, V any] interface {
	Store(K, V)
	Load(K) (V, bool)
	Access(fn func(m *map[K]V))
	RAccess(fn func(m map[K]V))
}

type syncMapImpl[K comparable, V any] struct {
	data map[K]V
	mux  sync.RWMutex
}

// NewSyncMap returns a new synchronized map.
func NewSyncMap[K comparable, V any]() SyncMap[K, V] {
	return &syncMapImpl[K, V]{data: make(map[K]V)}
}

// Load returns the stored value by key.
func (m *syncMapImpl[K, V]) Load(k K) (V, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	v, ok := m.data[k]
	return v, ok
}

// Store inserts the value v to the map at the key k, or updates the value if the
// comparison predicate returns true.
func (m *syncMapImpl[K, V]) Store(k K, v V) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.data[k] = v
}

// Access gives protected read and write access to the internal map.
func (m *syncMapImpl[K, V]) Access(fn func(m *map[K]V)) {
	m.mux.Lock()
	defer m.mux.Unlock()
	fn(&m.data)
}

// RAccess gives protected read access to the internal map.
func (m *syncMapImpl[K, V]) RAccess(fn func(m map[K]V)) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	fn(m.data)
}
