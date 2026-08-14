package clustermetrics

import (
	"context"
	"testing"

	usageMocks "github.com/stackrox/rox/central/administration/usage/datastore/securedunits/mocks"
	telemetryMocks "github.com/stackrox/rox/central/metrics/telemetry/mocks"
	"github.com/stackrox/rox/central/sensor/service/common"
	metricsMocks "github.com/stackrox/rox/central/sensor/service/pipeline/clustermetrics/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
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
	expectedMetrics := &central.ClusterMetrics{NodeCount: 1, CpuCapacity: 10}

	suite.metricsStore.EXPECT().Set(clusterID, expectedMetrics)
	suite.telemetryMetrics.EXPECT().SetClusterMetrics(clusterID, expectedMetrics)
	suite.usageStore.EXPECT().UpdateUsage(gomock.Any(), clusterID, &storage.SecuredUnits{
		NumNodes:    expectedMetrics.GetNodeCount(),
		NumCpuUnits: expectedMetrics.GetCpuCapacity(),
	}).Return(nil)

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
	suite.telemetryMetrics.EXPECT().DeleteClusterMetrics(clusterID)

	suite.pipeline.OnFinish(clusterID)
}

func TestHasVMTelemetryCap(t *testing.T) {
	tests := map[string]struct {
		injector common.MessageInjector
		want     bool
	}{
		"should return false when injector is nil": {
			injector: nil,
			want:     false,
		},
		"should return false when Sensor lacks the capability": {
			injector: &fakeInjector{},
			want:     false,
		},
		"should return true when Sensor advertises the capability": {
			injector: &fakeInjector{caps: map[centralsensor.SensorCapability]bool{
				centralsensor.VirtualMachineTelemetryCap: true,
			}},
			want: true,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, hasVMTelemetryCap(tc.injector))
		})
	}
}

func (suite *PipelineTestSuite) TestClusterMetricsMessageFromSensor_QueriesVMTelemetryCap() {
	tests := map[string]struct {
		caps map[centralsensor.SensorCapability]bool
	}{
		"should query VirtualMachineTelemetryCap when Sensor advertises it": {
			caps: map[centralsensor.SensorCapability]bool{
				centralsensor.VirtualMachineTelemetryCap: true,
			},
		},
		"should query VirtualMachineTelemetryCap when Sensor lacks it": {
			caps: map[centralsensor.SensorCapability]bool{},
		},
	}
	for name, tc := range tests {
		suite.Run(name, func() {
			clusterID := fixtures.GetDeployment().GetClusterId()
			expectedMetrics := &central.ClusterMetrics{NodeCount: 1, CpuCapacity: 10}

			suite.metricsStore.EXPECT().Set(clusterID, expectedMetrics)
			suite.telemetryMetrics.EXPECT().SetClusterMetrics(clusterID, expectedMetrics)
			suite.usageStore.EXPECT().UpdateUsage(gomock.Any(), clusterID, &storage.SecuredUnits{
				NumNodes:    expectedMetrics.GetNodeCount(),
				NumCpuUnits: expectedMetrics.GetCpuCapacity(),
			}).Return(nil)

			injector := &fakeInjector{caps: tc.caps}
			err := suite.pipeline.Run(suite.T().Context(), clusterID, &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_ClusterMetrics{
					ClusterMetrics: expectedMetrics,
				},
			}, injector)
			suite.NoError(err)
			suite.Equal([]centralsensor.SensorCapability{centralsensor.VirtualMachineTelemetryCap}, injector.queried)
		})
	}
}

var _ common.MessageInjector = (*fakeInjector)(nil)

type fakeInjector struct {
	caps    map[centralsensor.SensorCapability]bool
	queried []centralsensor.SensorCapability
}

func (f *fakeInjector) HasCapability(cap centralsensor.SensorCapability) bool {
	f.queried = append(f.queried, cap)
	return f.caps[cap]
}

func (f *fakeInjector) InjectMessage(concurrency.Waitable, *central.MsgToSensor) error {
	return nil
}

func (f *fakeInjector) InjectMessageIntoQueue(*central.MsgFromSensor) {}
