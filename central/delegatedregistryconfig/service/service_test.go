package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	deleDSMocks "github.com/stackrox/rox/central/delegatedregistryconfig/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	clusterHealthStatusScannerHealthy  = &storage.ClusterHealthStatus{ScannerHealthStatus: storage.ClusterHealthStatus_HEALTHY}
	clusterHealthStatusScannerDegraded = &storage.ClusterHealthStatus{ScannerHealthStatus: storage.ClusterHealthStatus_DEGRADED}
	clusterHealthStatusEmpty           = &storage.ClusterHealthStatus{}

	empty = &v1.Empty{}
)

func TestGetConfig(t *testing.T) {
	var err error
	var cfg *storage.DelegatedRegistryConfig
	_ = cfg

	s := New(nil, nil)
	_, err = s.GetConfig(context.Background(), nil)
	assert.ErrorContains(t, err, "not initialized")

	deleClusterDS := deleDSMocks.NewMockDataStore(gomock.NewController(t))
	s = New(deleClusterDS, nil)

	deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(nil, errors.New("broken"))
	_, err = s.GetConfig(context.Background(), empty)
	assert.ErrorContains(t, err, "retrieving config")

	deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(nil, nil)
	cfg, err = s.GetConfig(context.Background(), empty)
	assert.NoError(t, err)
	assert.Empty(t, cfg)

	deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(&storage.DelegatedRegistryConfig{
		EnabledFor:       storage.DelegatedRegistryConfig_SPECIFIC,
		DefaultClusterId: "id1",
	}, nil)
	cfg, err = s.GetConfig(context.Background(), empty)
	assert.NoError(t, err)
	assert.Equal(t, cfg.EnabledFor, storage.DelegatedRegistryConfig_SPECIFIC)
	assert.Equal(t, cfg.DefaultClusterId, "id1")
}

func TestGetClusters(t *testing.T) {
	var err error
	var resp *v1.DelegatedRegistryClustersResponse

	clustersDS := clusterDSMocks.NewMockDataStore(gomock.NewController(t))
	deleClusterDS := deleDSMocks.NewMockDataStore(gomock.NewController(t))

	s := New(nil, nil)

	_, err = s.GetClusters(context.Background(), nil)
	assert.ErrorContains(t, err, "not initialized")

	s = New(deleClusterDS, clustersDS)

	clustersDS.EXPECT().GetClusters(gomock.Any()).Return(nil, errors.New("broken"))
	resp, err = s.GetClusters(context.Background(), empty)
	assert.Nil(t, resp)
	assert.ErrorContains(t, err, "retrieving clusters")

	clustersDS.EXPECT().GetClusters(gomock.Any()).Return(nil, nil)
	resp, err = s.GetClusters(context.Background(), empty)
	assert.Nil(t, resp)
	assert.ErrorContains(t, err, "no valid clusters")

	clustersDS.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{}, nil)
	resp, err = s.GetClusters(context.Background(), empty)
	assert.Nil(t, resp)
	assert.ErrorContains(t, err, "no valid clusters")

	cluster := &storage.Cluster{Id: "id", Name: "fake", HealthStatus: nil}
	clustersDS.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{cluster}, nil)
	resp, err = s.GetClusters(context.Background(), empty)
	assert.NoError(t, err)
	require.Len(t, resp.Clusters, 1)
	assert.False(t, resp.Clusters[0].IsValid)

	cluster.HealthStatus = clusterHealthStatusEmpty
	clustersDS.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{cluster}, nil)
	resp, err = s.GetClusters(context.Background(), empty)
	assert.NoError(t, err)
	require.Len(t, resp.Clusters, 1)
	assert.False(t, resp.Clusters[0].IsValid)

	cluster.HealthStatus = clusterHealthStatusScannerDegraded
	clustersDS.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{cluster}, nil)
	resp, err = s.GetClusters(context.Background(), empty)
	assert.NoError(t, err)
	require.Len(t, resp.Clusters, 1)
	assert.False(t, resp.Clusters[0].IsValid)

	cluster.HealthStatus = clusterHealthStatusScannerHealthy
	clustersDS.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{cluster}, nil)
	resp, err = s.GetClusters(context.Background(), empty)
	assert.NoError(t, err)
	require.Len(t, resp.Clusters, 1)
	assert.True(t, resp.Clusters[0].IsValid)

	cluster1 := &storage.Cluster{Id: "id1", HealthStatus: clusterHealthStatusScannerHealthy}
	cluster2 := &storage.Cluster{Id: "id2", HealthStatus: clusterHealthStatusScannerDegraded}
	clustersDS.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{cluster1, cluster2}, nil)
	resp, err = s.GetClusters(context.Background(), empty)
	assert.NoError(t, err)
	require.Len(t, resp.Clusters, 2)
	assert.True(t, resp.Clusters[0].IsValid)
	assert.Equal(t, resp.Clusters[0].Id, "id1")
	assert.False(t, resp.Clusters[1].IsValid)
	assert.Equal(t, resp.Clusters[1].Id, "id2")
}

