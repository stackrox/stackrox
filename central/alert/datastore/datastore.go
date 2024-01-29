package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/alert/datastore/internal/search"
	"github.com/stackrox/rox/central/alert/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/alert/datastore/internal/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/postgres"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

var (
	// Override the default mutex pool size to allow for better parallelism
	mutexPoolSize uint32 = 1024
)

// DataStore is a transaction script with methods that provide the domain logic for CRUD uses cases for Alert objects.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchAlerts(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawAlerts(ctx context.Context, q *v1.Query) ([]*storage.Alert, error)
	SearchListAlerts(ctx context.Context, q *v1.Query) ([]*storage.ListAlert, error)

	WalkByQuery(ctx context.Context, q *v1.Query, db func(d *storage.Alert) error) error
	WalkAll(ctx context.Context, fn func(alert *storage.ListAlert) error) error
	GetAlert(ctx context.Context, id string) (*storage.Alert, bool, error)
	CountAlerts(ctx context.Context) (int, error)
	UpsertAlert(ctx context.Context, alert *storage.Alert) error
	UpsertAlerts(ctx context.Context, alerts []*storage.Alert) error
	// MarkAlertsResolvedBatch marks alerts with specified ids as RESOLVED in batch and returns resolved alerts.
	MarkAlertsResolvedBatch(ctx context.Context, id ...string) ([]*storage.Alert, error)

	DeleteAlerts(ctx context.Context, ids ...string) error
}

// New returns a new soleInstance of DataStore using the input store, and searcher.
func New(alertStore store.Store, searcher search.Searcher) (DataStore, error) {
	ds := &datastoreImpl{
		storage:    alertStore,
		searcher:   searcher,
		keyedMutex: concurrency.NewKeyedMutex(mutexPoolSize),
		keyFence:   concurrency.NewKeyFence(),
	}
	return ds, nil
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) (DataStore, error) {
	alertStore := pgStore.New(pool)
	indexer := pgStore.NewIndexer(pool)
	searcher := search.New(alertStore, indexer)

	return New(alertStore, searcher)
}
