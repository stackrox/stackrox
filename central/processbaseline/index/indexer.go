package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	search "github.com/stackrox/rox/pkg/search"
	blevesearch "github.com/stackrox/rox/pkg/search/blevesearch"
)

// Indexer is the indexer for ProcessBaselines.
//
//go:generate mockgen-wrapper
type Indexer interface {
	AddProcessBaseline(processbaseline *storage.ProcessBaseline) error
	AddProcessBaselines(processbaselines []*storage.ProcessBaseline) error
	Count(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
	DeleteProcessBaseline(id string) error
	DeleteProcessBaselines(ids []string) error
	MarkInitialIndexingComplete() error
	NeedsInitialIndexing() (bool, error)
	Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
}
