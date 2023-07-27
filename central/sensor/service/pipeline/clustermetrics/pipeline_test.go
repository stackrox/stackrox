package clustermetrics

import (
	"context"
	"testing"

	infoMocks "github.com/stackrox/rox/central/metrics/info/mocks"
	metricsMocks "github.com/stackrox/rox/central/sensor/service/pipeline/clustermetrics/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite
	pipeline     *pipelineImpl
	metricsStore *metricsMocks.MockMetricsStore
	infoMetric   *infoMocks.MockInfo
	usageStore   *metricsMocks.MockusageStore
	mockCtrl     *gomock.Controller
}

type testUsageStore struct{}

func (tus *testUsageStore) UpdateUsage(_ string, _ *central.ClusterMetrics) error {
	return nil
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.metricsStore = metricsMocks.NewMockMetricsStore(suite.mockCtrl)
	suite.infoMetric = infoMocks.NewMockInfo(suite.mockCtrl)
	suite.usageStore = metricsMocks.NewMockusageStore(suite.mockCtrl)
	suite.pipeline = NewPipeline(suite.metricsStore, suite.infoMetric, suite.usageStore).(*pipelineImpl)
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestClusterMetricsMessageFromSensor() {
	deployment := fixtures.GetDeployment()
	clusterID := deployment.GetClusterId()
	expectedMetrics := &central.ClusterMetrics{NodeCount: 1, CpuCapacity: 10}

	suite.metricsStore.EXPECT().Set(clusterID, expectedMetrics)
	suite.infoMetric.EXPECT().SetClusterMetrics(clusterID, expectedMetrics)
	suite.usageStore.EXPECT().UpdateUsage(clusterID, expectedMetrics)

	err := suite.pipeline.Run(context.Background(), clusterID, &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_ClusterMetrics{
			ClusterMetrics: expectedMetrics,
		},
	}, nil)
	suite.NoError(err)
}

func (suite *PipelineTestSuite) TestClusterMetricsResetOnPipelineFinish() {
	deployment := fixtures.GetDeployment()
	clusterID := deployment.GetClusterId()
	expectedMetrics := &central.ClusterMetrics{}

	suite.metricsStore.EXPECT().Set(clusterID, expectedMetrics)
	suite.infoMetric.EXPECT().DeleteClusterMetrics(clusterID)

	suite.pipeline.OnFinish(clusterID)
}
