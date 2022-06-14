package search

import (
	"context"

	"github.com/stackrox/stackrox/central/cve/image/datastore/internal/store/postgres"
	"github.com/stackrox/stackrox/central/cve/index"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/blevesearch"
	pkgPostgres "github.com/stackrox/stackrox/pkg/search/scoped/postgres"
)

// Searcher provides search functionality on existing cves.
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, query *v1.Query) (int, error)
	SearchCVEs(context.Context, *v1.Query) ([]*v1.SearchResult, error)
	SearchRawCVEs(ctx context.Context, query *v1.Query) ([]*storage.CVE, error)
}

// New returns a new instance of Searcher for the given storage and indexer.
func New(storage postgres.Store, indexer index.Indexer) Searcher {
	return &searcherImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: pkgPostgres.WithScoping(blevesearch.WrapUnsafeSearcherAsSearcher(indexer)),
	}
}
