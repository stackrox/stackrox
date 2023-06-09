package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	search "github.com/stackrox/rox/pkg/search"
	blevesearch "github.com/stackrox/rox/pkg/search/blevesearch"
)

// Indexer is the cve indexer.
//
//go:generate mockgen-wrapper
type Indexer interface {
	AddCVE(cve *storage.CVE) error
	AddCVEs(cves []*storage.CVE) error
	Count(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
	DeleteCVE(id string) error
	DeleteCVEs(ids []string) error
	MarkInitialIndexingComplete() error
	NeedsInitialIndexing() (bool, error)
	Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
}
