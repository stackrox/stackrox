package index

import (
	"context"

	bleve "github.com/blevesearch/bleve"
	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	search "github.com/stackrox/rox/pkg/search"
	blevesearch "github.com/stackrox/rox/pkg/search/blevesearch"
)

// Indexer is the process indicator indexer.
//go:generate mockgen-wrapper
type Indexer interface {
	AddProcessIndicator(processindicator *storage.ProcessIndicator) error
	AddProcessIndicators(processindicators []*storage.ProcessIndicator) error
	DeleteProcessIndicator(id string) error
	DeleteProcessIndicators(ids []string) error
	MarkInitialIndexingComplete() error
	NeedsInitialIndexing() (bool, error)
	Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
	Count(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
}

// New returns a new process indicator indexer.
func New(index bleve.Index) Indexer {
	return &indexerImpl{index: index}
}
