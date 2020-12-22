package common

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// ClusterManager envelopes functions that interact with clusters
type ClusterManager interface {
	UpdateClusterUpgradeStatus(ctx context.Context, clusterID string, status *storage.ClusterUpgradeStatus) error
	UpdateClusterHealth(ctx context.Context, id string, status *storage.ClusterHealthStatus) error
	GetCluster(ctx context.Context, id string) (*storage.Cluster, bool, error)
	GetClusters(ctx context.Context) ([]*storage.Cluster, error)
}

// PolicyManager implements an interface to retrieve policies
type PolicyManager interface {
	GetAllPolicies(ctx context.Context) ([]*storage.Policy, error)
}

// ProcessBaselineManager implements an interface to retrieve process baselines.
type ProcessBaselineManager interface {
	WalkAll(ctx context.Context, fn func(baseline *storage.ProcessBaseline) error) error
}

// NetworkEntityManager implements an interface to retrieve network entities.
//go:generate mockgen-wrapper
type NetworkEntityManager interface {
	GetAllEntitiesForCluster(ctx context.Context, clusterID string) ([]*storage.NetworkEntity, error)
}
