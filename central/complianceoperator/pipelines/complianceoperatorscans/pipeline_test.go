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

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     testutils.ScanUID,
				Action: central.ResourceAction_CREATE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorScan{
					ComplianceOperatorScan: testutils.GetScanV1Storage(s.T()),
				},
			},
		},
	}

	err := s.pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunV1Delete() {
	ctx := context.Background()

	s.manager.EXPECT().DeleteScan(testutils.GetScanV1Storage(s.T())).Return(nil).Times(1)

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     testutils.ScanUID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorScan{
					ComplianceOperatorScan: testutils.GetScanV1Storage(s.T()),
				},
			},
		},
	}

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
	v1Msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     testutils.ScanUID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorScan{
					ComplianceOperatorScan: testutils.GetScanV1Storage(s.T()),
				},
			},
		},
	}

	v2Msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     testutils.ScanUID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorScanV2{
					ComplianceOperatorScanV2: testutils.GetScanV2SensorMsg(s.T()),
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

	s.Require().True(s.pipeline.Match(v1Msg))
	s.Require().False(s.pipeline.Match(v2Msg))
	s.Require().False(s.pipeline.Match(otherMsg))
}
