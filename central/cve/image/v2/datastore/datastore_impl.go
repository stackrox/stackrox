package datastore

import (
	"context"
	"errors"

	converter "github.com/stackrox/rox/central/cve/converter/v2"
	edgeStorePkg "github.com/stackrox/rox/central/cve/image/componentcveedge/datastore/store/postgres"
	cveStore "github.com/stackrox/rox/central/cve/image/v2/datastore/store"
	componentStorePkg "github.com/stackrox/rox/central/imagecomponent/v2/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// errStopWalk is a sentinel error to stop walking early.
var errStopWalk = errors.New("stop walk")

type datastoreImpl struct {
	cveStore       cveStore.Store
	edgeStore      edgeStorePkg.Store
	componentStore componentStorePkg.Store
	converter      converter.ImageCVEConverter
}

// Search implements search.Searcher using the generated search framework.
func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.cveStore.Search(ctx, q)
}

// Count returns the number of rows in the cves table matching the query.
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.cveStore.Count(ctx, q)
}

// SearchImageCVEs returns search results synthesized from NormalizedCVE data.
func (ds *datastoreImpl) SearchImageCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	// Use Search() which returns search.Result format.
	searchResults, err := ds.cveStore.Search(ctx, q)
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
	searchResults, err := ds.cveStore.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	if len(searchResults) == 0 {
		return nil, nil
	}

	// For each CVE, get all component edges.
	// This creates one ImageCVEV2 per CVE+component combination.
	// TODO: Add GetEdgesForCVE() to store and implement conversion here.
	var allImageCVEs []*storage.ImageCVEV2
	for range searchResults {
		// Full implementation would iterate and convert each CVE with its edges.
		// For now, we return empty to maintain backward compatibility.
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
	return ds.cveStore.Exists(ctx, id)
}

// Get retrieves a single NormalizedCVE by ID with SAC enforcement.
func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.NormalizedCVE, bool, error) {
	return ds.cveStore.Get(ctx, id)
}

// Upsert inserts or updates a NormalizedCVE row.
func (ds *datastoreImpl) Upsert(ctx context.Context, cve *storage.NormalizedCVE) error {
	return ds.cveStore.Upsert(ctx, cve)
}

// UpsertEdge inserts or updates a component_cve_edges row.
// first_system_occurrence is preserved on conflict (not updated).
// is_fixable, fixed_by, and fix_available_at are refreshed on conflict.
func (ds *datastoreImpl) UpsertEdge(ctx context.Context, edge *storage.NormalizedComponentCVEEdge) error {
	return ds.edgeStore.Upsert(ctx, edge)
}

