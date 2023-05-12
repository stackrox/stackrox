package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	deleDSMocks "github.com/stackrox/rox/central/delegatedregistryconfig/datastore/mocks"
	"github.com/stackrox/rox/central/scrape"
	"github.com/stackrox/rox/central/sensor/networkentities"
	"github.com/stackrox/rox/central/sensor/networkpolicies"
	sensorConn "github.com/stackrox/rox/central/sensor/service/connection"
	connMgrMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/central/sensor/telemetry"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	fakeConnWithCap    = &fakeSensorConn{hasCap: true}
	fakeConnWithOutCap = &fakeSensorConn{hasCap: false}

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
	s := New(deleClusterDS, nil, nil)

	t.Run("empty", func(t *testing.T) {
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(nil, nil)
		cfg, err = s.GetConfig(context.Background(), empty)
		assert.NoError(t, err)
		assert.Empty(t, cfg)
	})

	t.Run("specific and default cluster", func(t *testing.T) {
		retVal := &storage.DelegatedRegistryConfig{EnabledFor: storage.DelegatedRegistryConfig_SPECIFIC, DefaultClusterId: "id1"}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(retVal, nil)
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
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(nil, errBroken)
		_, err = s.GetConfig(context.Background(), empty)
		assert.ErrorContains(t, err, "retrieving config")
	})
}

func TestGetClustersSuccess(t *testing.T) {
	var err error
	var resp *v1.DelegatedRegistryClustersResponse

	clustersDS := clusterDSMocks.NewMockDataStore(gomock.NewController(t))
	deleClusterDS := deleDSMocks.NewMockDataStore(gomock.NewController(t))
	connMgr := connMgrMocks.NewMockManager(gomock.NewController(t))

	s := New(deleClusterDS, clustersDS, connMgr)

	clusters := []*storage.Cluster{{Id: "id", Name: "fake"}}

	tt := map[string]struct {
		conn  *fakeSensorConn
		valid bool
	}{
		"without cap": {fakeConnWithOutCap, false},
		"with cap":    {fakeConnWithCap, true}, // only healthy scanners are valid
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
		connMgr.EXPECT().GetConnection("id2").Return(fakeConnWithOutCap)
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
		"nil cluster ds response ":   {nil, nil, "no valid clusters"},
		"empty cluster ds response ": {[]*storage.Cluster{}, nil, "no valid clusters"},
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
	_ = genCfg

	multiClusters := []*storage.Cluster{
		{Id: "id1"},
		{Id: "id2"},
	}
	_ = multiClusters

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
		// "enabled for all missing default id": {genCfg(all, "", nil), nil, "default cluster id required", true, nil, true, nil},
		// "enabled for specific missing default id": {genCfg(specific, "", nil), nil, "default cluster id required", true, nil, true, nil},
	}

	clustersDS := clusterDSMocks.NewMockDataStore(gomock.NewController(t))
	deleClusterDS := deleDSMocks.NewMockDataStore(gomock.NewController(t))
	connMgr := connMgrMocks.NewMockManager(gomock.NewController(t))
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

			_, err = s.PutConfig(context.Background(), test.cfg)
			assert.ErrorContains(t, err, test.expectedErrMsg)
		}

		t.Run(name, tf)
	}
}

func TestPutConfigSuccess(t *testing.T) {
	var err error
	var cfg *v1.DelegatedRegistryConfig

	clustersDS := clusterDSMocks.NewMockDataStore(gomock.NewController(t))
	deleClusterDS := deleDSMocks.NewMockDataStore(gomock.NewController(t))
	connMgr := connMgrMocks.NewMockManager(gomock.NewController(t))
	connMgr.EXPECT().GetConnection("id1").Return(fakeConnWithCap).AnyTimes()
	connMgr.EXPECT().GetConnection("id2").Return(fakeConnWithOutCap).AnyTimes()

	s := New(deleClusterDS, clustersDS, connMgr)
	clustersDS.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{{Id: "id1"}, {Id: "id2"}}, nil).AnyTimes()

	t.Run("default cluster id", func(t *testing.T) {
		deleClusterDS.EXPECT().UpsertConfig(gomock.Any(), gomock.Any())
		connMgr.EXPECT().SendMessage(gomock.Any(), gomock.Any())
		cfg = &v1.DelegatedRegistryConfig{EnabledFor: specific, DefaultClusterId: "id1"}
		_, err = s.PutConfig(context.Background(), cfg)
		assert.NoError(t, err)
	})

	t.Run("registries", func(t *testing.T) {
		deleClusterDS.EXPECT().UpsertConfig(gomock.Any(), gomock.Any())
		connMgr.EXPECT().SendMessage(gomock.Any(), gomock.Any())
		cfg.Registries = []*v1.DelegatedRegistryConfig_DelegatedRegistry{{ClusterId: "id1", RegistryPath: "something"}}
		_, err = s.PutConfig(context.Background(), cfg)
		assert.NoError(t, err)
	})

	t.Run("broadcast error allowed", func(t *testing.T) {
		// expect no error if sending to clusters fails but everything else succeeds
		deleClusterDS.EXPECT().UpsertConfig(gomock.Any(), gomock.Any())
		connMgr.EXPECT().SendMessage(gomock.Any(), gomock.Any()).Return(errBroken)
		_, err = s.PutConfig(context.Background(), cfg)
		assert.NoError(t, err)
	})

}

type fakeSensorConn struct {
	hasCap bool
}

var _ sensorConn.SensorConnection = (*fakeSensorConn)(nil)

func (f *fakeSensorConn) HasCapability(capability centralsensor.SensorCapability) bool {
	return f.hasCap
}
func (*fakeSensorConn) InjectMessageIntoQueue(msg *central.MsgFromSensor)      {}
func (*fakeSensorConn) CheckAutoUpgradeSupport() error                         { return nil }
func (*fakeSensorConn) ClusterID() string                                      { return "" }
func (*fakeSensorConn) NetworkEntities() networkentities.Controller            { return nil }
func (*fakeSensorConn) NetworkPolicies() networkpolicies.Controller            { return nil }
func (*fakeSensorConn) ObjectsDeletedByReconciliation() (map[string]int, bool) { return nil, false }
func (*fakeSensorConn) Scrapes() scrape.Controller                             { return nil }
func (*fakeSensorConn) Stopped() concurrency.ReadOnlyErrorSignal               { return nil }
func (*fakeSensorConn) Telemetry() telemetry.Controller                        { return nil }
func (*fakeSensorConn) Terminate(err error) bool                               { return false }
func (*fakeSensorConn) InjectMessage(ctx concurrency.Waitable, msg *central.MsgToSensor) error {
	return nil
}
