package complianceoperatorrulesv2

import (
	"context"
	"testing"

	v2Mocks "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore/mocks"
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

	s.v2DS.EXPECT().UpsertRule(ctx, testutils.GetRuleV2Storage(s.T())).Return(nil).Times(1)

	se := &central.SensorEvent{}
	se.SetId(testutils.RuleUID)
	se.SetAction(central.ResourceAction_CREATE_RESOURCE)
	se.SetComplianceOperatorRuleV2(proto.ValueOrDefault(testutils.GetRuleV2SensorMsg(s.T())))
	msg := central.MsgFromSensor_builder{
		Event: se,
	}.Build()

	err := s.pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunDelete() {
	ctx := context.Background()

	s.v2DS.EXPECT().DeleteRule(ctx, testutils.RuleUID).Return(nil).Times(1)

	se := &central.SensorEvent{}
	se.SetId(testutils.RuleUID)
	se.SetAction(central.ResourceAction_REMOVE_RESOURCE)
	se.SetComplianceOperatorRuleV2(proto.ValueOrDefault(testutils.GetRuleV2SensorMsg(s.T())))
	msg := central.MsgFromSensor_builder{
		Event: se,
	}.Build()

	err := s.pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunReconcileNoOp() {
	ctx := context.Background()

	s.v2DS.EXPECT().GetRulesByCluster(ctx, fixtureconsts.Cluster1).Return(nil, nil).Times(1)

	err := s.pipeline.Reconcile(ctx, fixtureconsts.Cluster1, reconciliation.NewStoreMap())
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunReconcile() {
	ctx := context.Background()

	s.v2DS.EXPECT().GetRulesByCluster(ctx, fixtureconsts.Cluster1).Return([]*storage.ComplianceOperatorRuleV2{testutils.GetRuleV2Storage(s.T())}, nil).Times(1)
	s.v2DS.EXPECT().DeleteRule(ctx, testutils.RuleUID).Return(nil).Times(1)

	err := s.pipeline.Reconcile(ctx, fixtureconsts.Cluster1, reconciliation.NewStoreMap())
	s.NoError(err)
}

func (s *PipelineTestSuite) TestCapabilities() {
	s.Require().Nil(s.pipeline.Capabilities())
}

func (s *PipelineTestSuite) TestMatch() {
	se := &central.SensorEvent{}
	se.SetId(testutils.RuleUID)
	se.SetAction(central.ResourceAction_REMOVE_RESOURCE)
	se.SetComplianceOperatorRule(proto.ValueOrDefault(testutils.GetRuleV1Storage(s.T())))
	v1Msg := central.MsgFromSensor_builder{
		Event: se,
	}.Build()

	se2 := &central.SensorEvent{}
	se2.SetId(testutils.RuleUID)
	se2.SetAction(central.ResourceAction_REMOVE_RESOURCE)
	se2.SetComplianceOperatorRuleV2(proto.ValueOrDefault(testutils.GetRuleV2SensorMsg(s.T())))
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
