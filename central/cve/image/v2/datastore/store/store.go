package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// CVEEdgePair holds a NormalizedCVE and its related component edge.
// Used by converters to transform normalized data to ImageCVEV2 format.
type CVEEdgePair struct {
	CVE  *storage.NormalizedCVE
	Edge *storage.NormalizedComponentCVEEdge
}

// Store provides storage functionality for normalized CVEs.
//
//go:generate mockgen-wrapper
type Store interface {
	// Generated CRUD from pg-table-bindings-wrapper:

	Upsert(ctx context.Context, obj *storage.NormalizedCVE) error
	UpsertMany(ctx context.Context, objs []*storage.NormalizedCVE) error
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error
	Count(ctx context.Context, q *v1.Query) (int, error)
	Exists(ctx context.Context, id string) (bool, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Get(ctx context.Context, id string) (*storage.NormalizedCVE, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.NormalizedCVE, []int, error)
	GetIDs(ctx context.Context) ([]string, error)
	Walk(ctx context.Context, fn func(*storage.NormalizedCVE) error) error

	// GetCVEsWithEdges retrieves CVEs and their component edges for an image.
	// Joins cves → component_cve_edges → image_component_v2 tables.
	// Returns pairs that can be converted to ImageCVEV2.
	GetCVEsWithEdges(ctx context.Context, imageID string) ([]CVEEdgePair, error)

	// GetCVEWithEdge retrieves a single CVE by ID along with its edge for a component.
	// Returns (pair, found, error) where found=false means CVE or edge doesn't exist.
	GetCVEWithEdge(ctx context.Context, cveID string, componentID string) (*CVEEdgePair, bool, error)

	// Custom edge SQL (not generated):

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
