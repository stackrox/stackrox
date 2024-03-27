package complianceoperatorresults

import (
	"context"
	"testing"

	v1ResultMocks "github.com/stackrox/rox/central/complianceoperator/checkresults/datastore/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	mockCheckRuleName = "mock-rule"
)

var (
	id      = uuid.NewV4().String()
	checkID = uuid.NewV4().String()
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite

	v1ResultDS *v1ResultMocks.MockDataStore
	mockCtrl   *gomock.Controller
}

func (suite *PipelineTestSuite) SetupSuite() {
	suite.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		suite.T().Skip("Skip tests when ComplianceEnhancements disabled")
		suite.T().SkipNow()
	}
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.v1ResultDS = v1ResultMocks.NewMockDataStore(suite.mockCtrl)
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestRunV1Create() {
	ctx := context.Background()

	suite.v1ResultDS.EXPECT().Upsert(ctx, getV1TestRec(fixtureconsts.Cluster1)).Return(nil).Times(1)
	pipeline := NewPipeline(suite.v1ResultDS)

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     id,
				Action: central.ResourceAction_CREATE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorResult{
					ComplianceOperatorResult: &storage.ComplianceOperatorCheckResult{
						Id:           id,
						CheckId:      checkID,
						CheckName:    mockCheckRuleName,
						ClusterId:    fixtureconsts.Cluster1,
						Status:       storage.ComplianceOperatorCheckResult_FAIL,
						Description:  "this is a test",
						Instructions: "this is a test",
						Labels:       nil,
						Annotations:  nil,
					},
				},
			},
		},
	}

	err := pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	suite.NoError(err)
}

func (suite *PipelineTestSuite) TestRunV1Delete() {
	ctx := context.Background()

	suite.v1ResultDS.EXPECT().Delete(ctx, id).Return(nil).Times(1)
	pipeline := NewPipeline(suite.v1ResultDS)

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     id,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorResult{
					ComplianceOperatorResult: &storage.ComplianceOperatorCheckResult{
						Id:           id,
						CheckId:      checkID,
						CheckName:    mockCheckRuleName,
						ClusterId:    fixtureconsts.Cluster1,
						Status:       storage.ComplianceOperatorCheckResult_FAIL,
						Description:  "this is a test",
						Instructions: "this is a test",
						Labels:       nil,
						Annotations:  nil,
					},
				},
			},
		},
	}

	err := pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	suite.NoError(err)
}

func getV1TestRec(clusterID string) *storage.ComplianceOperatorCheckResult {
	return &storage.ComplianceOperatorCheckResult{
		Id:           id,
		CheckId:      checkID,
		CheckName:    mockCheckRuleName,
		ClusterId:    clusterID,
		Status:       storage.ComplianceOperatorCheckResult_FAIL,
		Description:  "this is a test",
		Instructions: "this is a test",
		Labels:       nil,
		Annotations:  nil,
	}
}
