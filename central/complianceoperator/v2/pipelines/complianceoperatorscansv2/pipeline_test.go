package complianceoperatorscansv2

import (
	"context"
	"testing"

	reportMgrMocks "github.com/stackrox/rox/central/complianceoperator/v2/report/manager/mocks"
	v2ScanMocks "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore/mocks"
	"github.com/stackrox/rox/central/convert/testutils"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
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
	pipeline  *pipelineImpl
	v2ScanDS  *v2ScanMocks.MockDataStore
	reportMgr *reportMgrMocks.MockManager
	mockCtrl  *gomock.Controller
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

	s.v2ScanDS = v2ScanMocks.NewMockDataStore(s.mockCtrl)
	s.reportMgr = reportMgrMocks.NewMockManager(s.mockCtrl)
	s.pipeline = NewPipeline(s.v2ScanDS, s.reportMgr).(*pipelineImpl)
}

func (s *PipelineTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *PipelineTestSuite) TestRunCreate() {
	ctx := context.Background()

	s.v2ScanDS.EXPECT().UpsertScan(ctx, testutils.GetScanV2Storage(s.T())).Return(nil).Times(1)
	s.reportMgr.EXPECT().HandleScan(gomock.Any(), gomock.Any()).Times(1)

	se := &central.SensorEvent{}
	se.SetId(testutils.ScanUID)
	se.SetAction(central.ResourceAction_CREATE_RESOURCE)
	se.SetComplianceOperatorScanV2(proto.ValueOrDefault(testutils.GetScanV2SensorMsg(s.T())))
	msg := central.MsgFromSensor_builder{
		Event: se,
	}.Build()

	err := s.pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunDelete() {
	ctx := context.Background()

	s.reportMgr.EXPECT().HandleScanRemove(testutils.ScanUID).Return(nil).Times(1)
	s.v2ScanDS.EXPECT().DeleteScan(ctx, testutils.ScanUID).Return(nil).Times(1)

	se := &central.SensorEvent{}
	se.SetId(testutils.ScanUID)
	se.SetAction(central.ResourceAction_REMOVE_RESOURCE)
	se.SetComplianceOperatorScanV2(proto.ValueOrDefault(testutils.GetScanV2SensorMsg(s.T())))
	msg := central.MsgFromSensor_builder{
		Event: se,
	}.Build()

	err := s.pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunReconcileNoOp() {
	ctx := context.Background()

	s.v2ScanDS.EXPECT().GetScansByCluster(ctx, fixtureconsts.Cluster1).Return(nil, nil).Times(1)

	err := s.pipeline.Reconcile(ctx, fixtureconsts.Cluster1, reconciliation.NewStoreMap())
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunReconcile() {
	ctx := context.Background()

	s.v2ScanDS.EXPECT().GetScansByCluster(ctx, fixtureconsts.Cluster1).Return([]*storage.ComplianceOperatorScanV2{testutils.GetScanV2Storage(s.T())}, nil).Times(1)
	s.v2ScanDS.EXPECT().DeleteScan(ctx, testutils.ScanUID).Return(nil).Times(1)

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

	s.Require().False(s.pipeline.Match(v1Msg))
	s.Require().True(s.pipeline.Match(v2Msg))
	s.Require().False(s.pipeline.Match(otherMsg))
}
