package common

import (
	"context"

	"github.com/stackrox/stackrox/generated/storage"
)

// ClusterManager envelopes functions that interact with clusters
type ClusterManager interface {
	UpdateClusterUpgradeStatus(ctx context.Context, clusterID string, status *storage.ClusterUpgradeStatus) error
	UpdateClusterHealth(ctx context.Context, id string, status *storage.ClusterHealthStatus) error
	UpdateSensorDeploymentIdentification(ctx context.Context, clusterID string, identification *storage.SensorDeploymentIdentification) error
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

// NetworkBaselineManager implements an interface to retrieve network baselines.
type NetworkBaselineManager interface {
	Walk(ctx context.Context, fn func(baseline *storage.NetworkBaseline) error) error
}

// NetworkEntityManager implements an interface to retrieve network entities.
//go:generate mockgen-wrapper
type NetworkEntityManager interface {
	GetAllEntitiesForCluster(ctx context.Context, clusterID string) ([]*storage.NetworkEntity, error)
}
