package service

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	deleDSMocks "github.com/stackrox/rox/central/delegatedregistryconfig/datastore/mocks"
	connMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	none     = v1.DelegatedRegistryConfig_NONE
	all      = v1.DelegatedRegistryConfig_ALL
	specific = v1.DelegatedRegistryConfig_SPECIFIC

	empty = &v1.Empty{}

	errBroken = errors.New("broken")
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestGetConfigSuccess(t *testing.T) {
	var err error
	var cfg *v1.DelegatedRegistryConfig

	deleClusterDS := deleDSMocks.NewMockDataStore(gomock.NewController(t))
	s := New(deleClusterDS, nil, nil)

	t.Run("empty", func(t *testing.T) {
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(nil, false, nil)
		cfg, err = s.GetConfig(context.Background(), empty)
		assert.NoError(t, err)
		assert.Empty(t, cfg)
	})

	t.Run("specific and default cluster", func(t *testing.T) {
		retVal := &storage.DelegatedRegistryConfig{EnabledFor: storage.DelegatedRegistryConfig_SPECIFIC, DefaultClusterId: "id1"}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(retVal, true, nil)
		cfg, err = s.GetConfig(context.Background(), empty)
		assert.NoError(t, err)
		assert.Equal(t, cfg.EnabledFor, specific)
		assert.Equal(t, cfg.DefaultClusterId, "id1")
	})
}

func TestGetConfigError(t *testing.T) {
	var err error

	deleClusterDS := deleDSMocks.NewMockDataStore(gomock.NewController(t))
	s := New(deleClusterDS, nil, nil)

	t.Run("expect error", func(t *testing.T) {
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(nil, false, errBroken)
		_, err = s.GetConfig(context.Background(), empty)
		assert.ErrorContains(t, err, "retrieving config")
	})
}

