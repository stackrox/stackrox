package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/cve/image/v2/datastore/store"
	pgStore "github.com/stackrox/rox/central/cve/image/v2/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to CVE storage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// Search implements search.Searcher for the search framework.
	// NOTE: returns empty results — image_cves_v2 table has been replaced by the
	// normalized cves + component_cve_edges tables. Update callers to use GetCVEsForImage.
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)

	// Count implements search.Searcher for the search framework.
	// NOTE: returns 0 — see Search note above.
	Count(ctx context.Context, q *v1.Query) (int, error)

	// SearchImageCVEs returns search results for the legacy image CVE query path.
	// NOTE: returns empty results — see Search note above.
	SearchImageCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)

	// SearchRawImageCVEs returns raw ImageCVEV2 objects for the legacy GraphQL path.
	// NOTE: returns empty results — see Search note above.
	SearchRawImageCVEs(ctx context.Context, q *v1.Query) ([]*storage.ImageCVEV2, error)

	// GetBatch returns CVE objects by ID for the legacy reporting path.
	// NOTE: returns empty results — see Search note above.
	GetBatch(ctx context.Context, ids []string) ([]*storage.ImageCVEV2, error)

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
