package datastore

import (
	"context"

	"github.com/stackrox/rox/central/imagecveedge/search"
	"github.com/stackrox/rox/central/imagecveedge/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to Image/CVE edge storage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.ImageCVEEdge, error)
	Get(ctx context.Context, id string) (*storage.ImageCVEEdge, bool, error)
	// UpdateVulnerabilityStates updates vuln states for image-cve edges given a map of image ID to cve IDs to states.
	UpdateVulnerabilityStates(ctx context.Context, edgeStates map[string]map[string]storage.VulnerabilityState) error
}

// New returns a new instance of a DataStore.
func New(graphProvider graph.Provider, storage store.Store, searcher search.Searcher) DataStore {
	return &datastoreImpl{
		graphProvider: graphProvider,
		storage:       storage,
		searcher:      searcher,
	}
}
