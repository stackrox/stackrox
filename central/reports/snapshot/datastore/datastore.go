package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/reports/snapshot/datastore/search"
	pgStore "github.com/stackrox/rox/central/reports/snapshot/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// DataStore is the entry point for searching, inserting or modifying ReportSnapshots.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchReportSnapshots(ctx context.Context, q *v1.Query) ([]*storage.ReportSnapshot, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.ReportSnapshot, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ReportSnapshot, error)

	AddReportSnapshot(ctx context.Context, report *storage.ReportSnapshot) error
	DeleteReportSnapshot(ctx context.Context, id string) error

	Walk(ctx context.Context, fn func(report *storage.ReportSnapshot) error) error
}

// New returns a new instance of a DataStore
func New(storage pgStore.Store, searcher search.Searcher) (DataStore, error) {
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		return nil, nil
	}
	ds := &datastoreImpl{
		storage:  storage,
		searcher: searcher,
	}
	return ds, nil
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB) (DataStore, error) {
	store := pgStore.New(pool)
	indexer := pgStore.NewIndexer(pool)
	searcher := search.New(store, indexer)

	return New(store, searcher)
}
