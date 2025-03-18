package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/alert/datastore/internal/search"
	"github.com/stackrox/rox/central/alert/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/alert/datastore/internal/store/postgres"
	platformmatcher "github.com/stackrox/rox/central/platform/matcher"
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
	Search(ctx context.Context, q *v1.Query, excludeResolved bool) ([]searchPkg.Result, error)
	Count(ctx context.Context, q *v1.Query, excludeResolved bool) (int, error)
	SearchAlerts(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawAlerts(ctx context.Context, q *v1.Query, excludeResolved bool) ([]*storage.Alert, error)
	SearchListAlerts(ctx context.Context, q *v1.Query, excludeResolved bool) ([]*storage.ListAlert, error)

	GetByQuery(ctx context.Context, q *v1.Query) ([]*storage.Alert, error)
	WalkByQuery(ctx context.Context, q *v1.Query, db func(d *storage.Alert) error) error
	WalkAll(ctx context.Context, fn func(alert *storage.ListAlert) error) error
	GetAlert(ctx context.Context, id string) (*storage.Alert, bool, error)
	CountAlerts(ctx context.Context) (int, error)
	UpsertAlert(ctx context.Context, alert *storage.Alert) error
	UpsertAlerts(ctx context.Context, alerts []*storage.Alert) error
	// MarkAlertsResolvedBatch marks alerts with specified ids as RESOLVED in batch and returns resolved alerts.
	MarkAlertsResolvedBatch(ctx context.Context, id ...string) ([]*storage.Alert, error)

	DeleteAlerts(ctx context.Context, ids ...string) error
	PruneAlerts(ctx context.Context, ids ...string) error
}

// New returns a new soleInstance of DataStore using the input store, and searcher.
func New(alertStore store.Store, searcher search.Searcher, platformMatcher platformmatcher.PlatformMatcher) DataStore {
	ds := &datastoreImpl{
		storage:         alertStore,
		searcher:        searcher,
		keyedMutex:      concurrency.NewKeyedMutex(mutexPoolSize),
		keyFence:        concurrency.NewKeyFence(),
		platformMatcher: platformMatcher,
	}
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	alertStore := pgStore.New(pool)
	searcher := search.New(alertStore)

	return New(alertStore, searcher, platformmatcher.Singleton())
}
