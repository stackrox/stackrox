package postgres

import (
	"context"

	storeParent "github.com/stackrox/rox/central/cve/image/v2/datastore/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// combinedStore satisfies storeParent.Store by embedding the generated Store
// and delegating edge operations to edgeStoreImpl.
type combinedStore struct {
	Store
	edges EdgeStore
}

// NewCombined returns a Store backed by both the generated NormalizedCVE CRUD store
// and the custom edge SQL operations.
func NewCombined(db postgres.DB) storeParent.Store {
	return &combinedStore{
		Store: New(db),
		edges: NewEdgeStore(db),
	}
}

// UpsertEdge delegates to the edge store.
func (c *combinedStore) UpsertEdge(ctx context.Context, edge *storage.NormalizedComponentCVEEdge) error {
	return c.edges.UpsertEdge(ctx, edge)
}

// DeleteStaleEdges delegates to the edge store.
func (c *combinedStore) DeleteStaleEdges(ctx context.Context, componentID string, keepCVEIDs []string) error {
	return c.edges.DeleteStaleEdges(ctx, componentID, keepCVEIDs)
}

// GetCVEsForImage delegates to the edge store.
func (c *combinedStore) GetCVEsForImage(ctx context.Context, imageID string) ([]*storage.NormalizedCVE, error) {
	return c.edges.GetCVEsForImage(ctx, imageID)
}

// GetAllReferencedCVEs delegates to the edge store.
func (c *combinedStore) GetAllReferencedCVEs(ctx context.Context) ([]*storage.NormalizedCVE, error) {
	return c.edges.GetAllReferencedCVEs(ctx)
}

// DeleteOrphanedCVEsBatch delegates to the edge store.
func (c *combinedStore) DeleteOrphanedCVEsBatch(ctx context.Context, batchSize int) (int64, error) {
	return c.edges.DeleteOrphanedCVEsBatch(ctx, batchSize)
}
