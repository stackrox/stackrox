package service

import (
	"context"
	"strings"
	"testing"
	"time"

	datastoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	configDatastoreMocks "github.com/stackrox/rox/central/config/datastore/mocks"
	probeSourcesMocks "github.com/stackrox/rox/central/probesources/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestClusterService(t *testing.T) {
	suite.Run(t, new(ClusterServiceTestSuite))
}

type ClusterServiceTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	dataStore          *datastoreMocks.MockDataStore
	sysConfigDatastore *configDatastoreMocks.MockDataStore
}

var _ suite.TearDownTestSuite = (*ClusterServiceTestSuite)(nil)

func (suite *ClusterServiceTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.dataStore = datastoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.sysConfigDatastore = configDatastoreMocks.NewMockDataStore(suite.mockCtrl)
	testutils.SetExampleVersion(suite.T())
}

func (suite *ClusterServiceTestSuite) TearDownTest() {
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
	suite.T().Setenv(defaults.ImageFlavorEnvName, defaults.ImageFlavorNameRHACSRelease)
	flavor := defaults.GetImageFlavorFromEnv()
	for name, testCase := range cases {
		suite.Run(name, func() {
			ps := probeSourcesMocks.NewMockProbeSources(suite.mockCtrl)
			ps.EXPECT().AnyAvailable(gomock.Any()).Times(1).Return(testCase.kernelSupportAvailable, nil)
			clusterService := New(suite.dataStore, nil, ps, suite.sysConfigDatastore)

			defaults, err := clusterService.GetClusterDefaultValues(context.Background(), nil)
			suite.NoError(err)
			suite.Equal(flavor.MainImageNoTag(), defaults.GetMainImageRepository())
			suite.Equal(flavor.CollectorImageNoTag(), defaults.GetCollectorImageRepository())
			suite.Equal(testCase.kernelSupportAvailable, defaults.GetKernelSupportAvailable())
		})
	}
}

func (suite *ClusterServiceTestSuite) TestGetClusterWithRetentionInfo() {
	tenDaysAgo, err := protocompat.ConvertTimeToTimestampOrError(daysAgo(10))
	suite.NoError(err)
	eightyDaysAgo, err := protocompat.ConvertTimeToTimestampOrError(daysAgo(80))
	suite.NoError(err)

	cases := map[string]struct {
		cluster  *storage.Cluster
		config   *storage.Config
		expected string
	}{
		"HEALTHY cluster": {
			cluster: &storage.Cluster{
				Id: "HEALTHY cluster",
				HealthStatus: &storage.ClusterHealthStatus{
					SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
				},
			},
			config:   suite.getTestSystemConfig(60, 30, 7),
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
			config:   suite.getTestSystemConfig(60, 30, 7),
			expected: "is_excluded:true",
		},
		"UNHEALTHY cluster with last contact time after config creation time": {
			cluster: &storage.Cluster{
				Id:     "UNHEALTHY cluster with last contact time after config creation time",
				Labels: map[string]string{"k1": "v2"},
				HealthStatus: &storage.ClusterHealthStatus{
					SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
					LastContact:        tenDaysAgo,
				},
			},
			config:   suite.getTestSystemConfig(60, 30, 7),
			expected: "days_until_deletion:50",
		},
		"UNHEALTHY cluster with last contact time before config creation time": {
			cluster: &storage.Cluster{
				Id:     "UNHEALTHY cluster with last contact time before config creation time",
				Labels: map[string]string{"k1": "v2"},
				HealthStatus: &storage.ClusterHealthStatus{
					SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
					LastContact:        eightyDaysAgo,
				},
			},
			config:   suite.getTestSystemConfig(60, 30, 7),
			expected: "days_until_deletion:30",
		},
		"UNHEALTHY cluster, cluster removal disabled": {
			cluster: &storage.Cluster{
				Id: "UNHEALTHY CLUSTER",
				HealthStatus: &storage.ClusterHealthStatus{
					SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
					LastContact:        tenDaysAgo,
				},
			},
			config:   suite.getTestSystemConfig(0, 30, 7),
			expected: "<nil>",
		},
	}

	for name, testCase := range cases {
		suite.Run(name, func() {
			ps := probeSourcesMocks.NewMockProbeSources(suite.mockCtrl)
			suite.dataStore.EXPECT().GetCluster(gomock.Any(), gomock.Any()).Times(1).Return(testCase.cluster, true, nil)
			if testCase.cluster.GetHealthStatus().GetSensorHealthStatus() == storage.ClusterHealthStatus_UNHEALTHY {
				suite.sysConfigDatastore.EXPECT().GetPrivateConfig(gomock.Any()).Times(1).Return(testCase.config.GetPrivateConfig(), nil)
			}
			clusterService := New(suite.dataStore, nil, ps, suite.sysConfigDatastore)

			clusterID := &v1.ResourceByID{
				Id: testCase.cluster.GetId(),
			}
			result, err := clusterService.GetCluster(context.Background(), clusterID)
			suite.NoError(err)
			suite.Equal(testCase.expected, strings.TrimSpace(result.GetClusterRetentionInfo().String()))
		})
	}
}

func (suite *ClusterServiceTestSuite) TestGetClustersWithRetentionInfoMap() {
	config := suite.getTestSystemConfig(60, 30, 7)

	tenDaysAgo, err := protocompat.ConvertTimeToTimestampOrError(daysAgo(10))
	suite.NoError(err)
	eightyDaysAgo, err := protocompat.ConvertTimeToTimestampOrError(daysAgo(80))
	suite.NoError(err)
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
				LastContact:        tenDaysAgo,
			},
		},
		{
			Id:     "UNHEALTHY cluster with last contact time before config creation time",
			Labels: map[string]string{"k1": "v2"},
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
				LastContact:        eightyDaysAgo,
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
	suite.sysConfigDatastore.EXPECT().GetPrivateConfig(gomock.Any()).Times(3).Return(config.GetPrivateConfig(), nil)

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

func daysAgo(days int) time.Time {
	return time.Now().Add(-time.Duration(days) * 24 * time.Hour)
}

func (suite *ClusterServiceTestSuite) getTestSystemConfig(retentionDays, createdBeforeDays, lastUpdatedBeforeDays int) *storage.Config {
	lastUpdated, err := protocompat.ConvertTimeToTimestampOrError(daysAgo(lastUpdatedBeforeDays))
	suite.NoError(err)
	createdAt, err := protocompat.ConvertTimeToTimestampOrError(daysAgo(createdBeforeDays))
	suite.NoError(err)
	return &storage.Config{
		PrivateConfig: &storage.PrivateConfig{
			DecommissionedClusterRetention: &storage.DecommissionedClusterRetentionConfig{
				RetentionDurationDays: int32(retentionDays),
				IgnoreClusterLabels: map[string]string{
					"k1": "v1",
					"k2": "v2",
					"k3": "v3",
				},
				LastUpdated: lastUpdated,
				CreatedAt:   createdAt,
			},
		},
	}
}
