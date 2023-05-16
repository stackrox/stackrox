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

	none     = v1.DelegatedRegistryConfig_NONE
	all      = v1.DelegatedRegistryConfig_ALL
	specific = v1.DelegatedRegistryConfig_SPECIFIC

	empty = &v1.Empty{}

	errBroken = errors.New("broken")
)

func TestGetConfigSuccess(t *testing.T) {
	var err error
	var cfg *v1.DelegatedRegistryConfig

	deleClusterDS := deleDSMocks.NewMockDataStore(gomock.NewController(t))
	s := New(deleClusterDS, nil)

	deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(nil, false, nil)
	cfg, err = s.GetConfig(context.Background(), empty)
	assert.NoError(t, err)
	assert.Empty(t, cfg)

	retVal := &storage.DelegatedRegistryConfig{EnabledFor: storage.DelegatedRegistryConfig_SPECIFIC, DefaultClusterId: "id1"}
	deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(retVal, true, nil)
	cfg, err = s.GetConfig(context.Background(), empty)
	assert.NoError(t, err)
	assert.Equal(t, cfg.EnabledFor, specific)
	assert.Equal(t, cfg.DefaultClusterId, "id1")
}

func TestGetConfigError(t *testing.T) {
	var err error

	deleClusterDS := deleDSMocks.NewMockDataStore(gomock.NewController(t))
	s := New(deleClusterDS, nil)

	deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(nil, false, errBroken)
	_, err = s.GetConfig(context.Background(), empty)
	assert.ErrorContains(t, err, "retrieving config")
}

func TestGetClustersSuccess(t *testing.T) {
	var err error
	var resp *v1.DelegatedRegistryClustersResponse

	clustersDS := clusterDSMocks.NewMockDataStore(gomock.NewController(t))
	deleClusterDS := deleDSMocks.NewMockDataStore(gomock.NewController(t))
	s := New(deleClusterDS, clustersDS)

	genClusters := func(healthStatus *storage.ClusterHealthStatus) []*storage.Cluster {
		return []*storage.Cluster{{Id: "id", Name: "fake", HealthStatus: healthStatus}}
	}

	tt := []struct {
		name     string
		clusters []*storage.Cluster
		valid    bool
	}{
		{"missing health", genClusters(nil), false},
		{"empty health", genClusters(clusterHealthStatusEmpty), false},
		{"degraded", genClusters(clusterHealthStatusScannerDegraded), false},
		{"healthy", genClusters(clusterHealthStatusScannerHealthy), true}, // only healthy scanners are valid
	}

	for _, test := range tt {
		tf := func(t *testing.T) {
			clustersDS.EXPECT().GetClusters(gomock.Any()).Return(test.clusters, nil)
			resp, err = s.GetClusters(context.Background(), empty)
			assert.NoError(t, err)
			require.Len(t, resp.Clusters, 1)
			assert.Equal(t, resp.Clusters[0].IsValid, test.valid)
		}

		t.Run(test.name, tf)
	}

	t.Run("multi cluster", func(t *testing.T) {
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
	})

}

func TestGetClustersError(t *testing.T) {
	var err error

	var resp *v1.DelegatedRegistryClustersResponse
	clustersDS := clusterDSMocks.NewMockDataStore(gomock.NewController(t))
	deleClusterDS := deleDSMocks.NewMockDataStore(gomock.NewController(t))

	s := New(deleClusterDS, clustersDS)

	tt := []struct {
		name           string
		clusters       []*storage.Cluster
		err            error
		expectedErrMsg string
	}{
		{"cluster ds error", nil, errBroken, "retrieving clusters"},
		{"nil cluster ds response ", nil, nil, "no valid clusters"},
		{"empty cluster ds response ", []*storage.Cluster{}, nil, "no valid clusters"},
	}

	for _, test := range tt {
		tf := func(t *testing.T) {
			clustersDS.EXPECT().GetClusters(gomock.Any()).Return(test.clusters, test.err)
			resp, err = s.GetClusters(context.Background(), empty)
			assert.Nil(t, resp)
			assert.ErrorContains(t, err, test.expectedErrMsg)
		}

		t.Run(test.name, tf)
	}
}

