package clustermetrics

import (
	"context"
	"testing"

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
	mockCtrl     *gomock.Controller
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.metricsStore = metricsMocks.NewMockMetricsStore(suite.mockCtrl)
	suite.pipeline = NewPipeline(suite.metricsStore).(*pipelineImpl)
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestClusterMetricsMessageFromSensor() {
	deployment := fixtures.GetDeployment()
	clusterID := deployment.GetClusterId()
	expectedMetrics := &central.ClusterMetrics{NodeCount: 1, CpuCapacity: 10}

	suite.metricsStore.EXPECT().Set(clusterID, expectedMetrics)

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

	suite.pipeline.OnFinish(clusterID)
}
