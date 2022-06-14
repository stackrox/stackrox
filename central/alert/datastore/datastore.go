package datastore

import (
	"context"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/alert/datastore/internal/index"
	"github.com/stackrox/rox/central/alert/datastore/internal/search"
	"github.com/stackrox/rox/central/alert/datastore/internal/store"
	"github.com/stackrox/rox/central/alert/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	rocksdbBase "github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is a transaction script with methods that provide the domain logic for CRUD uses cases for Alert objects.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchAlerts(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawAlerts(ctx context.Context, q *v1.Query) ([]*storage.Alert, error)
	SearchListAlerts(ctx context.Context, q *v1.Query) ([]*storage.ListAlert, error)

	ListAlerts(ctx context.Context, request *v1.ListAlertsRequest) ([]*storage.ListAlert, error)
	WalkAll(ctx context.Context, fn func(alert *storage.ListAlert) error) error
	GetAlert(ctx context.Context, id string) (*storage.Alert, bool, error)
	CountAlerts(ctx context.Context) (int, error)
	UpsertAlert(ctx context.Context, alert *storage.Alert) error
	UpsertAlerts(ctx context.Context, alerts []*storage.Alert) error
	MarkAlertStale(ctx context.Context, id string) error

	DeleteAlerts(ctx context.Context, ids ...string) error

	AddAlertTags(ctx context.Context, alertID string, tags []string) ([]string, error)
	RemoveAlertTags(ctx context.Context, alertID string, tags []string) error
}

// New returns a new soleInstance of DataStore using the input store, indexer, and searcher.
func New(alertStore store.Store, indexer index.Indexer, searcher search.Searcher) (DataStore, error) {
	ds := &datastoreImpl{
		storage:    alertStore,
		indexer:    indexer,
		searcher:   searcher,
		keyedMutex: concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
	}
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Alert)))
	if err := ds.buildIndex(ctx); err != nil {
		return nil, err
	}
	return ds, nil
}

// NewWithDb returns a new soleInstance of DataStore using the input indexer, and searcher.
func NewWithDb(db *rocksdbBase.RocksDB, bIndex bleve.Index) DataStore {
	alertStore := rocksdb.New(db)
	indexer := index.New(bIndex)
	searcher := search.New(alertStore, indexer)

	return &datastoreImpl{
		storage:    alertStore,
		indexer:    indexer,
		searcher:   searcher,
		keyedMutex: concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
	}
}
