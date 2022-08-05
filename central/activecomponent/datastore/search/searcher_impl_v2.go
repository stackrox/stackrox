package search

import (
	"context"

	"github.com/stackrox/rox/central/activecomponent/datastore/index"
	"github.com/stackrox/rox/central/activecomponent/datastore/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/scoped/postgres"
)

// NewV2 returns a new instance of Searcher for the given storage and indexer.
func NewV2(storage store.Store, indexer index.Indexer) Searcher {
	return &searcherImplV2{
		storage:  storage,
		indexer:  indexer,
		searcher: postgres.WithScoping(blevesearch.WrapUnsafeSearcherAsSearcher(indexer)),
	}
}

// searcherImplV2 provides an intermediary implementation layer for image storage.
type searcherImplV2 struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

// SearchRawActiveComponents retrieves SearchResults from the indexer and storage
func (s *searcherImplV2) SearchRawActiveComponents(ctx context.Context, q *v1.Query) ([]*storage.ActiveComponent, error) {
	return s.storage.GetByQuery(ctx, q)
}

func (s *searcherImplV2) Search(ctx context.Context, q *v1.Query) (res []search.Result, err error) {
	return s.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (s *searcherImplV2) Count(ctx context.Context, q *v1.Query) (count int, err error) {
	return s.searcher.Count(ctx, q)
}
