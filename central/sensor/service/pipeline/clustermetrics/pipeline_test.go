package clustermetrics

import (
	"context"
	"testing"

	usageMocks "github.com/stackrox/rox/central/administration/usage/datastore/securedunits/mocks"
	telemetryMocks "github.com/stackrox/rox/central/metrics/telemetry/mocks"
	metricsMocks "github.com/stackrox/rox/central/sensor/service/pipeline/clustermetrics/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite
	pipeline         *pipelineImpl
	metricsStore     *metricsMocks.MockMetricsStore
	telemetryMetrics *telemetryMocks.MockTelemetry
	usageStore       *usageMocks.MockDataStore
	mockCtrl         *gomock.Controller
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.metricsStore = metricsMocks.NewMockMetricsStore(suite.mockCtrl)
	suite.telemetryMetrics = telemetryMocks.NewMockTelemetry(suite.mockCtrl)
	suite.usageStore = usageMocks.NewMockDataStore(suite.mockCtrl)
	suite.pipeline = NewPipeline(suite.metricsStore, suite.telemetryMetrics, suite.usageStore).(*pipelineImpl)
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestClusterMetricsMessageFromSensor() {
	deployment := fixtures.GetDeployment()
	clusterID := deployment.GetClusterId()
	expectedMetrics := &central.ClusterMetrics{}
	expectedMetrics.SetNodeCount(1)
	expectedMetrics.SetCpuCapacity(10)

	suite.metricsStore.EXPECT().Set(clusterID, expectedMetrics)
	suite.telemetryMetrics.EXPECT().SetClusterMetrics(clusterID, expectedMetrics)
	su := &storage.SecuredUnits{}
	su.SetNumNodes(expectedMetrics.GetNodeCount())
	su.SetNumCpuUnits(expectedMetrics.GetCpuCapacity())
	suite.usageStore.EXPECT().UpdateUsage(gomock.Any(), clusterID, su).Return(nil)

	mfs := &central.MsgFromSensor{}
	mfs.SetClusterMetrics(proto.ValueOrDefault(expectedMetrics))
	err := suite.pipeline.Run(context.Background(), clusterID, mfs, nil)
	suite.NoError(err)
}

func (suite *PipelineTestSuite) TestClusterMetricsResetOnPipelineFinish() {
	deployment := fixtures.GetDeployment()
	clusterID := deployment.GetClusterId()
	expectedMetrics := &central.ClusterMetrics{}

	suite.metricsStore.EXPECT().Set(clusterID, expectedMetrics)
	suite.telemetryMetrics.EXPECT().DeleteClusterMetrics(clusterID)

	suite.pipeline.OnFinish(clusterID)
}