// DeleteStaleEdges removes edges for a component whose cve_id is NOT in keepCVEIDs.
// If keepCVEIDs is empty, all edges for the component are deleted.
func (ds *datastoreImpl) DeleteStaleEdges(ctx context.Context, componentID string, keepCVEIDs []string) error {
	// Build query for all edges of this component
	componentQuery := pkgSearch.NewQueryBuilder().
		AddExactMatches(pkgSearch.ComponentID, componentID).
		ProtoQuery()

	// If keepCVEIDs is empty, delete all edges for this component
	if len(keepCVEIDs) == 0 {
		return ds.edgeStore.DeleteByQuery(ctx, componentQuery)
	}

	// Otherwise, get all edges for this component and delete stale ones
	keepSet := make(map[string]bool, len(keepCVEIDs))
	for _, cveID := range keepCVEIDs {
		keepSet[cveID] = true
	}

	var staleEdgeIDs []string
	err := ds.edgeStore.WalkByQuery(ctx, componentQuery, func(edge *storage.NormalizedComponentCVEEdge) error {
		if !keepSet[edge.GetCveId()] {
			staleEdgeIDs = append(staleEdgeIDs, edge.GetId())
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(staleEdgeIDs) > 0 {
		return ds.edgeStore.DeleteMany(ctx, staleEdgeIDs)
	}

	return nil
}

// GetCVEsForImage returns all CVEs for a given image (joined through component_cve_edges and image_component_v2).
func (ds *datastoreImpl) GetCVEsForImage(ctx context.Context, imageID string) ([]*storage.NormalizedCVE, error) {
	// Step 1: Get all component IDs for this image
	componentQuery := pkgSearch.NewQueryBuilder().
		AddExactMatches(pkgSearch.ImageSHA, imageID).
		ProtoQuery()

	var componentIDs []string
	err := ds.componentStore.WalkByQuery(ctx, componentQuery, func(comp *storage.ImageComponentV2) error {
		componentIDs = append(componentIDs, comp.GetId())
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(componentIDs) == 0 {
		return nil, nil
	}

	// Step 2: Get all edges for these components
	edgeQuery := pkgSearch.NewQueryBuilder().
		AddExactMatches(pkgSearch.ComponentID, componentIDs...).
		ProtoQuery()

	cveIDSet := make(map[string]bool)
	err = ds.edgeStore.WalkByQuery(ctx, edgeQuery, func(edge *storage.NormalizedComponentCVEEdge) error {
		cveIDSet[edge.GetCveId()] = true
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(cveIDSet) == 0 {
		return nil, nil
	}

	// Step 3: Convert set to slice
	cveIDs := make([]string, 0, len(cveIDSet))
	for cveID := range cveIDSet {
		cveIDs = append(cveIDs, cveID)
	}

	// Step 4: Fetch all CVEs by IDs
	cves, _, err := ds.cveStore.GetMany(ctx, cveIDs)
	if err != nil {
		return nil, err
	}

	return cves, nil
}

// GetAllReferencedCVEs returns all CVEs referenced by at least one component_cve_edges row.
func (ds *datastoreImpl) GetAllReferencedCVEs(ctx context.Context) ([]*storage.NormalizedCVE, error) {
	// Step 1: Walk all edges to collect unique CVE IDs
	cveIDSet := make(map[string]bool)
	err := ds.edgeStore.Walk(ctx, func(edge *storage.NormalizedComponentCVEEdge) error {
		cveIDSet[edge.GetCveId()] = true
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(cveIDSet) == 0 {
		return nil, nil
	}

	// Step 2: Convert set to slice
	cveIDs := make([]string, 0, len(cveIDSet))
	for cveID := range cveIDSet {
		cveIDs = append(cveIDs, cveID)
	}

	// Step 3: Fetch CVEs in a single batch (GetMany handles batching internally if needed)
	cves, _, err := ds.cveStore.GetMany(ctx, cveIDs)
	if err != nil {
		return nil, err
	}

	return cves, nil
}

// DeleteOrphanedCVEsBatch deletes up to batchSize CVEs with no referencing edges.
// Returns the number of rows deleted.
func (ds *datastoreImpl) DeleteOrphanedCVEsBatch(ctx context.Context, batchSize int) (int64, error) {
	if batchSize <= 0 {
		return 0, errox.InvalidArgs.New("batchSize must be positive")
	}

	// Step 1: Build set of all referenced CVE IDs
	referencedCVEs := make(map[string]bool)
	err := ds.edgeStore.Walk(ctx, func(edge *storage.NormalizedComponentCVEEdge) error {
		referencedCVEs[edge.GetCveId()] = true
		return nil
	})
	if err != nil {
		return 0, err
	}

	// Step 2: Walk CVEs and find orphans (CVEs not in the referenced set)
	orphanIDs := make([]string, 0, batchSize)
	err = ds.cveStore.Walk(ctx, func(cve *storage.NormalizedCVE) error {
		if !referencedCVEs[cve.GetId()] {
			orphanIDs = append(orphanIDs, cve.GetId())
			// Stop early once we have enough orphans
			if len(orphanIDs) >= batchSize {
				return errStopWalk
			}
		}
		return nil
	})
	// Ignore the sentinel error we used to stop early
	if err != nil && !errors.Is(err, errStopWalk) {
		return 0, err
	}

	// Step 3: Delete the orphaned CVEs
	if len(orphanIDs) > 0 {
		err := ds.cveStore.DeleteMany(ctx, orphanIDs)
		if err != nil {
			return 0, err
		}
		return int64(len(orphanIDs)), nil
	}

	return 0, nil
}
