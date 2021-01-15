package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

// InMemoryStore implements a simple in-memory-store for mocking.
type InMemoryStore struct {
	// map: ID -> bundle meta
	bundles map[string]*storage.InitBundleMeta
	mutex   sync.Mutex
}

// GetAll returns metadata for all active init bundles.
func (s *InMemoryStore) GetAll(ctx context.Context) ([]*storage.InitBundleMeta, error) {
	bundleMetas := make([]*storage.InitBundleMeta, 0, len(s.bundles))
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, bundleMeta := range s.bundles {
		bundleMetas = append(bundleMetas, bundleMeta.Clone())
	}
	return bundleMetas, nil
}

// Get lookups an init bundle by its ID.
func (s *InMemoryStore) Get(ctx context.Context, id string) (*storage.InitBundleMeta, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	bundle, ok := s.bundles[id]
	if !ok {
		return nil, ErrInitBundleNotFound
	}
	return bundle.Clone(), nil
}

// Add adds metadata for a new init bundles.
func (s *InMemoryStore) Add(ctx context.Context, bundleMeta *storage.InitBundleMeta) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, existsAlready := s.bundles[bundleMeta.GetId()]
	if existsAlready {
		return ErrInitBundleIDCollision
	}
	s.bundles[bundleMeta.GetId()] = bundleMeta.Clone()
	return nil
}

// NewInMemory returns a new in-memory store for init bundles for testing & mocking.
func NewInMemory() *InMemoryStore {
	return &InMemoryStore{
		bundles: make(map[string]*storage.InitBundleMeta),
	}
}