func TestGetClustersSuccess(t *testing.T) {
	var err error
	var resp *v1.DelegatedRegistryClustersResponse

	clustersDS := clusterDSMocks.NewMockDataStore(gomock.NewController(t))
	deleClusterDS := deleDSMocks.NewMockDataStore(gomock.NewController(t))
	connMgr := connMocks.NewMockManager(gomock.NewController(t))

	fakeConnWithCap := connMocks.NewMockSensorConnection(gomock.NewController(t))
	fakeConnWithCap.EXPECT().HasCapability(gomock.Any()).Return(true).AnyTimes()

	fakeConnWithoutCap := connMocks.NewMockSensorConnection(gomock.NewController(t))
	fakeConnWithoutCap.EXPECT().HasCapability(gomock.Any()).Return(false).AnyTimes()

	s := New(deleClusterDS, clustersDS, connMgr)

	clusters := []*storage.Cluster{{Id: "id", Name: "fake"}}

	tt := map[string]struct {
		conn  *connMocks.MockSensorConnection
		valid bool
	}{
		"without cap": {fakeConnWithoutCap, false},
		"with cap":    {fakeConnWithCap, true},
	}

	for name, test := range tt {
		tf := func(t *testing.T) {
			clustersDS.EXPECT().GetClusters(gomock.Any()).Return(clusters, nil)
			connMgr.EXPECT().GetConnection(gomock.Any()).Return(test.conn)
			resp, err = s.GetClusters(context.Background(), empty)
			assert.NoError(t, err)
			require.Len(t, resp.Clusters, 1)
			assert.Equal(t, resp.Clusters[0].IsValid, test.valid)
		}

		t.Run(name, tf)
	}

	t.Run("multi cluster", func(t *testing.T) {
		cluster1 := &storage.Cluster{Id: "id1"}
		cluster2 := &storage.Cluster{Id: "id2"}
		clustersDS.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{cluster1, cluster2}, nil)
		connMgr.EXPECT().GetConnection("id1").Return(fakeConnWithCap)
		connMgr.EXPECT().GetConnection("id2").Return(fakeConnWithoutCap)
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

	s := New(deleClusterDS, clustersDS, nil)

	tt := map[string]struct {
		clusters       []*storage.Cluster
		err            error
		expectedErrMsg string
	}{
		"cluster ds error":           {nil, errBroken, "retrieving clusters"},
		"nil cluster ds response ":   {nil, nil, "no clusters found"},
		"empty cluster ds response ": {[]*storage.Cluster{}, nil, "no clusters found"},
	}

	for name, test := range tt {
		tf := func(t *testing.T) {
			clustersDS.EXPECT().GetClusters(gomock.Any()).Return(test.clusters, test.err)
			resp, err = s.GetClusters(context.Background(), empty)
			assert.Nil(t, resp)
			assert.ErrorContains(t, err, test.expectedErrMsg)
		}

		t.Run(name, tf)
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
		{Id: "id1"},
		{Id: "id2"},
	}

	tt := map[string]struct {
		cfg                 *v1.DelegatedRegistryConfig
		clusters            []*storage.Cluster
		expectedErrMsg      string
		upsertExpected      bool
		upsertErr           error
		getClustersExpected bool
		getClustersErr      error
	}{
		"nil config":                                            {nil, nil, "config missing", false, nil, false, nil},
		"upsert failed":                                         {genCfg(none, "", nil), nil, "upserting config", true, errBroken, true, nil},
		"get clusters error":                                    {genCfg(all, "fake", nil), nil, "broken", false, nil, true, errBroken},
		"multi cluster invalid default id":                      {genCfg(specific, "fake", nil), multiClusters, "is not valid", false, nil, true, nil},
		"multi cluster invalid registry path":                   {genCfg(specific, "fake", []string{"id1"}), multiClusters, "missing registry path", false, nil, true, nil},
		"multi cluster invalid registry id and path (id msg)":   {genCfg(specific, "fake", []string{"fake"}), multiClusters, "is not valid", false, nil, true, nil},
		"multi cluster invalid registry id and path (path msg)": {genCfg(specific, "fake", []string{"fake"}), multiClusters, "missing registry path", false, nil, true, nil},
	}

	clustersDS := clusterDSMocks.NewMockDataStore(gomock.NewController(t))
	deleClusterDS := deleDSMocks.NewMockDataStore(gomock.NewController(t))
	connMgr := connMocks.NewMockManager(gomock.NewController(t))
	connMgr.EXPECT().GetConnection(gomock.Any()).AnyTimes()
	s := New(deleClusterDS, clustersDS, connMgr)

	for name, test := range tt {
		tf := func(t *testing.T) {
			if test.upsertExpected {
				deleClusterDS.EXPECT().UpsertConfig(gomock.Any(), gomock.Any()).Return(test.upsertErr)
			}

			if test.getClustersExpected {
				clustersDS.EXPECT().GetClusters(gomock.Any()).Return(test.clusters, test.getClustersErr)
			}

			_, err = s.UpdateConfig(context.Background(), test.cfg)
			assert.ErrorContains(t, err, test.expectedErrMsg)
		}

		t.Run(name, tf)
	}
}

func TestUpdateConfigSuccess(t *testing.T) {
	var err error
	var cfg *v1.DelegatedRegistryConfig

	clustersDS := clusterDSMocks.NewMockDataStore(gomock.NewController(t))
	deleClusterDS := deleDSMocks.NewMockDataStore(gomock.NewController(t))

	fakeConnWithCap := connMocks.NewMockSensorConnection(gomock.NewController(t))
	fakeConnWithCap.EXPECT().HasCapability(gomock.Any()).Return(true).AnyTimes()

	fakeConnWithoutCap := connMocks.NewMockSensorConnection(gomock.NewController(t))
	fakeConnWithoutCap.EXPECT().HasCapability(gomock.Any()).Return(false).AnyTimes()

	connMgr := connMocks.NewMockManager(gomock.NewController(t))
	connMgr.EXPECT().GetConnection("id1").Return(fakeConnWithCap).AnyTimes()
	connMgr.EXPECT().GetConnection("id2").Return(fakeConnWithoutCap).AnyTimes()

	s := New(deleClusterDS, clustersDS, connMgr)
	clustersDS.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{{Id: "id1"}, {Id: "id2"}}, nil).AnyTimes()

	t.Run("default cluster id", func(t *testing.T) {
		deleClusterDS.EXPECT().UpsertConfig(gomock.Any(), gomock.Any())
		connMgr.EXPECT().SendMessage(gomock.Any(), gomock.Any())
		cfg = &v1.DelegatedRegistryConfig{EnabledFor: specific, DefaultClusterId: "id1"}
		_, err = s.UpdateConfig(context.Background(), cfg)
		assert.NoError(t, err)
	})

	t.Run("registries", func(t *testing.T) {
		deleClusterDS.EXPECT().UpsertConfig(gomock.Any(), gomock.Any())
		connMgr.EXPECT().SendMessage(gomock.Any(), gomock.Any())
		cfg.Registries = []*v1.DelegatedRegistryConfig_DelegatedRegistry{{ClusterId: "id1", Path: "something"}}
		_, err = s.UpdateConfig(context.Background(), cfg)
		assert.NoError(t, err)
	})

	t.Run("broadcast error allowed", func(t *testing.T) {
		// expect no error if sending to clusters fails but everything else succeeds
		deleClusterDS.EXPECT().UpsertConfig(gomock.Any(), gomock.Any())
		connMgr.EXPECT().SendMessage(gomock.Any(), gomock.Any()).Return(errBroken)
		_, err = s.UpdateConfig(context.Background(), cfg)
		assert.NoError(t, err)
	})

}
