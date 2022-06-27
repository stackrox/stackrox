package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	datastoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	configDatastoreMocks "github.com/stackrox/rox/central/config/datastore/mocks"
	probeSourcesMocks "github.com/stackrox/rox/central/probesources/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/suite"
)

func TestClusterService(t *testing.T) {
	suite.Run(t, new(ClusterServiceTestSuite))
}

type ClusterServiceTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller
	ei       *envisolator.EnvIsolator

	dataStore          *datastoreMocks.MockDataStore
	sysConfigDatastore *configDatastoreMocks.MockDataStore
}

var _ suite.TearDownTestSuite = (*ClusterServiceTestSuite)(nil)

func (suite *ClusterServiceTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.dataStore = datastoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.sysConfigDatastore = configDatastoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.ei = envisolator.NewEnvIsolator(suite.T())

	suite.ei.Setenv("ROX_IMAGE_FLAVOR", "rhacs")
	testbuildinfo.SetForTest(suite.T())
	testutils.SetExampleVersion(suite.T())
}

func (suite *ClusterServiceTestSuite) TearDownTest() {
	suite.ei.RestoreAll()
	suite.mockCtrl.Finish()
}

func (suite *ClusterServiceTestSuite) TestGetClusterDefaults() {

	cases := map[string]struct {
		kernelSupportAvailable bool
	}{
		"No kernel suppport": {
			kernelSupportAvailable: false,
		},
		"With kernel suppport": {
			kernelSupportAvailable: true,
		},
	}
	flavor := defaults.GetImageFlavorFromEnv()
	for name, testCase := range cases {
		suite.Run(name, func() {
			ps := probeSourcesMocks.NewMockProbeSources(suite.mockCtrl)
			ps.EXPECT().AnyAvailable(gomock.Any()).Times(1).Return(testCase.kernelSupportAvailable, nil)
			clusterService := New(suite.dataStore, nil, ps, suite.sysConfigDatastore)

			defaults, err := clusterService.GetClusterDefaultValues(context.Background(), nil)
			suite.NoError(err)
			suite.Equal(flavor.MainImageNoTag(), defaults.GetMainImageRepository())
			suite.Equal(flavor.CollectorFullImageNoTag(), defaults.GetCollectorImageRepository())
			suite.Equal(testCase.kernelSupportAvailable, defaults.GetKernelSupportAvailable())
		})
	}
}

func (suite *ClusterServiceTestSuite) TestGetClusterWithRetentionInfo() {
	if !features.DecommissionedClusterRetention.Enabled() {
		suite.T().Skip("Skipping GetCluster with RetentionInfo tests because decommissioned cluster retention feature is turned off")
	}
	config := suite.getTestSystemConfig()

	cases := map[string]struct {
		cluster  *storage.Cluster
		expected string
	}{
		"HEALTHY cluster": {
			cluster: &storage.Cluster{
				Id: "HEALTHY cluster",
				HealthStatus: &storage.ClusterHealthStatus{
					SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
				},
			},
			expected: "<nil>",
		},
		"UNHEALTHY cluster with label matching ignored labels": {
			cluster: &storage.Cluster{
				Id:     "UNHEALTHY cluster matching a label to ignore the cluster",
				Labels: map[string]string{"k2": "v2"},
				HealthStatus: &storage.ClusterHealthStatus{
					SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
				},
			},
			expected: "is_excluded:true",
		},
		"UNHEALTHY cluster with last contact time after config creation time": {
			cluster: &storage.Cluster{
				Id:     "UNHEALTHY cluster with last contact time after config creation time",
				Labels: map[string]string{"k1": "v2"},
				HealthStatus: &storage.ClusterHealthStatus{
					SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
					LastContact:        suite.timeBeforeDays(10),
				},
			},
			expected: "days_until_deletion:50",
		},
		"UNHEALTHY cluster with last contact time before config creation time": {
			cluster: &storage.Cluster{
				Id:     "UNHEALTHY cluster with last contact time before config creation time",
				Labels: map[string]string{"k1": "v2"},
				HealthStatus: &storage.ClusterHealthStatus{
					SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
					LastContact:        suite.timeBeforeDays(80),
				},
			},
			expected: "days_until_deletion:30",
		},
	}

	for name, testCase := range cases {
		suite.Run(name, func() {
			ps := probeSourcesMocks.NewMockProbeSources(suite.mockCtrl)
			suite.dataStore.EXPECT().GetCluster(gomock.Any(), gomock.Any()).Times(1).Return(testCase.cluster, true, nil)
			suite.sysConfigDatastore.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(config, nil)
			clusterService := New(suite.dataStore, nil, ps, suite.sysConfigDatastore)

			clusterID := &v1.ResourceByID{
				Id: testCase.cluster.GetId(),
			}
			result, err := clusterService.GetCluster(context.Background(), clusterID)
			suite.NoError(err)
			suite.Equal(strings.TrimSpace(result.GetClusterRetentionInfo().String()), testCase.expected)
		})
	}
}

