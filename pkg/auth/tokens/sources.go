package tokens

import (
	"fmt"

	"github.com/stackrox/rox/pkg/sync"
)

type sourceStore struct {
	sources      map[string]Source
	sourcesMutex sync.RWMutex
}

func newSourceStore() *sourceStore {
	return &sourceStore{
		sources: make(map[string]Source),
	}
}

func (s *sourceStore) Get(id string) Source {
	s.sourcesMutex.RLock()
	defer s.sourcesMutex.RUnlock()
	return s.sources[id]
}

func (s *sourceStore) GetAll(ids ...string) ([]Source, error) {
	result := make([]Source, len(ids))
	s.sourcesMutex.Lock()
	defer s.sourcesMutex.Unlock()
	for i, id := range ids {
		src := s.sources[id]
		if src == nil {
			return nil, fmt.Errorf("invalid source %s", id)
		}
		result[i] = src
	}
	return result, nil
}

func (s *sourceStore) Register(source Source) error {
	s.sourcesMutex.Lock()
	defer s.sourcesMutex.Unlock()

	if _, exists := s.sources[source.ID()]; exists {
		return fmt.Errorf("source with id %s already exists", source.ID())
	}
	s.sources[source.ID()] = source
	return nil
}

func (s *sourceStore) Unregister(source Source) error {
	s.sourcesMutex.Lock()
	defer s.sourcesMutex.Unlock()
	if _, exists := s.sources[source.ID()]; !exists {
		return fmt.Errorf("source with id %s does not exist", source.ID())
	}
	log.Debug("removing token source ", source.ID())
	delete(s.sources, source.ID())
	return nil
}
