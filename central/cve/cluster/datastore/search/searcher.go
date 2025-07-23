package search

import (
	"context"

	"github.com/stackrox/rox/central/cve/cluster/datastore/store"
	"github.com/stackrox/rox/central/cve/edgefields"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Searcher provides search functionality on existing cves.
//
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	SearchClusterCVEs(context.Context, *v1.Query) ([]*v1.SearchResult, error)
	SearchRawClusterCVEs(ctx context.Context, query *v1.Query) ([]*storage.ClusterCVE, error)
}

// New returns a new instance of Searcher for the given the storage.
func New(storage store.Store) Searcher {
	return &searcherImpl{
		storage:  storage,
		searcher: formatSearcherV2(storage),
	}
}

func formatSearcherV2(searcher search.Searcher) search.Searcher {
	return edgefields.TransformFixableFields(searcher)
}
