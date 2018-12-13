package datastore

import (
	"github.com/stackrox/rox/central/processindicator/index"
	"github.com/stackrox/rox/central/processindicator/pruner"
	"github.com/stackrox/rox/central/processindicator/search"
	"github.com/stackrox/rox/central/processindicator/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to PolicyStorage.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	Search(q *v1.Query) ([]pkgSearch.Result, error)
	SearchRawProcessIndicators(q *v1.Query) ([]*storage.ProcessIndicator, error)

	GetProcessIndicator(id string) (*storage.ProcessIndicator, bool, error)
	GetProcessIndicators() ([]*storage.ProcessIndicator, error)
	AddProcessIndicator(*storage.ProcessIndicator) error
	AddProcessIndicators(...*storage.ProcessIndicator) error
	RemoveProcessIndicator(id string) error
	RemoveProcessIndicatorsByDeployment(id string) error
	RemoveProcessIndicatorsOfStaleContainers(deploymentID string, currentContainerIDs []string) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher, pruner pruner.Pruner) DataStore {
	d := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
		pruner:   pruner,
	}
	go d.prunePeriodically()
	return d
}
