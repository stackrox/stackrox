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

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     testutils.RuleUID,
				Action: central.ResourceAction_CREATE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorRuleV2{
					ComplianceOperatorRuleV2: testutils.GetRuleV2SensorMsg(s.T()),
				},
			},
		},
	}

	err := s.pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunDelete() {
	ctx := context.Background()

	s.v2DS.EXPECT().DeleteRule(ctx, testutils.RuleUID).Return(nil).Times(1)

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     testutils.RuleUID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorRuleV2{
					ComplianceOperatorRuleV2: testutils.GetRuleV2SensorMsg(s.T()),
				},
			},
		},
	}

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
	v1Msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     testutils.RuleUID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorRule{
					ComplianceOperatorRule: testutils.GetRuleV1Storage(s.T()),
				},
			},
		},
	}

	v2Msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     testutils.RuleUID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorRuleV2{
					ComplianceOperatorRuleV2: testutils.GetRuleV2SensorMsg(s.T()),
				},
			},
		},
	}

	otherMsg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     testutils.ProfileUID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorProfileV2{
					ComplianceOperatorProfileV2: testutils.GetProfileV2SensorMsg(s.T()),
				},
			},
		},
	}

	s.Require().False(s.pipeline.Match(v1Msg))
	s.Require().True(s.pipeline.Match(v2Msg))
	s.Require().False(s.pipeline.Match(otherMsg))
}
