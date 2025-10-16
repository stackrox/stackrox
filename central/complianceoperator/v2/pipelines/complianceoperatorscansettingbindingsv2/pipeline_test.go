package complianceoperatorscansettingbindingsv2

import (
	"context"
	"testing"

	v2Mocks "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore/mocks"
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
	pipeline *pipelineImpl
	v2DS     *v2Mocks.MockDataStore
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

	s.v2DS = v2Mocks.NewMockDataStore(s.mockCtrl)
	s.pipeline = NewPipeline(s.v2DS).(*pipelineImpl)
}

func (s *PipelineTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *PipelineTestSuite) TestRunCreate() {
	ctx := context.Background()

	s.v2DS.EXPECT().UpsertScanSettingBinding(ctx, testutils.GetScanSettingBindingV2Storage(s.T(), fixtureconsts.Cluster1)).Return(nil).Times(1)

	se := &central.SensorEvent{}
	se.SetId(testutils.ScanSettingUID)
	se.SetAction(central.ResourceAction_CREATE_RESOURCE)
	se.SetComplianceOperatorScanSettingBindingV2(proto.ValueOrDefault(testutils.GetScanSettingBindingV2SensorMsg(s.T())))
	msg := central.MsgFromSensor_builder{
		Event: se,
	}.Build()

	err := s.pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunDelete() {
	ctx := context.Background()

	s.v2DS.EXPECT().DeleteScanSettingBinding(ctx, testutils.ScanSettingUID).Return(nil).Times(1)

	se := &central.SensorEvent{}
	se.SetId(testutils.ScanSettingUID)
	se.SetAction(central.ResourceAction_REMOVE_RESOURCE)
	se.SetComplianceOperatorScanSettingBindingV2(proto.ValueOrDefault(testutils.GetScanSettingBindingV2SensorMsg(s.T())))
	msg := central.MsgFromSensor_builder{
		Event: se,
	}.Build()

	err := s.pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunReconcileNoOp() {
	ctx := context.Background()

	s.v2DS.EXPECT().GetScanSettingBindingsByCluster(ctx, fixtureconsts.Cluster1).Return(nil, nil).Times(1)

	err := s.pipeline.Reconcile(ctx, fixtureconsts.Cluster1, reconciliation.NewStoreMap())
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunReconcile() {
	ctx := context.Background()

	s.v2DS.EXPECT().GetScanSettingBindingsByCluster(ctx, fixtureconsts.Cluster1).Return([]*storage.ComplianceOperatorScanSettingBindingV2{
		testutils.GetScanSettingBindingV2Storage(s.T(), fixtureconsts.Cluster1)}, nil).Times(1)
	s.v2DS.EXPECT().DeleteScanSettingBinding(ctx, testutils.ScanSettingUID).Return(nil).Times(1)

	err := s.pipeline.Reconcile(ctx, fixtureconsts.Cluster1, reconciliation.NewStoreMap())
	s.NoError(err)
}

func (s *PipelineTestSuite) TestCapabilities() {
	s.Require().Nil(s.pipeline.Capabilities())
}

func (s *PipelineTestSuite) TestMatch() {
	se := &central.SensorEvent{}
	se.SetId(testutils.ScanSettingUID)
	se.SetAction(central.ResourceAction_REMOVE_RESOURCE)
	se.SetComplianceOperatorScanSettingBinding(proto.ValueOrDefault(testutils.GetScanSettingBindingV1Storage(s.T(), fixtureconsts.Cluster1)))
	v1Msg := central.MsgFromSensor_builder{
		Event: se,
	}.Build()

	se2 := &central.SensorEvent{}
	se2.SetId(testutils.ScanSettingUID)
	se2.SetAction(central.ResourceAction_REMOVE_RESOURCE)
	se2.SetComplianceOperatorScanSettingBindingV2(proto.ValueOrDefault(testutils.GetScanSettingBindingV2SensorMsg(s.T())))
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
