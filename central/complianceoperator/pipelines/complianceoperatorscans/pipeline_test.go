package complianceoperatorscans

import (
	"context"
	"testing"

	managerMocks "github.com/stackrox/rox/central/complianceoperator/manager/mocks"
	v1ScanMocks "github.com/stackrox/rox/central/complianceoperator/scans/datastore/mocks"
	"github.com/stackrox/rox/central/convert/testutils"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite
	pipeline *pipelineImpl
	manager  *managerMocks.MockManager
	v1ScanDS *v1ScanMocks.MockDataStore
	mockCtrl *gomock.Controller
}

func (s *PipelineTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip tests when ComplianceEnhancements disabled")
		s.T().SkipNow()
	}
}

func (s *PipelineTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.manager = managerMocks.NewMockManager(s.mockCtrl)
	s.v1ScanDS = v1ScanMocks.NewMockDataStore(s.mockCtrl)
	s.pipeline = NewPipeline(s.v1ScanDS, s.manager).(*pipelineImpl)
}

func (s *PipelineTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *PipelineTestSuite) TestRunV1Create() {
	ctx := context.Background()

	s.manager.EXPECT().AddScan(testutils.GetScanV1Storage(s.T())).Return(nil).Times(1)

	se := &central.SensorEvent{}
	se.SetId(testutils.ScanUID)
	se.SetAction(central.ResourceAction_CREATE_RESOURCE)
	se.SetComplianceOperatorScan(proto.ValueOrDefault(testutils.GetScanV1Storage(s.T())))
	msg := central.MsgFromSensor_builder{
		Event: se,
	}.Build()

	err := s.pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunV1Delete() {
	ctx := context.Background()

	s.manager.EXPECT().DeleteScan(testutils.GetScanV1Storage(s.T())).Return(nil).Times(1)

	se := &central.SensorEvent{}
	se.SetId(testutils.ScanUID)
	se.SetAction(central.ResourceAction_REMOVE_RESOURCE)
	se.SetComplianceOperatorScan(proto.ValueOrDefault(testutils.GetScanV1Storage(s.T())))
	msg := central.MsgFromSensor_builder{
		Event: se,
	}.Build()

	err := s.pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunReconcileNoOp() {
	ctx := context.Background()

	s.v1ScanDS.EXPECT().Walk(ctx, gomock.Any()).Return(nil).Times(1)

	err := s.pipeline.Reconcile(ctx, fixtureconsts.Cluster1, reconciliation.NewStoreMap())
	s.NoError(err)
}

func (s *PipelineTestSuite) TestCapabilities() {
	s.Require().Nil(s.pipeline.Capabilities())
}

func (s *PipelineTestSuite) TestMatch() {
	se := &central.SensorEvent{}
	se.SetId(testutils.ScanUID)
	se.SetAction(central.ResourceAction_REMOVE_RESOURCE)
	se.SetComplianceOperatorScan(proto.ValueOrDefault(testutils.GetScanV1Storage(s.T())))
	v1Msg := central.MsgFromSensor_builder{
		Event: se,
	}.Build()

	se2 := &central.SensorEvent{}
	se2.SetId(testutils.ScanUID)
	se2.SetAction(central.ResourceAction_REMOVE_RESOURCE)
	se2.SetComplianceOperatorScanV2(proto.ValueOrDefault(testutils.GetScanV2SensorMsg(s.T())))
	v2Msg := central.MsgFromSensor_builder{
		Event: se2,
	}.Build()

	se3 := &central.SensorEvent{}
	se3.SetId(testutils.ProfileUID)
	se3.SetAction(central.ResourceAction_REMOVE_RESOURCE)
	se3.SetComplianceOperatorProfileV2(proto.ValueOrDefault(testutils.GetProfileV2SensorMsg(s.T())))
	otherMsg := central.MsgFromSensor_builder{
		Event: se3,
	}.Build()

	s.Require().True(s.pipeline.Match(v1Msg))
	s.Require().False(s.pipeline.Match(v2Msg))
	s.Require().False(s.pipeline.Match(otherMsg))
}
