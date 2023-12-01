package complianceoperatorprofiles

import (
	"context"
	"testing"

	managerMocks "github.com/stackrox/rox/central/complianceoperator/manager/mocks"
	v1ProfileMocks "github.com/stackrox/rox/central/complianceoperator/profiles/datastore/mocks"
	v2ProfileMocks "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/mocks"
	"github.com/stackrox/rox/central/convert/testutils"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite
	pipeline    *pipelineImpl
	manager     *managerMocks.MockManager
	v1ProfileDS *v1ProfileMocks.MockDataStore
	v2ProfileDS *v2ProfileMocks.MockDataStore
	mockCtrl    *gomock.Controller
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
	s.v1ProfileDS = v1ProfileMocks.NewMockDataStore(s.mockCtrl)
	s.v2ProfileDS = v2ProfileMocks.NewMockDataStore(s.mockCtrl)
	s.pipeline = NewPipeline(s.v1ProfileDS, s.manager, s.v2ProfileDS).(*pipelineImpl)
}

func (s *PipelineTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *PipelineTestSuite) TestRunCreate() {
	ctx := context.Background()

	pipeline := NewPipeline(s.v1ProfileDS, s.manager, s.v2ProfileDS)
	s.manager.EXPECT().AddProfile(testutils.GetProfileV1SensorMsg(s.T())).Return(nil).Times(1)
	s.v2ProfileDS.EXPECT().UpsertProfile(ctx, testutils.GetProfileV2Storage(s.T()), fixtureconsts.Cluster1, testutils.ProfileUID).Return(nil).Times(1)

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     testutils.ProfileUID,
				Action: central.ResourceAction_CREATE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorProfileV2{
					ComplianceOperatorProfileV2: testutils.GetProfileV2SensorMsg(s.T()),
				},
			},
		},
	}

	err := pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunDelete() {
	ctx := context.Background()

	pipeline := NewPipeline(s.v1ProfileDS, s.manager, s.v2ProfileDS)
	s.manager.EXPECT().DeleteProfile(testutils.GetProfileV1SensorMsg(s.T())).Return(nil).Times(1)
	s.v2ProfileDS.EXPECT().DeleteProfileForCluster(ctx, testutils.ProfileUID, fixtureconsts.Cluster1).Return(nil).Times(1)

	msg := &central.MsgFromSensor{
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

	err := pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunV1Create() {
	ctx := context.Background()

	pipeline := NewPipeline(s.v1ProfileDS, s.manager, s.v2ProfileDS)
	s.manager.EXPECT().AddProfile(testutils.GetProfileV1SensorMsg(s.T())).Return(nil).Times(1)

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     testutils.ProfileUID,
				Action: central.ResourceAction_CREATE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorProfile{
					ComplianceOperatorProfile: testutils.GetProfileV1SensorMsg(s.T()),
				},
			},
		},
	}

	err := pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunV1Delete() {
	ctx := context.Background()

	pipeline := NewPipeline(s.v1ProfileDS, s.manager, s.v2ProfileDS)
	s.manager.EXPECT().DeleteProfile(testutils.GetProfileV1SensorMsg(s.T())).Return(nil).Times(1)

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     testutils.ProfileUID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorProfile{
					ComplianceOperatorProfile: testutils.GetProfileV1SensorMsg(s.T()),
				},
			},
		},
	}

	err := pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunReconcileNoOp() {
	ctx := context.Background()

	pipeline := NewPipeline(s.v1ProfileDS, s.manager, s.v2ProfileDS)

	s.v1ProfileDS.EXPECT().Walk(ctx, gomock.Any()).Return(nil).Times(1)
	s.v2ProfileDS.EXPECT().GetProfileEdgesByCluster(ctx, fixtureconsts.Cluster1).Return(nil, nil).Times(1)

	err := pipeline.Reconcile(ctx, fixtureconsts.Cluster1, reconciliation.NewStoreMap())
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunReconcile() {
	ctx := context.Background()

	edgeID := uuid.NewV4().String()
	edgeRecs := []*storage.ComplianceOperatorProfileClusterEdge{
		{
			Id:         edgeID,
			ProfileId:  "",
			ProfileUid: testutils.ProfileUID,
			ClusterId:  fixtureconsts.Cluster1,
		},
	}
	pipeline := NewPipeline(s.v1ProfileDS, s.manager, s.v2ProfileDS)

	s.v1ProfileDS.EXPECT().Walk(ctx, gomock.Any()).Return(nil).Times(1)
	s.v1ProfileDS.EXPECT().Delete(ctx, testutils.ProfileUID).Return(nil).Times(1)
	s.v2ProfileDS.EXPECT().GetProfileEdgesByCluster(ctx, fixtureconsts.Cluster1).Return(edgeRecs, nil).Times(1)
	s.v2ProfileDS.EXPECT().DeleteProfileForCluster(ctx, testutils.ProfileUID, fixtureconsts.Cluster1).Return(nil).Times(1)

	err := pipeline.Reconcile(ctx, fixtureconsts.Cluster1, reconciliation.NewStoreMap())
	s.NoError(err)
}
