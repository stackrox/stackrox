package datastore

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// EntityDataStore stores network graph entities across all clusters.
// Note: Currently only external sources are stored i.e. user-created CIDR blocks
//go:generate mockgen-wrapper
type EntityDataStore interface {
	// This getter does not respect the current graph configuration.
	GetEntity(ctx context.Context, id string) (*storage.NetworkEntity, bool, error)
	// This getter respects the current graph configuration.
	GetAllEntitiesForCluster(ctx context.Context, clusterID string) ([]*storage.NetworkEntity, error)
	// This getter respects the current graph configuration.
	GetAllEntities(ctx context.Context) ([]*storage.NetworkEntity, error)
	// This getter respects only the predicate and not the current graph configuration.
	GetAllMatchingEntities(ctx context.Context, pred func(entity *storage.NetworkEntity) bool) ([]*storage.NetworkEntity, error)

	UpsertExternalNetworkEntity(ctx context.Context, entity *storage.NetworkEntity, skipPush bool) error
	DeleteExternalNetworkEntity(ctx context.Context, id string) error
	DeleteExternalNetworkEntitiesForCluster(ctx context.Context, clusterID string) error

	RegisterCluster(clusterID string)
}
