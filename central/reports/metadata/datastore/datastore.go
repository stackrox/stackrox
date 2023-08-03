package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/reports/metadata/datastore/search"
	pgStore "github.com/stackrox/rox/central/reports/metadata/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// DataStore is the entry point for searching, inserting or modifying ReportMetadatas.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchReportMetadatas(ctx context.Context, q *v1.Query) ([]*storage.ReportMetadata, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.ReportMetadata, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ReportMetadata, error)
	// AddReportMetadata adds the given ReportMetadata object; Populates the `ReportId` field on the object and returns
	// the generated `ReportId`
	AddReportMetadata(ctx context.Context, report *storage.ReportMetadata) (string, error)
	DeleteReportMetadata(ctx context.Context, id string) error
	UpdateReportMetadata(ctx context.Context, report *storage.ReportMetadata) error

	Walk(ctx context.Context, fn func(report *storage.ReportMetadata) error) error
}

// New returns a new instance of a DataStore
func New(storage pgStore.Store, searcher search.Searcher) (DataStore, error) {
	if !env.VulnReportingEnhancements.BooleanSetting() {
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
