package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/processindicator"
	"github.com/stackrox/rox/central/processindicator/pruner"
	"github.com/stackrox/rox/central/processindicator/store"
	pgStore "github.com/stackrox/rox/central/processindicator/store/postgres"
	"github.com/stackrox/rox/central/processindicator/views"
	plopStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// DataStore represents the interface to access data.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Count(ctx context.Context, q *v1.Query) (int, error)

	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	SearchRawProcessIndicators(ctx context.Context, q *v1.Query) ([]*storage.ProcessIndicator, error)
	GetByQueryFn(ctx context.Context, query *v1.Query, fn func(obj *storage.ProcessIndicator) error) error

	GetProcessIndicator(ctx context.Context, id string) (*storage.ProcessIndicator, bool, error)
	GetProcessIndicators(ctx context.Context, ids []string) ([]*storage.ProcessIndicator, bool, error)
	AddProcessIndicators(context.Context, ...*storage.ProcessIndicator) error
	RemoveProcessIndicatorsByPod(ctx context.Context, id string) error
	RemoveProcessIndicators(ctx context.Context, ids []string) error
	PruneProcessIndicators(ctx context.Context, ids []string) (int, error)

	WalkByQuery(ctx context.Context, query *v1.Query, fn func(obj *storage.ProcessIndicator) error) error

	// IterateOverProcessIndicatorsRiskView iterates over minimal fields from process indicator for risk evaluation
	IterateOverProcessIndicatorsRiskView(ctx context.Context, q *v1.Query, fn func(*views.ProcessIndicatorRiskView) error) error

	// Stop signals all goroutines associated with this object to terminate.
	Stop()
	// Wait waits until all goroutines associated with this object have terminated, or cancelWhen gets triggered.
	// A return value of false indicates that cancelWhen was triggered.
	Wait(cancelWhen concurrency.Waitable) bool
}

// New returns a new instance of DataStore using the input store, and searcher.
func New(db postgres.DB, store store.Store, plopStorage plopStore.Store, prunerFactory pruner.Factory) DataStore {
	d := &datastoreImpl{
		db:                    db,
		storage:               store,
		plopStorage:           plopStorage,
		prunerFactory:         prunerFactory,
		prunedArgsLengthCache: make(map[processindicator.ProcessWithContainerInfo]int),
		stopper:               concurrency.NewStopper(),
	}
	ctx := sac.WithAllAccess(context.Background())

	if env.ProcessPruningEnabled.BooleanSetting() {
		go d.prunePeriodically(ctx)
	}
	return d
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	dbstore := pgStore.New(pool)
	plopDBstore := plopStore.New(pool)
	return New(pool, dbstore, plopDBstore, nil)
}
