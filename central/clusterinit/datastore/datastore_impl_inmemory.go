package datastore

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

// Store implements a simple in-memory-store for mocking.
type Store struct {
	// map: boot strap token ID -> meta
	tokens map[string]*storage.BootstrapTokenWithMeta
	mutex  sync.Mutex
}

// GetAll returns metadata for all active bootstrap tokens.
func (s *Store) GetAll(ctx context.Context) ([]*storage.BootstrapTokenWithMeta, error) {
	tokenMetas := make([]*storage.BootstrapTokenWithMeta, 0, len(s.tokens))
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, tokenMeta := range s.tokens {
		tokenMetas = append(tokenMetas, tokenMeta.Clone())
	}
	return tokenMetas, nil
}

// Get lookups a bootstrap token by ID.
func (s *Store) Get(ctx context.Context, tokenID string) (*storage.BootstrapTokenWithMeta, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	token, ok := s.tokens[tokenID]
	if !ok {
		return nil, ErrTokenNotFound
	}
	return token.Clone(), nil
}

// Add adds metadata for a new bootstrap token.
func (s *Store) Add(ctx context.Context, tokenMeta *storage.BootstrapTokenWithMeta) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, existsAlready := s.tokens[tokenMeta.GetId()]
	if existsAlready {
		return ErrTokenIDCollision
	}
	s.tokens[tokenMeta.GetId()] = tokenMeta.Clone()
	return nil
}

// Delete deletes metadata for a bootstrap token identified by its ID.
func (s *Store) Delete(ctx context.Context, tokenID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, found := s.tokens[tokenID]
	if !found {
		return ErrTokenNotFound
	}
	delete(s.tokens, tokenID)
	return nil
}

// SetActive sets the `active` property of the referenced bootstrap token.
func (s *Store) SetActive(ctx context.Context, tokenID string, active bool) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tokenMeta, found := s.tokens[tokenID]
	if !found {
		return ErrTokenNotFound
	}

	tokenMeta.Active = active

	return nil
}

// NewInMemory returns a new in-memory BootstrapTokenStore for testing & mocking.
func NewInMemory() *Store {
	return &Store{
		tokens: make(map[string]*storage.BootstrapTokenWithMeta),
	}
}
