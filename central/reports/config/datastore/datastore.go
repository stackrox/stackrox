package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/reports/config/search"
	"github.com/stackrox/rox/central/reports/config/store"
	pgStore "github.com/stackrox/rox/central/reports/config/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is the datastore for report configurations.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)

	GetReportConfigurations(ctx context.Context, query *v1.Query) ([]*storage.ReportConfiguration, error)
	GetReportConfiguration(ctx context.Context, id string) (*storage.ReportConfiguration, bool, error)
	AddReportConfiguration(ctx context.Context, reportConfig *storage.ReportConfiguration) (string, error)
	UpdateReportConfiguration(ctx context.Context, reportConfig *storage.ReportConfiguration) error
	RemoveReportConfiguration(ctx context.Context, id string) error

	Walk(ctx context.Context, fn func(reportConfig *storage.ReportConfiguration) error) error
}

// New returns a new DataStore instance.
func New(reportConfigStore store.Store, searcher search.Searcher) DataStore {
	return &dataStoreImpl{
		reportConfigStore: reportConfigStore,
		searcher:          searcher,
	}
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB) DataStore {
	store := pgStore.New(pool)
	indexer := pgStore.NewIndexer(pool)
	searcher := search.New(store, indexer)

	return New(store, searcher)
}
