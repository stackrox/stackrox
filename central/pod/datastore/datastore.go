package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/pod/datastore/internal/search"
	"github.com/stackrox/rox/central/pod/store/cache"
	pgStore "github.com/stackrox/rox/central/pod/store/postgres"
	piDS "github.com/stackrox/rox/central/processindicator/datastore"
	piFilter "github.com/stackrox/rox/central/processindicator/filter"
	plopDS "github.com/stackrox/rox/central/processlisteningonport/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/process/filter"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to PodStorage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	SearchRawPods(ctx context.Context, q *v1.Query) ([]*storage.Pod, error)

	GetPod(ctx context.Context, id string) (*storage.Pod, bool, error)
	WalkAll(ctx context.Context, fn func(pod *storage.Pod) error) error

	UpsertPod(ctx context.Context, pod *storage.Pod) error

	RemovePod(ctx context.Context, id string) error

	GetPodIDs(ctx context.Context) ([]string, error)
}

// NewPostgresDB creates a pod datastore based on Postgres
func NewPostgresDB(db postgres.DB, indicators piDS.DataStore, plops plopDS.DataStore, processFilter filter.Filter) (DataStore, error) {
	store, err := cache.NewCachedStore(pgStore.New(db))
	if err != nil {
		return nil, err
	}
	searcher := search.New(store, pgStore.NewIndexer(db))
	return newDatastoreImpl(store, searcher, indicators, plops, processFilter), nil
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) (DataStore, error) {
	processIndicatorStore, err := piDS.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	plopStore := plopDS.GetTestPostgresDataStore(t, pool)
	processIndicatorFilter := piFilter.Singleton()
	return NewPostgresDB(pool, processIndicatorStore, plopStore, processIndicatorFilter)
}
