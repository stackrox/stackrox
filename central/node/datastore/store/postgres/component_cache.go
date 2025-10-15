package postgres

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// txWithCache defines an interface for database transactions with optional caching
type txWithCache interface {
	// Access to underlying transaction
	GetTx() *postgres.Tx

	// Cache-aware methods
	GetNodeComponents(ctx context.Context, componentIDs []string) (map[string]*storage.NodeComponent, error)
	GetComponentCVEEdges(ctx context.Context, componentIDs []string) (map[string][]*storage.NodeComponentCVEEdge, error)
}

// transactionCache is a transaction-scoped cache for node components and CVE edges
// This cache is safe to use within a read-only transaction in WalkByQuery
type transactionCache struct {
	*postgres.Tx
	nodeComponents map[string]*storage.NodeComponent
	cveEdges       map[string][]*storage.NodeComponentCVEEdge
}

// transactionNoCache provides a no-cache implementation for single operations
type transactionNoCache struct {
	*postgres.Tx
}

// newTransactionCache creates a new transaction-scoped cache
func newTransactionCache(tx *postgres.Tx) *transactionCache {
	return &transactionCache{
		Tx:             tx,
		nodeComponents: make(map[string]*storage.NodeComponent),
		cveEdges:       make(map[string][]*storage.NodeComponentCVEEdge),
	}
}

// newTransactionNoCache creates a new no-cache transaction wrapper
func newTransactionNoCache(tx *postgres.Tx) *transactionNoCache {
	return &transactionNoCache{Tx: tx}
}

func (tc *transactionCache) GetTx() *postgres.Tx {
	return tc.Tx
}

// GetNodeComponents returns cached components or fetches missing ones from database
func (tc *transactionCache) GetNodeComponents(ctx context.Context, componentIDs []string) (map[string]*storage.NodeComponent, error) {
	result := make(map[string]*storage.NodeComponent)
	var missingIDs []string

	// Check cache first
	for _, id := range componentIDs {
		if component, exists := tc.nodeComponents[id]; exists {
			result[id] = component.CloneVT() // Clone for safety
		} else {
			missingIDs = append(missingIDs, id)
		}
	}

	// Fetch missing components from database
	if len(missingIDs) > 0 {
		dbComponents, err := getNodeComponents(ctx, tc.Tx, missingIDs)
		if err != nil {
			return nil, err
		}

		// Add to cache and result
		for id, component := range dbComponents {
			tc.nodeComponents[id] = component
			result[id] = component.CloneVT() // Clone for result
		}
	}

	return result, nil
}

// GetComponentCVEEdges returns cached CVE edges or fetches missing ones from database
func (tc *transactionCache) GetComponentCVEEdges(ctx context.Context, componentIDs []string) (map[string][]*storage.NodeComponentCVEEdge, error) {
	result := make(map[string][]*storage.NodeComponentCVEEdge)
	var missingIDs []string

	// Check cache first
	for _, id := range componentIDs {
		if edges, exists := tc.cveEdges[id]; exists {
			// Clone edges for safety
			clonedEdges := make([]*storage.NodeComponentCVEEdge, len(edges))
			for i, edge := range edges {
				clonedEdges[i] = edge.CloneVT()
			}
			result[id] = clonedEdges
		} else {
			missingIDs = append(missingIDs, id)
		}
	}

	// Fetch missing edges from database
	if len(missingIDs) > 0 {
		dbEdges, err := getComponentCVEEdges(ctx, tc.Tx, missingIDs)
		if err != nil {
			return nil, err
		}

		// Add to cache and result
		for componentID, edges := range dbEdges {
			// Clone for cache
			cachedEdges := make([]*storage.NodeComponentCVEEdge, len(edges))
			for i, edge := range edges {
				cachedEdges[i] = edge
			}
			tc.cveEdges[componentID] = cachedEdges

			// Clone for result
			resultEdges := make([]*storage.NodeComponentCVEEdge, len(edges))
			for i, edge := range edges {
				resultEdges[i] = edge.CloneVT()
			}
			result[componentID] = resultEdges
		}
	}

	return result, nil
}

func (tnc *transactionNoCache) GetTx() *postgres.Tx {
	return tnc.Tx
}

// GetNodeComponents returns components directly from database without caching
func (tnc *transactionNoCache) GetNodeComponents(ctx context.Context, componentIDs []string) (map[string]*storage.NodeComponent, error) {
	return getNodeComponents(ctx, tnc.Tx, componentIDs)
}

// GetComponentCVEEdges returns CVE edges directly from database without caching
func (tnc *transactionNoCache) GetComponentCVEEdges(ctx context.Context, componentIDs []string) (map[string][]*storage.NodeComponentCVEEdge, error) {
	return getComponentCVEEdges(ctx, tnc.Tx, componentIDs)
}
