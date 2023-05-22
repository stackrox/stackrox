package datastore

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// EntityDataStore stores network graph entities across all clusters.
// Note: Currently only external sources are stored i.e. user-created CIDR blocks
//
//go:generate mockgen-wrapper
type EntityDataStore interface {
	Exists(ctx context.Context, id string) (bool, error)
	GetIDs(ctx context.Context) ([]string, error)
	// This getter does not respect the current graph configuration.
	GetEntity(ctx context.Context, id string) (*storage.NetworkEntity, bool, error)
	// This getter respects the current graph configuration.
	GetAllEntitiesForCluster(ctx context.Context, clusterID string) ([]*storage.NetworkEntity, error)
	// This getter respects the current graph configuration.
	GetAllEntities(ctx context.Context) ([]*storage.NetworkEntity, error)
	// This getter respects only the predicate and not the current graph configuration.
	GetAllMatchingEntities(ctx context.Context, pred func(entity *storage.NetworkEntity) bool) ([]*storage.NetworkEntity, error)

	CreateExternalNetworkEntity(ctx context.Context, entity *storage.NetworkEntity, skipPush bool) error
	UpdateExternalNetworkEntity(ctx context.Context, entity *storage.NetworkEntity, skipPush bool) error

	CreateExtNetworkEntitiesForCluster(ctx context.Context, cluster string, entities ...*storage.NetworkEntity) ([]string, error)

	DeleteExternalNetworkEntity(ctx context.Context, id string) error
	DeleteExternalNetworkEntitiesForCluster(ctx context.Context, clusterID string) error

	RegisterCluster(ctx context.Context, clusterID string)
}
