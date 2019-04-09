package datastore

import (
	"github.com/stackrox/rox/central/processwhitelist/index"
	"github.com/stackrox/rox/central/processwhitelist/search"
	"github.com/stackrox/rox/central/processwhitelist/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore wraps storage, indexer, and searcher for ProcessWhitelists.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	SearchRawProcessWhitelists(q *v1.Query) ([]*storage.ProcessWhitelist, error)

	GetProcessWhitelist(id string) (*storage.ProcessWhitelist, error)
	GetProcessWhitelistByNames(deploymentID, containerName string) (*storage.ProcessWhitelist, error)
	GetProcessWhitelists() ([]*storage.ProcessWhitelist, error)
	AddProcessWhitelist(whitelist *storage.ProcessWhitelist) (string, error)
	RemoveProcessWhitelist(id string) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	d := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}
	return d
}
