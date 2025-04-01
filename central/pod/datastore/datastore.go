package datastore

import (
	"context"

	"github.com/stackrox/rox/central/pod/datastore/internal/search"
	pgStore "github.com/stackrox/rox/central/pod/datastore/internal/store/postgres"
	piDS "github.com/stackrox/rox/central/processindicator/datastore"
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
	WalkByQuery(ctx context.Context, q *v1.Query, fn func(p *storage.Pod) error) error

	UpsertPod(ctx context.Context, pod *storage.Pod) error

	RemovePod(ctx context.Context, id string) error
}

// NewPostgresDB creates a pod datastore based on Postgres
func NewPostgresDB(db postgres.DB, indicators piDS.DataStore, plops plopDS.DataStore, processFilter filter.Filter) DataStore {
	store := pgStore.New(db)
	searcher := search.New(store)
	return newDatastoreImpl(store, searcher, indicators, plops, processFilter)
}
