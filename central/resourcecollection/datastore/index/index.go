package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// Indexer provides functionality to index node components.
type Indexer interface {
	AddResourceCollection(collection *storage.ResourceCollection) error
	AddResourceCollections(collections []*storage.ResourceCollection) error
	Count(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
	DeleteResourceCollection(id string) error
	DeleteResourceCollections(ids []string) error
	Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
}
