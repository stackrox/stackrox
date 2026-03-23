package datastore

import (
	"context"

	"github.com/stackrox/rox/central/cve/image/v2/datastore/store"
)

type datastoreImpl struct {
	storage store.Store
}

// UpsertCVE inserts a CVE row if it doesn't exist (two-phase: insert then fetch).
// Returns the UUID of the CVE row (whether newly inserted or pre-existing).
func (ds *datastoreImpl) UpsertCVE(ctx context.Context, cveRow *store.CVERow) (string, error) {
	return ds.storage.UpsertCVE(ctx, cveRow)
}

// UpsertEdge inserts or updates a component_cve_edges row.
// first_system_occurrence is preserved on conflict (not updated).
// is_fixable and fixed_by are refreshed on conflict.
func (ds *datastoreImpl) UpsertEdge(ctx context.Context, edge *store.EdgeRow) error {
	return ds.storage.UpsertEdge(ctx, edge)
}

// DeleteStaleEdges removes edges for a component whose cve_id is NOT in keepCVEIDs.
// If keepCVEIDs is empty, all edges for the component are deleted.
func (ds *datastoreImpl) DeleteStaleEdges(ctx context.Context, componentID string, keepCVEIDs []string) error {
	return ds.storage.DeleteStaleEdges(ctx, componentID, keepCVEIDs)
}

// GetCVEsForImage returns all CVEs for a given image (joined through component_cve_edges and image_component_v2).
func (ds *datastoreImpl) GetCVEsForImage(ctx context.Context, imageID string) ([]*store.CVERow, error) {
	return ds.storage.GetCVEsForImage(ctx, imageID)
}

// GetAllReferencedCVEs returns all CVEs referenced by at least one component_cve_edges row.
func (ds *datastoreImpl) GetAllReferencedCVEs(ctx context.Context) ([]*store.CVERow, error) {
	return ds.storage.GetAllReferencedCVEs(ctx)
}

// DeleteOrphanedCVEsBatch deletes up to batchSize CVEs with no referencing edges.
// Returns number of rows deleted.
func (ds *datastoreImpl) DeleteOrphanedCVEsBatch(ctx context.Context, batchSize int) (int64, error) {
	return ds.storage.DeleteOrphanedCVEsBatch(ctx, batchSize)
}
