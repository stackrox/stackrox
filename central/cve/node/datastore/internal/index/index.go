package index

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/blevesearch"
)

// Indexer provides functionality to index node cves.
//go:generate mockgen-wrapper
type Indexer interface {
	AddNodeCVE(cve *storage.NodeCVE) error
	AddNodeCVEs(cves []*storage.NodeCVE) error
	Count(q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
	DeleteNodeCVE(id string) error
	DeleteNodeCVEs(ids []string) error
	MarkInitialIndexingComplete() error
	NeedsInitialIndexing() (bool, error)
	Search(q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
}
