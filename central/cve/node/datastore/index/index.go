package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// Indexer provides functionality to index node cves.
//
//go:generate mockgen-wrapper
type Indexer interface {
	AddNodeCVE(cve *storage.NodeCVE) error
	AddNodeCVEs(cves []*storage.NodeCVE) error
	Count(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
	DeleteNodeCVE(id string) error
	DeleteNodeCVEs(ids []string) error
	MarkInitialIndexingComplete() error
	NeedsInitialIndexing() (bool, error)
	Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
}
