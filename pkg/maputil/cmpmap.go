package maputil

import "github.com/stackrox/rox/pkg/sync"

type cmpmap[K comparable, V any, C func(a, b V) bool] struct {
	data map[K]V
	cmp  C
	mux  sync.RWMutex
}

// NewCmpMap creates a new comparing map, which only updates the stored values if the
// cmp predicate returns true when provided with the existing and new values.
// Example:
//
//	NewCmpMap[string](Max[int])
func NewCmpMap[K comparable, V any, Cmp func(a, b V) bool](cmp Cmp) *cmpmap[K, V, Cmp] {
	return &cmpmap[K, V, Cmp]{cmp: cmp}
}

type orderable interface {
	~int | ~int64 | ~string | ~float32 | ~float64 | ~byte
}

// Max returns true if b is greater than a. May be used as a predicate to a
// comparing map to hold maximum values of the keys.
func Max[V orderable](a, b V) bool {
	return a < b
}

// NewMaxMap is a shortcut to create a comparing map[string]int64.
func NewMaxMap[K comparable, V orderable]() *cmpmap[K, V, func(a, b V) bool] {
	return NewCmpMap[K](Max[V])
}

// Reset cleans the map and returns the previously stored one.
func (m *cmpmap[K, V, _]) Reset() map[K]V {
	m.mux.Lock()
	defer m.mux.Unlock()
	prev := m.data
	m.data = nil
	return prev
}

// Get returns the stored value by key.
func (m *cmpmap[K, V, _]) Get(k K) (V, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	v, ok := m.data[k]
	return v, ok
}

// Add inserts the value v to the map at the key k, or updates the value if the
// comparison predicate returns true.
func (m *cmpmap[K, V, C]) Add(k K, v V) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if existing, ok := m.data[k]; ok {
		if m.cmp(existing, v) {
			m.data[k] = v
		}
	} else {
		if m.data == nil {
			m.data = make(map[K]V, 1)
		}
		m.data[k] = v
	}
}
