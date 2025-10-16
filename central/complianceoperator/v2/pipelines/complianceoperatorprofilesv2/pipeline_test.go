package complianceoperatorprofilesv2

import (
	"context"
	"testing"

	v2ProfileMocks "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/mocks"
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
	pipeline    *pipelineImpl
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

	s.v2ProfileDS = v2ProfileMocks.NewMockDataStore(s.mockCtrl)
	s.pipeline = NewPipeline(s.v2ProfileDS).(*pipelineImpl)
}

func (s *PipelineTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *PipelineTestSuite) TestRunCreate() {
	ctx := context.Background()

	pipeline := NewPipeline(s.v2ProfileDS)
	s.v2ProfileDS.EXPECT().UpsertProfile(ctx, testutils.GetProfileV2Storage(s.T())).Return(nil).Times(1)

	se := &central.SensorEvent{}
	se.SetId(testutils.ProfileUID)
	se.SetAction(central.ResourceAction_CREATE_RESOURCE)
	se.SetComplianceOperatorProfileV2(proto.ValueOrDefault(testutils.GetProfileV2SensorMsg(s.T())))
	msg := central.MsgFromSensor_builder{
		Event: se,
	}.Build()

	err := pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunDelete() {
	ctx := context.Background()

	pipeline := NewPipeline(s.v2ProfileDS)
	s.v2ProfileDS.EXPECT().DeleteProfileForCluster(ctx, testutils.ProfileUID, fixtureconsts.Cluster1).Return(nil).Times(1)

	se := &central.SensorEvent{}
	se.SetId(testutils.ProfileUID)
	se.SetAction(central.ResourceAction_REMOVE_RESOURCE)
	se.SetComplianceOperatorProfileV2(proto.ValueOrDefault(testutils.GetProfileV2SensorMsg(s.T())))
	msg := central.MsgFromSensor_builder{
		Event: se,
	}.Build()

	err := pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunReconcileNoOp() {
	ctx := context.Background()

	pipeline := NewPipeline(s.v2ProfileDS)

	s.v2ProfileDS.EXPECT().GetProfilesByClusters(ctx, []string{fixtureconsts.Cluster1}).Return(nil, nil).Times(1)

	err := pipeline.Reconcile(ctx, fixtureconsts.Cluster1, reconciliation.NewStoreMap())
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunReconcile() {
	ctx := context.Background()
	pipeline := NewPipeline(s.v2ProfileDS)

	s.v2ProfileDS.EXPECT().GetProfilesByClusters(ctx, []string{fixtureconsts.Cluster1}).Return([]*storage.ComplianceOperatorProfileV2{testutils.GetProfileV2Storage(s.T())}, nil).Times(1)
	s.v2ProfileDS.EXPECT().DeleteProfileForCluster(ctx, testutils.ProfileUID, fixtureconsts.Cluster1).Return(nil).Times(1)

	err := pipeline.Reconcile(ctx, fixtureconsts.Cluster1, reconciliation.NewStoreMap())
	s.NoError(err)
}
