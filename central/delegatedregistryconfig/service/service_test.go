package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	var cfg *storage.DelegatedRegistryConfig
	var err error

	clustersDS := clusterDSMocks.NewMockDataStore(gomock.NewController(t))
	clustersDS.EXPECT().GetClusters(gomock.Any()).Return(nil, nil).AnyTimes()

	s := serviceImpl{clusterDataStore: clustersDS}

	err = s.validate(context.Background(), cfg)
	assert.ErrorContains(t, err, "config missing")

	cfg = &storage.DelegatedRegistryConfig{}
	cfg.EnabledFor = storage.DelegatedRegistryConfig_ALL
	err = s.validate(context.Background(), cfg)
	assert.ErrorContains(t, err, "defaultClusterId required")

	cfg.EnabledFor = storage.DelegatedRegistryConfig_SPECIFIC
	err = s.validate(context.Background(), cfg)
	assert.ErrorContains(t, err, "defaultClusterId required")
}

func TestGetClusters(t *testing.T) {
	var err error
	var clusters []*v1.DelegatedRegistryCluster

	clustersDS := clusterDSMocks.NewMockDataStore(gomock.NewController(t))

	s := serviceImpl{clusterDataStore: clustersDS}

	healthStatusScannerHealthy := &storage.ClusterHealthStatus{ScannerHealthStatus: storage.ClusterHealthStatus_HEALTHY}
	healthStatusScannerDegraded := &storage.ClusterHealthStatus{ScannerHealthStatus: storage.ClusterHealthStatus_DEGRADED}
	healthStatusEmpty := &storage.ClusterHealthStatus{}

	clustersDS.EXPECT().GetClusters(gomock.Any()).Return(nil, nil)
	clusters, err = s.getClusters(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, clusters)

	clustersDS.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{}, nil)
	clusters, err = s.getClusters(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, clusters)

	cluster := &storage.Cluster{Id: "id", Name: "fake", HealthStatus: nil}
	clustersDS.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{cluster}, nil)
	clusters, err = s.getClusters(context.Background())
	assert.NoError(t, err)
	require.Len(t, clusters, 1)
	assert.False(t, clusters[0].IsValid)

	cluster.HealthStatus = healthStatusEmpty
	clustersDS.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{cluster}, nil)
	clusters, err = s.getClusters(context.Background())
	assert.NoError(t, err)
	require.Len(t, clusters, 1)
	assert.False(t, clusters[0].IsValid)

	cluster.HealthStatus = healthStatusScannerDegraded
	clustersDS.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{cluster}, nil)
	clusters, err = s.getClusters(context.Background())
	assert.NoError(t, err)
	require.Len(t, clusters, 1)
	assert.False(t, clusters[0].IsValid)

	cluster.HealthStatus = healthStatusScannerHealthy
	clustersDS.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{cluster}, nil)
	clusters, err = s.getClusters(context.Background())
	assert.NoError(t, err)
	require.Len(t, clusters, 1)
	assert.True(t, clusters[0].IsValid)

	cluster1 := &storage.Cluster{Id: "id1", HealthStatus: healthStatusScannerHealthy}
	cluster2 := &storage.Cluster{Id: "id2", HealthStatus: healthStatusScannerDegraded}
	clustersDS.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{cluster1, cluster2}, nil)
	clusters, err = s.getClusters(context.Background())
	assert.NoError(t, err)
	require.Len(t, clusters, 2)
	assert.True(t, clusters[0].IsValid)
	assert.Equal(t, clusters[0].Id, "id1")
	assert.False(t, clusters[1].IsValid)
	assert.Equal(t, clusters[1].Id, "id2")

}
