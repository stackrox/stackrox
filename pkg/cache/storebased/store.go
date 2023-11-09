package storebased

import (
	"context"

	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/sync"
)

// GetObjectFunction is the datastore function which retrieves a specific object.
type GetObjectFunction[T any] func(ctx context.Context, id string) (T, error)

// NewCache returns a cache which is expected to be backed by a store (i.e. the GetObjectFunction is a call to the
// database).
// As underlying in-memory structure a maputil.FastRMap is used, hence it is only recommended to use this cache
// in case of heavy read but sparse write operations.
func NewCache[T any](refreshFn GetObjectFunction[T]) *Cache[T] {
	return &Cache[T]{
		cachedObjects: maputil.NewFastRMap[string, T](),
		getObjectFn:   refreshFn,
	}
}

// Cache represents a cache which is backed by a store.
type Cache[T any] struct {
	mutex         sync.Mutex
	cachedObjects *maputil.FastRMap[string, T]
	getObjectFn   GetObjectFunction[T]
}

// GetObject retrieves the given object either from cache or from the specified datastore function.
func (s *Cache[T]) GetObject(ctx context.Context, id string) (T, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	obj, exists := s.cachedObjects.Get(id)
	if exists {
		return obj, nil
	}

	obj, err := s.getObjectFn(ctx, id)
	if err != nil {
		return *new(T), err
	}

	s.cachedObjects.Set(id, obj)
	return obj, nil
}

// InvalidateCache invalidates specific objects in the cache.
func (s *Cache[T]) InvalidateCache(ids ...string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.cachedObjects.DeleteMany(ids...)
}
