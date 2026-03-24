package datastore

import (
	"context"
	"testing"

	converter "github.com/stackrox/rox/central/cve/converter/v2"
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
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)

	// Count returns the number of rows in the cves table matching the query.
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

	// Exists returns true if a CVE row with the given UUID exists in the cves table.
	Exists(ctx context.Context, id string) (bool, error)

	// Get retrieves a single NormalizedCVE by ID with SAC enforcement.
	// Returns (cve, found, error) where found=false means CVE doesn't exist or access denied.
	Get(ctx context.Context, id string) (*storage.NormalizedCVE, bool, error)

	// Upsert inserts or updates a NormalizedCVE row.
	Upsert(ctx context.Context, cve *storage.NormalizedCVE) error

	// UpsertEdge inserts or updates a component_cve_edges row.
	// first_system_occurrence is preserved on conflict (not updated).
	// is_fixable, fixed_by, and fix_available_at are refreshed on conflict.
	UpsertEdge(ctx context.Context, edge *storage.NormalizedComponentCVEEdge) error

	// DeleteStaleEdges removes edges for a component whose cve_id is NOT in keepCVEIDs.
	// If keepCVEIDs is empty, all edges for the component are deleted.
	DeleteStaleEdges(ctx context.Context, componentID string, keepCVEIDs []string) error

	// GetCVEsForImage returns all CVEs for a given image (joined through component_cve_edges and image_component_v2).
	GetCVEsForImage(ctx context.Context, imageID string) ([]*storage.NormalizedCVE, error)

	// GetAllReferencedCVEs returns all CVEs referenced by at least one component_cve_edges row.
	GetAllReferencedCVEs(ctx context.Context) ([]*storage.NormalizedCVE, error)

	// DeleteOrphanedCVEsBatch deletes up to batchSize CVEs with no referencing edges.
	// Returns the number of rows deleted.
	DeleteOrphanedCVEsBatch(ctx context.Context, batchSize int) (int64, error)
}

// New returns a new instance of a DataStore.
func New(storage store.Store) DataStore {
	ds := &datastoreImpl{
		storage:   storage,
		converter: converter.NewImageCVEConverter(),
	}
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	dbstore := pgStore.NewCombined(pool)
	return New(dbstore)
}
