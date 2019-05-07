package datastore

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/processwhitelist/index"
	"github.com/stackrox/rox/central/processwhitelist/search"
	"github.com/stackrox/rox/central/processwhitelist/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
)

// DataStore wraps storage, indexer, and searcher for ProcessWhitelists.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	SearchRawProcessWhitelists(ctx context.Context, q *v1.Query) ([]*storage.ProcessWhitelist, error)

	GetProcessWhitelist(ctx context.Context, key *storage.ProcessWhitelistKey) (*storage.ProcessWhitelist, error)
	GetProcessWhitelists(ctx context.Context) ([]*storage.ProcessWhitelist, error)
	AddProcessWhitelist(ctx context.Context, whitelist *storage.ProcessWhitelist) (string, error)
	RemoveProcessWhitelist(ctx context.Context, key *storage.ProcessWhitelistKey) error
	UpdateProcessWhitelistElements(ctx context.Context, key *storage.ProcessWhitelistKey, addElements []*storage.WhitelistItem, removeElements []*storage.WhitelistItem, auto bool) (*storage.ProcessWhitelist, error)
	UpsertProcessWhitelist(ctx context.Context, key *storage.ProcessWhitelistKey, addElements []*storage.WhitelistItem, auto bool) (*storage.ProcessWhitelist, error)
	UserLockProcessWhitelist(ctx context.Context, key *storage.ProcessWhitelistKey, locked bool) (*storage.ProcessWhitelist, error)
	RoxLockProcessWhitelist(ctx context.Context, key *storage.ProcessWhitelistKey, locked bool) (*storage.ProcessWhitelist, error)
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	d := &datastoreImpl{
		storage:       storage,
		indexer:       indexer,
		searcher:      searcher,
		whitelistLock: concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
	}
	return d
}
