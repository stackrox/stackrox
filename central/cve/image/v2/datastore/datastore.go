package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/cve/image/v2/datastore/store"
	pgStore "github.com/stackrox/rox/central/cve/image/v2/datastore/store/postgres"
	"github.com/stackrox/rox/pkg/postgres"
)

// DataStore is an intermediary to CVE storage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// UpsertCVE inserts a CVE row if it doesn't exist (two-phase: insert then fetch).
	// Returns the UUID of the CVE row (whether newly inserted or pre-existing).
	UpsertCVE(ctx context.Context, cveRow *store.CVERow) (string, error)

	// UpsertEdge inserts or updates a component_cve_edges row.
	// first_system_occurrence is preserved on conflict (not updated).
	// is_fixable and fixed_by are refreshed on conflict.
	UpsertEdge(ctx context.Context, edge *store.EdgeRow) error

	// DeleteStaleEdges removes edges for a component whose cve_id is NOT in keepCVEIDs.
	// If keepCVEIDs is empty, all edges for the component are deleted.
	DeleteStaleEdges(ctx context.Context, componentID string, keepCVEIDs []string) error

	// GetCVEsForImage returns all CVEs for a given image (joined through component_cve_edges and image_component_v2).
	GetCVEsForImage(ctx context.Context, imageID string) ([]*store.CVERow, error)

	// GetAllReferencedCVEs returns all CVEs referenced by at least one component_cve_edges row.
	GetAllReferencedCVEs(ctx context.Context) ([]*store.CVERow, error)

	// DeleteOrphanedCVEsBatch deletes up to batchSize CVEs with no referencing edges.
	// Returns number of rows deleted.
	DeleteOrphanedCVEsBatch(ctx context.Context, batchSize int) (int64, error)
}

// New returns a new instance of a DataStore.
func New(storage store.Store) DataStore {
	ds := &datastoreImpl{
		storage: storage,
	}
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	dbstore := pgStore.New(pool)
	return New(dbstore)
}
