package datastore

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/processwhitelist/index"
	"github.com/stackrox/rox/central/processwhitelist/search"
	"github.com/stackrox/rox/central/processwhitelist/store"
	"github.com/stackrox/rox/central/processwhitelistresults/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// DataStore wraps storage, indexer, and searcher for ProcessWhitelists.
//go:generate mockgen-wrapper
type DataStore interface {
	SearchRawProcessWhitelists(ctx context.Context, q *v1.Query) ([]*storage.ProcessWhitelist, error)
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)

	GetProcessWhitelist(ctx context.Context, key *storage.ProcessWhitelistKey) (*storage.ProcessWhitelist, bool, error)
	AddProcessWhitelist(ctx context.Context, whitelist *storage.ProcessWhitelist) (string, error)
	RemoveProcessWhitelist(ctx context.Context, key *storage.ProcessWhitelistKey) error
	RemoveProcessWhitelistsByDeployment(ctx context.Context, deploymentID string) error
	UpdateProcessWhitelistElements(ctx context.Context, key *storage.ProcessWhitelistKey, addElements []*storage.WhitelistItem, removeElements []*storage.WhitelistItem, auto bool) (*storage.ProcessWhitelist, error)
	UpsertProcessWhitelist(ctx context.Context, key *storage.ProcessWhitelistKey, addElements []*storage.WhitelistItem, auto bool) (*storage.ProcessWhitelist, error)
	UserLockProcessWhitelist(ctx context.Context, key *storage.ProcessWhitelistKey, locked bool) (*storage.ProcessWhitelist, error)

	WalkAll(ctx context.Context, fn func(whitelist *storage.ProcessWhitelist) error) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher, processWhitelistResults datastore.DataStore) DataStore {
	d := &datastoreImpl{
		storage:                 storage,
		indexer:                 indexer,
		searcher:                searcher,
		whitelistLock:           concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
		processWhitelistResults: processWhitelistResults,
	}
	return d
}
