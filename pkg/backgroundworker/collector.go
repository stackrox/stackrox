package backgroundworker

// Collector is the accumulation strategy for BatchAccumulator. It defines
// how items are stored and retrieved. Implementations must be safe for
// concurrent use when accessed under BatchAccumulator's internal lock.
type Collector[T any] interface {
	// Add appends an item to the collection.
	Add(T)
	// DrainAll returns all accumulated items and resets the collection.
	DrainAll() []T
	// Len returns the number of accumulated items.
	Len() int
}

// SliceCollector appends every item to a slice. This is the default
// collector used by BatchAccumulator when no Collector is specified.
type SliceCollector[T any] struct {
	items []T
}

func (c *SliceCollector[T]) Add(item T) {
	c.items = append(c.items, item)
}

func (c *SliceCollector[T]) DrainAll() []T {
	items := c.items
	c.items = nil
	return items
}

func (c *SliceCollector[T]) Len() int {
	return len(c.items)
}

// SetCollector deduplicates items by value. T must be comparable.
// Later additions of an already-present value are no-ops.
type SetCollector[T comparable] struct {
	set map[T]struct{}
}

func (c *SetCollector[T]) Add(item T) {
	if c.set == nil {
		c.set = make(map[T]struct{})
	}
	c.set[item] = struct{}{}
}

func (c *SetCollector[T]) DrainAll() []T {
	items := make([]T, 0, len(c.set))
	for k := range c.set {
		items = append(items, k)
	}
	c.set = nil
	return items
}

func (c *SetCollector[T]) Len() int {
	return len(c.set)
}

// MapCollector deduplicates by key, keeping the latest value for each
// key. KeyFunc extracts the dedup key from each item.
type MapCollector[K comparable, V any] struct {
	KeyFunc func(V) K
	m       map[K]V
}

func (c *MapCollector[K, V]) Add(item V) {
	if c.m == nil {
		c.m = make(map[K]V)
	}
	c.m[c.KeyFunc(item)] = item
}

func (c *MapCollector[K, V]) DrainAll() []V {
	items := make([]V, 0, len(c.m))
	for _, v := range c.m {
		items = append(items, v)
	}
	c.m = nil
	return items
}

func (c *MapCollector[K, V]) Len() int {
	return len(c.m)
}
