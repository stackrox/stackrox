package complianceoperatorresultsv2

import (
	"context"
	"testing"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	v2ResultMocks "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/mocks"
	"github.com/stackrox/rox/central/convert/internaltov2storage"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	mockScanName      = "mock-scan"
	mockCheckRuleName = "mock-rule"
	mockSuiteName     = "mock-suite"
)

var (
	createdTime = protocompat.TimestampNow()
	id          = uuid.NewV4().String()
	checkID     = uuid.NewV4().String()
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite

	v2ResultDS *v2ResultMocks.MockDataStore
	clusterDS  *clusterMocks.MockDataStore
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
	suite.v2ResultDS = v2ResultMocks.NewMockDataStore(suite.mockCtrl)
	suite.clusterDS = clusterMocks.NewMockDataStore(suite.mockCtrl)
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestRunCreate() {
	ctx := context.Background()

	suite.clusterDS.EXPECT().GetClusterName(ctx, fixtureconsts.Cluster1).Return("cluster1", true, nil).Times(1)
	suite.v2ResultDS.EXPECT().UpsertResult(ctx, getTestRec(fixtureconsts.Cluster1)).Return(nil).Times(1)
	pipeline := NewPipeline(suite.v2ResultDS, suite.clusterDS)

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     id,
				Action: central.ResourceAction_CREATE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorResultV2{
					ComplianceOperatorResultV2: &central.ComplianceOperatorCheckResultV2{
						Id:           id,
						CheckId:      checkID,
						CheckName:    mockCheckRuleName,
						ClusterId:    fixtureconsts.Cluster1,
						Status:       central.ComplianceOperatorCheckResultV2_FAIL,
						Severity:     central.ComplianceOperatorRuleSeverity_HIGH_RULE_SEVERITY,
						Description:  "this is a test",
						Instructions: "this is a test",
						Labels:       nil,
						Annotations:  nil,
						CreatedTime:  createdTime,
						ScanName:     mockScanName,
						SuiteName:    mockSuiteName,
						Rationale:    "test rationale",
						ValuesUsed:   []string{"var1", "var2"},
						Warnings:     []string{"warning1", "warning2"},
					},
				},
			},
		},
	}

	err := pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	suite.NoError(err)
}

func (suite *PipelineTestSuite) TestRunDelete() {
	ctx := context.Background()

	suite.v2ResultDS.EXPECT().DeleteResult(ctx, id).Return(nil).Times(1)
	pipeline := NewPipeline(suite.v2ResultDS, suite.clusterDS)

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     id,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorResultV2{
					ComplianceOperatorResultV2: &central.ComplianceOperatorCheckResultV2{
						Id:           id,
						CheckId:      checkID,
						CheckName:    mockCheckRuleName,
						ClusterId:    fixtureconsts.Cluster1,
						Status:       central.ComplianceOperatorCheckResultV2_FAIL,
						Severity:     central.ComplianceOperatorRuleSeverity_HIGH_RULE_SEVERITY,
						Description:  "this is a test",
						Instructions: "this is a test",
						Labels:       nil,
						Annotations:  nil,
						CreatedTime:  createdTime,
						ScanName:     mockScanName,
						SuiteName:    mockSuiteName,
						Rationale:    "test rationale",
						ValuesUsed:   []string{"var1", "var2"},
						Warnings:     []string{"warning1", "warning2"},
					},
				},
			},
		},
	}

	err := pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	suite.NoError(err)
}

func getTestRec(clusterID string) *storage.ComplianceOperatorCheckResultV2 {
	return &storage.ComplianceOperatorCheckResultV2{
		Id:             id,
		CheckId:        checkID,
		CheckName:      mockCheckRuleName,
		ClusterId:      clusterID,
		ClusterName:    "cluster1",
		Status:         storage.ComplianceOperatorCheckResultV2_FAIL,
		Severity:       storage.RuleSeverity_HIGH_RULE_SEVERITY,
		Description:    "this is a test",
		Instructions:   "this is a test",
		Labels:         nil,
		Annotations:    nil,
		CreatedTime:    createdTime,
		ScanConfigName: mockSuiteName,
		ScanName:       mockScanName,
		Rationale:      "test rationale",
		ValuesUsed:     []string{"var1", "var2"},
		Warnings:       []string{"warning1", "warning2"},
		ScanRefId:      internaltov2storage.BuildScanRefID(clusterID, mockScanName),
	}
}
