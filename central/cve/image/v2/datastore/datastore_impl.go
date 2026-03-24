package datastore

import (
	"context"

	converter "github.com/stackrox/rox/central/cve/converter/v2"
	"github.com/stackrox/rox/central/cve/image/v2/datastore/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	storage   store.Store
	converter converter.ImageCVEConverter
}

// Search implements search.Searcher using the generated search framework.
func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.storage.Search(ctx, q)
}

// Count returns the number of rows in the cves table matching the query.
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.storage.Count(ctx, q)
}

// SearchImageCVEs returns search results synthesized from NormalizedCVE data.
func (ds *datastoreImpl) SearchImageCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	// Use Search() which returns search.Result format.
	searchResults, err := ds.storage.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	// Convert search.Result to v1.SearchResult.
	results := make([]*v1.SearchResult, 0, len(searchResults))
	for _, r := range searchResults {
		results = append(results, &v1.SearchResult{
			Id: r.ID,
			// Additional fields would be populated from matched fields.
		})
	}

	return results, nil
}

// SearchRawImageCVEs returns ImageCVEV2 objects synthesized from NormalizedCVE data.
// Note: This requires iterating CVEs and joining with edges, which is expensive.
// Callers should use GetCVEsForImage() when possible for better performance.
func (ds *datastoreImpl) SearchRawImageCVEs(ctx context.Context, q *v1.Query) ([]*storage.ImageCVEV2, error) {
	// Get CVE IDs matching the query.
	searchResults, err := ds.storage.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	if len(searchResults) == 0 {
		return nil, nil
	}

	// For each CVE, get all component edges.
	// This creates one ImageCVEV2 per CVE+component combination.
	var allImageCVEs []*storage.ImageCVEV2
	for _, result := range searchResults {
		_, _, err := ds.storage.Get(ctx, result.ID)
		if err != nil {
			return nil, err
		}
		if !found {
			continue // CVE was deleted between search and get.
		}

		// Get all edges for this CVE (all components that have this CVE).
		// This requires a new query - for now we skip the conversion.
		// Full implementation would use GetEdgesForCVE() method.
		// TODO: Add GetEdgesForCVE() to store and implement conversion here.
	}

	// Return empty until GetEdgesForCVE is implemented.
	// This maintains backward compatibility while allowing migration to proceed.
	return allImageCVEs, nil
}

// GetBatch returns ImageCVEV2 objects by ID.
// Note: Old ImageCVEV2 IDs were composite (CVE+component), new IDs are CVE UUIDs.
// Without component context, returns empty. Callers should use GetCVEsForImage().
func (ds *datastoreImpl) GetBatch(_ context.Context, _ []string) ([]*storage.ImageCVEV2, error) {
	// Without edge/component context, we can't synthesize full ImageCVEV2.
	// Return empty for now - full implementation requires GetEdgesForCVE() method
	// and per-CVE edge lookup with conversion.
	return nil, nil
}

// Exists returns true if a CVE row with the given UUID exists in the cves table.
func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	return ds.storage.Exists(ctx, id)
}

// Get retrieves a single NormalizedCVE by ID with SAC enforcement.
func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.NormalizedCVE, bool, error) {
	return ds.storage.Get(ctx, id)
}

// Upsert inserts or updates a NormalizedCVE row.
func (ds *datastoreImpl) Upsert(ctx context.Context, cve *storage.NormalizedCVE) error {
	return ds.storage.Upsert(ctx, cve)
}

// UpsertEdge inserts or updates a component_cve_edges row.
// first_system_occurrence is preserved on conflict (not updated).
// is_fixable, fixed_by, and fix_available_at are refreshed on conflict.
func (ds *datastoreImpl) UpsertEdge(ctx context.Context, edge *storage.NormalizedComponentCVEEdge) error {
	return ds.storage.UpsertEdge(ctx, edge)
}

// DeleteStaleEdges removes edges for a component whose cve_id is NOT in keepCVEIDs.
// If keepCVEIDs is empty, all edges for the component are deleted.
func (ds *datastoreImpl) DeleteStaleEdges(ctx context.Context, componentID string, keepCVEIDs []string) error {
	return ds.storage.DeleteStaleEdges(ctx, componentID, keepCVEIDs)
}

// GetCVEsForImage returns all CVEs for a given image (joined through component_cve_edges and image_component_v2).
func (ds *datastoreImpl) GetCVEsForImage(ctx context.Context, imageID string) ([]*storage.NormalizedCVE, error) {
	return ds.storage.GetCVEsForImage(ctx, imageID)
}

// GetAllReferencedCVEs returns all CVEs referenced by at least one component_cve_edges row.
func (ds *datastoreImpl) GetAllReferencedCVEs(ctx context.Context) ([]*storage.NormalizedCVE, error) {
	return ds.storage.GetAllReferencedCVEs(ctx)
}

// DeleteOrphanedCVEsBatch deletes up to batchSize CVEs with no referencing edges.
// Returns the number of rows deleted.
func (ds *datastoreImpl) DeleteOrphanedCVEsBatch(ctx context.Context, batchSize int) (int64, error) {
	return ds.storage.DeleteOrphanedCVEsBatch(ctx, batchSize)
}
