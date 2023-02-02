package datastore

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/reportconfigurations/index"
	"github.com/stackrox/rox/central/reportconfigurations/search"
	"github.com/stackrox/rox/central/reportconfigurations/store"
	"github.com/stackrox/rox/central/reportconfigurations/store/postgres"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
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
func New(reportConfigStore store.Store, indexer index.Indexer, searcher search.Searcher) (DataStore, error) {
	d := &dataStoreImpl{
		reportConfigStore: reportConfigStore,
		searcher:          searcher,
		indexer:           indexer,
	}

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			// TODO: ROX-13888 Replace VulnerabilityReports with WorkflowAdministration.
			sac.ResourceScopeKeys(resources.VulnerabilityReports)))
	if err := d.buildIndex(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to build index from existing store")
	}
	return d, nil
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool *pgxpool.Pool) (DataStore, error) {
	store := postgres.New(pool)
	indexer := postgres.NewIndexer(pool)
	searcher := search.New(store, indexer)

	return New(store, indexer, searcher)
}
