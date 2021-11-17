package datastore

import (
	"context"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/imagecveedge/index"
	"github.com/stackrox/rox/central/imagecveedge/search"
	"github.com/stackrox/rox/central/imagecveedge/store/dackbox"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	pkgDackBox "github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/features"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to Image/CVE edge storage.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.ImageCVEEdge, error)
	Get(ctx context.Context, id string) (*storage.ImageCVEEdge, bool, error)
	UpdateVulnerabilityState(ctx context.Context, cve string, images []string, state storage.VulnerabilityState) error
}

func newDataStore(dacky *pkgDackBox.DackBox, keyFence concurrency.KeyFence, globalIndex bleve.Index) DataStore {
	storage := dackbox.New(dacky, keyFence)

	var searcher search.Searcher
	if features.VulnRiskManagement.Enabled() {
		searcher = search.New(storage, index.New(globalIndex))
	}

	return &datastoreImpl{
		graphProvider: dacky,
		storage:       storage,
		searcher:      searcher,
	}
}

// New returns a new instance of a DataStore.
func New(dacky *pkgDackBox.DackBox, keyFence concurrency.KeyFence, globalIndex bleve.Index) DataStore {
	return newDataStore(dacky, keyFence, globalIndex)
}
