package maputil

///
// CmpMap is a version of syncmap, which compares the values before inserting.
//

type cmpmap[K comparable, V any] struct {
	syncmap[K, V]
	cmp func(a, b V) bool
}

// NewCmpMap creates a new comparing map, which only updates the stored values if the
// cmp predicate returns true when provided with the existing and new values.
// If cmp is nil, the map works as a normal map, with synchronized access.
//
// Example:
//
//	m := NewCmpMap[string](Max[int])
//	m.Add("a", 10)
//	m.Add("a", 5)
//	v, ok := m.Get("a") // 10
func NewCmpMap[K comparable, V any](cmp func(a, b V) bool) SyncMap[K, V] {
	return &cmpmap[K, V]{
		syncmap: syncmap[K, V]{data: make(map[K]V)},
		cmp:     cmp}
}

// Store inserts the value v to the map at the key k, or updates the value if the
// comparison predicate returns true.
func (m *cmpmap[K, V]) Store(k K, v V) {
	if m.cmp == nil {
		m.syncmap.Store(k, v)
		return
	}
	m.mux.Lock()
	defer m.mux.Unlock()
	if existing, ok := m.data[k]; ok {
		if !m.cmp(existing, v) {
			return
		}
	} else if m.data == nil {
		m.data = make(map[K]V, 1)
	}
	m.data[k] = v
}

//
// MaxMap is a version of CmpMap that stores maximum values.
//

type orderable interface {
	~int | ~int32 | ~int64 | ~string | ~float32 | ~float64 | ~byte
}

// Max returns true if b is greater than a. May be used as a predicate to a
// comparing map to hold maximum values.
func Max[V orderable](a, b V) bool {
	return a < b
}

// NewMaxMap is a shortcut to create a comparing map[string]int64.
func NewMaxMap[K comparable, V orderable]() SyncMap[K, V] {
	return NewCmpMap[K](Max[V])
}
