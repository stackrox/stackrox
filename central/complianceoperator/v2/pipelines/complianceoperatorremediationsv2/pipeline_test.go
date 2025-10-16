package complianceoperatorremediationsv2

import (
	"context"
	"testing"

	v2RemediationsMocks "github.com/stackrox/rox/central/complianceoperator/v2/remediations/datastore/mocks"
	"github.com/stackrox/rox/central/convert/testutils"
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
	pipeline        *pipelineImpl
	v2RemediationDS *v2RemediationsMocks.MockDataStore
	mockCtrl        *gomock.Controller
}

func (s *PipelineTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceRemediationV2.EnvVar(), "true")
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceRemediationV2.Enabled() {
		s.T().Skip("Skip tests when ComplianceRemediationV2 disabled")
		s.T().SkipNow()
	}
}

func (s *PipelineTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.v2RemediationDS = v2RemediationsMocks.NewMockDataStore(s.mockCtrl)
	s.pipeline = NewPipeline(s.v2RemediationDS).(*pipelineImpl)
}

func (s *PipelineTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *PipelineTestSuite) TestRun() {
	ctx := context.Background()

	pipeline := NewPipeline(s.v2RemediationDS)

	// test create
	s.v2RemediationDS.EXPECT().UpsertRemediation(ctx, testutils.GetComplianceRemediationV2Storage(s.T())).Return(nil).Times(1)

	se := &central.SensorEvent{}
	se.SetId(testutils.RemediationUID)
	se.SetAction(central.ResourceAction_CREATE_RESOURCE)
	se.SetComplianceOperatorRemediationV2(proto.ValueOrDefault(testutils.GetComplianceRemediationV2Msg(s.T())))
	msg := central.MsgFromSensor_builder{
		Event: se,
	}.Build()

	err := pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.NoError(err)

	// test delete
	s.v2RemediationDS.EXPECT().DeleteRemediation(ctx, testutils.RemediationUID).Return(nil).Times(1)
	msg.GetEvent().SetAction(central.ResourceAction_REMOVE_RESOURCE)
	err = pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.Require().NoError(err)
}