func (suite *ClusterServiceTestSuite) TestGetClustersWithRetentionInfoMap() {
	if !features.DecommissionedClusterRetention.Enabled() {
		suite.T().Skip("Skipping GetClusters with RetentionInfo map tests because decommissioned cluster retention feature is turned off")
	}

	config := suite.getTestSystemConfig()

	clusters := []*storage.Cluster{
		{
			Id: "HEALTHY cluster",
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
			},
		},
		{
			Id:     "UNHEALTHY cluster matching a label to ignore the cluster",
			Labels: map[string]string{"k2": "v2"},
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
			},
		},
		{
			Id:     "UNHEALTHY cluster with last contact time after config creation time",
			Labels: map[string]string{"k1": "v2"},
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
				LastContact:        suite.timeBeforeDays(10),
			},
		},
		{
			Id:     "UNHEALTHY cluster with last contact time before config creation time",
			Labels: map[string]string{"k1": "v2"},
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
				LastContact:        suite.timeBeforeDays(80),
			},
		},
	}

	expectedIds := []string{
		"UNHEALTHY cluster matching a label to ignore the cluster",
		"UNHEALTHY cluster with last contact time after config creation time",
		"UNHEALTHY cluster with last contact time before config creation time",
	}

	ps := probeSourcesMocks.NewMockProbeSources(suite.mockCtrl)
	suite.dataStore.EXPECT().SearchRawClusters(gomock.Any(), gomock.Any()).Times(1).Return(clusters, nil)
	suite.sysConfigDatastore.EXPECT().GetConfig(gomock.Any()).Times(3).Return(config, nil)

	clusterService := New(suite.dataStore, nil, ps, suite.sysConfigDatastore)
	results, err := clusterService.GetClusters(context.Background(), &v1.GetClustersRequest{Query: search.EmptyQuery().String()})
	suite.NoError(err)

	idToRetentionInfoMap := results.GetClusterIdToRetentionInfo()
	suite.Equal(3, len(idToRetentionInfoMap))

	for _, k := range expectedIds {
		_, exists := idToRetentionInfoMap[k]
		suite.True(exists)
	}
}

func (suite *ClusterServiceTestSuite) timeBeforeDays(days int) *types.Timestamp {
	result, err := types.TimestampProto(time.Now().Add(-24 * time.Duration(days) * time.Hour))
	suite.NoError(err)
	return result
}

func (suite *ClusterServiceTestSuite) getTestSystemConfig() *storage.Config {
	return &storage.Config{
		PrivateConfig: &storage.PrivateConfig{
			DecommissionedClusterRetention: &storage.DecommissionedClusterRetentionConfig{
				RetentionDurationDays: 60,
				IgnoreClusterLabels: map[string]string{
					"k1": "v1",
					"k2": "v2",
					"k3": "v3",
				},
				LastUpdated: suite.timeBeforeDays(7),
				CreatedAt:   suite.timeBeforeDays(30),
			},
		},
	}
}