func TestPutConfigError(t *testing.T) {
	var err error

	genCfg := func(ef v1.DelegatedRegistryConfig_EnabledFor, defId string, regIds []string) *v1.DelegatedRegistryConfig {
		regs := make([]*v1.DelegatedRegistryConfig_DelegatedRegistry, len(regIds))
		for i, id := range regIds {
			regs[i] = &v1.DelegatedRegistryConfig_DelegatedRegistry{ClusterId: id}
		}

		return &v1.DelegatedRegistryConfig{
			EnabledFor:       ef,
			DefaultClusterId: defId,
			Registries:       regs,
		}
	}

	multiClusters := []*storage.Cluster{
		{Id: "id1", HealthStatus: clusterHealthStatusScannerHealthy},
		{Id: "id2", HealthStatus: clusterHealthStatusScannerDegraded},
	}

	tt := []struct {
		name           string
		cfg            *v1.DelegatedRegistryConfig
		deleDSErr      error
		clusterDSErr   error
		expectedErrMsg string
		clusters       []*storage.Cluster
	}{
		{"nil config", nil, nil, nil, "config missing", nil},
		{"upsert failed", genCfg(none, "", nil), errBroken, nil, "upserting config", nil},
		{"enabled for all missing default id", genCfg(all, "", nil), nil, nil, "default cluster id required", nil},
		{"enabled for specific missing default id", genCfg(specific, "", nil), nil, nil, "default cluster id required", nil},
		{"cluster ds error", genCfg(specific, "fake", nil), nil, errBroken, "broken", nil},
		{"multi cluster invalid default id", genCfg(specific, "fake", nil), nil, nil, "is not valid", multiClusters},
		{"multi cluster invalid registry id and path", genCfg(specific, "fake", []string{"fake"}), nil, nil, "is not valid", multiClusters},
		{"multi cluster invalid registry id and path", genCfg(specific, "fake", []string{"fake"}), nil, nil, "missing registry path", multiClusters},
		{"multi cluster invalid registry path", genCfg(specific, "fake", []string{"id1"}), nil, nil, "missing registry path", multiClusters},
	}

	clustersDS := clusterDSMocks.NewMockDataStore(gomock.NewController(t))
	deleClusterDS := deleDSMocks.NewMockDataStore(gomock.NewController(t))
	s := New(deleClusterDS, clustersDS)
	for _, test := range tt {
		tf := func(t *testing.T) {
			if test.deleDSErr != nil {
				deleClusterDS.EXPECT().UpsertConfig(gomock.Any(), gomock.Any()).Return(test.deleDSErr)
			}

			if len(test.clusters) > 0 || test.clusterDSErr != nil {
				clustersDS.EXPECT().GetClusters(gomock.Any()).Return(test.clusters, test.clusterDSErr)
			}

			_, err = s.UpdateConfig(context.Background(), test.cfg)
			assert.ErrorContains(t, err, test.expectedErrMsg)
		}

		t.Run(test.name, tf)
	}
}

func TestPutConfigSuccess(t *testing.T) {
	var err error
	var cfg *v1.DelegatedRegistryConfig

	clustersDS := clusterDSMocks.NewMockDataStore(gomock.NewController(t))
	deleClusterDS := deleDSMocks.NewMockDataStore(gomock.NewController(t))
	s := New(deleClusterDS, clustersDS)
	cluster1 := &storage.Cluster{Id: "id1", HealthStatus: clusterHealthStatusScannerHealthy}
	cluster2 := &storage.Cluster{Id: "id2", HealthStatus: clusterHealthStatusScannerDegraded}
	clustersDS.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{cluster1, cluster2}, nil).AnyTimes()

	cfg = &v1.DelegatedRegistryConfig{EnabledFor: specific, DefaultClusterId: "id1"}
	deleClusterDS.EXPECT().UpsertConfig(gomock.Any(), gomock.Any())
	cfg.DefaultClusterId = "id1"
	_, err = s.UpdateConfig(context.Background(), cfg)
	assert.NoError(t, err)

	deleClusterDS.EXPECT().UpsertConfig(gomock.Any(), gomock.Any())
	cfg.Registries = []*v1.DelegatedRegistryConfig_DelegatedRegistry{{ClusterId: "id1", RegistryPath: "something"}}
	_, err = s.UpdateConfig(context.Background(), cfg)
	assert.NoError(t, err)
}
