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

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     testutils.ScanSettingUID,
				Action: central.ResourceAction_CREATE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorScanSettingBindingV2{
					ComplianceOperatorScanSettingBindingV2: testutils.GetScanSettingBindingV2SensorMsg(s.T()),
				},
			},
		},
	}

	err := s.pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunDelete() {
	ctx := context.Background()

	s.v2DS.EXPECT().DeleteScanSettingBinding(ctx, testutils.ScanSettingUID).Return(nil).Times(1)

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     testutils.ScanSettingUID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorScanSettingBindingV2{
					ComplianceOperatorScanSettingBindingV2: testutils.GetScanSettingBindingV2SensorMsg(s.T()),
				},
			},
		},
	}

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
	v1Msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     testutils.ScanSettingUID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorScanSettingBinding{
					ComplianceOperatorScanSettingBinding: testutils.GetScanSettingBindingV1Storage(s.T(), fixtureconsts.Cluster1),
				},
			},
		},
	}

	v2Msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     testutils.ScanSettingUID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorScanSettingBindingV2{
					ComplianceOperatorScanSettingBindingV2: testutils.GetScanSettingBindingV2SensorMsg(s.T()),
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