func TestPutConfig(t *testing.T) {
	var err error
	var cfg *storage.DelegatedRegistryConfig

	s := New(nil, nil)

	// error scenarios
	_, err = s.PutConfig(context.Background(), nil)
	assert.ErrorContains(t, err, "not initialized")

	deleClusterDS := deleDSMocks.NewMockDataStore(gomock.NewController(t))
	s = New(deleClusterDS, nil)

	_, err = s.PutConfig(context.Background(), cfg)
	assert.ErrorContains(t, err, "config missing")

	cfg = &storage.DelegatedRegistryConfig{}
	cfg.EnabledFor = storage.DelegatedRegistryConfig_NONE
	deleClusterDS.EXPECT().UpsertConfig(gomock.Any(), gomock.Any()).Return(errors.New("broken"))
	_, err = s.PutConfig(context.Background(), cfg)
	assert.ErrorContains(t, err, "upserting config")

	cfg.EnabledFor = storage.DelegatedRegistryConfig_ALL
	_, err = s.PutConfig(context.Background(), cfg)
	assert.ErrorContains(t, err, "defaultClusterId required")

	cfg.EnabledFor = storage.DelegatedRegistryConfig_SPECIFIC
	_, err = s.PutConfig(context.Background(), cfg)
	assert.ErrorContains(t, err, "defaultClusterId required")

	clustersDS := clusterDSMocks.NewMockDataStore(gomock.NewController(t))
	clustersDS.EXPECT().GetClusters(gomock.Any()).Return(nil, errors.New("broken")).AnyTimes()
	s = New(deleClusterDS, clustersDS)
	cfg.DefaultClusterId = "fake-id"
	_, err = s.PutConfig(context.Background(), cfg)
	assert.ErrorContains(t, err, "broken")

	clustersDS = clusterDSMocks.NewMockDataStore(gomock.NewController(t))
	s = New(deleClusterDS, clustersDS)
	cluster1 := &storage.Cluster{Id: "id1", HealthStatus: clusterHealthStatusScannerHealthy}
	cluster2 := &storage.Cluster{Id: "id2", HealthStatus: clusterHealthStatusScannerDegraded}
	clustersDS.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{cluster1, cluster2}, nil).AnyTimes()

	cfg.DefaultClusterId = "fake-id"
	_, err = s.PutConfig(context.Background(), cfg)
	assert.ErrorContains(t, err, "not a valid cluster")

	deleClusterDS.EXPECT().UpsertConfig(gomock.Any(), gomock.Any()).MinTimes(2)
	cfg.DefaultClusterId = "id1"
	_, err = s.PutConfig(context.Background(), cfg)
	assert.NoError(t, err) // successful upsert

	cfg.Registries = []*storage.DelegatedRegistryConfig_DelegatedRegistry{{ClusterId: "id1", RegistryPath: "something"}}
	_, err = s.PutConfig(context.Background(), cfg)
	assert.NoError(t, err) // successful upsert

	cfg.Registries = []*storage.DelegatedRegistryConfig_DelegatedRegistry{{ClusterId: "fake-id"}}
	_, err = s.PutConfig(context.Background(), cfg)
	assert.ErrorContains(t, err, "is not valid")
	assert.ErrorContains(t, err, "missing registry path")

	cfg.Registries = []*storage.DelegatedRegistryConfig_DelegatedRegistry{{ClusterId: "id1"}}
	_, err = s.PutConfig(context.Background(), cfg)
	assert.NotContains(t, err.Error(), "is not valid")
	assert.ErrorContains(t, err, "missing registry path")

}
