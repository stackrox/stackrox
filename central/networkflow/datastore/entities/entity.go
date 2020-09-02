package entities

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// EntityDataStore stores network graph entities across all clusters.
// Note: Currently only external sources are stored i.e. user-created CIDR blocks
//go:generate mockgen-wrapper
type EntityDataStore interface {
	GetEntity(ctx context.Context, id string) (*storage.NetworkEntity, bool, error)
	GetAllEntitiesForCluster(ctx context.Context, clusterID string) ([]*storage.NetworkEntity, error)
	GetAllEntities(ctx context.Context) ([]*storage.NetworkEntity, error)

	UpsertExternalNetworkEntity(ctx context.Context, entity *storage.NetworkEntity) error
	DeleteExternalNetworkEntity(ctx context.Context, id string) error
	DeleteExternalNetworkEntitiesForCluster(ctx context.Context, clusterID string) error
}
