package search

import (
	"context"

	"github.com/stackrox/rox/central/activecomponent/datastore/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// NewV2 returns a new instance of Searcher for the given storage.
func NewV2(storage store.Store) Searcher {
	return &searcherImplV2{
		storage: storage,
	}
}

// searcherImplV2 provides an intermediary implementation layer for image storage.
type searcherImplV2 struct {
	storage store.Store
}

// SearchRawActiveComponents retrieves SearchResults from the storage
func (s *searcherImplV2) SearchRawActiveComponents(ctx context.Context, q *v1.Query) ([]*storage.ActiveComponent, error) {
	results, err := s.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	images, _, err := s.storage.GetMany(ctx, search.ResultsToIDs(results))
	if err != nil {
		return nil, err
	}
	return images, nil
}

func (s *searcherImplV2) Search(ctx context.Context, q *v1.Query) (res []search.Result, err error) {
	return s.storage.Search(ctx, q)
}
