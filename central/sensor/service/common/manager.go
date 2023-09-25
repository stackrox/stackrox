package common

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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

// DelegatedRegistryConfigManager defines an interface to retrieve the delegated registry config.
type DelegatedRegistryConfigManager interface {
	GetConfig(ctx context.Context) (*storage.DelegatedRegistryConfig, bool, error)
}

// ImageIntegrationManager defines an interface to retrieve image integrations.
type ImageIntegrationManager interface {
	GetImageIntegrations(ctx context.Context, integration *v1.GetImageIntegrationsRequest) ([]*storage.ImageIntegration, error)
}

// NetworkEntityManager implements an interface to retrieve network entities.
//
//go:generate mockgen-wrapper
type NetworkEntityManager interface {
	GetAllEntitiesForCluster(ctx context.Context, clusterID string) ([]*storage.NetworkEntity, error)
}

// ComplianceOperatorManager implements an interface to process scan request responses
//
//go:generate mockgen-wrapper
type ComplianceOperatorManager interface {
	HandleScanRequestResponse(ctx context.Context, requestID string, clusterID string, responsePayload string) error
}
