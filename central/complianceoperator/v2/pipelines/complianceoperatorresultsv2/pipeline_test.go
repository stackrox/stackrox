package complianceoperatorresultsv2

import (
	"context"
	"testing"
	"time"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	v2ResultMocks "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/mocks"
	reportMgrMocks "github.com/stackrox/rox/central/complianceoperator/v2/report/manager/mocks"
	"github.com/stackrox/rox/central/convert/internaltov2storage"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	// mockLastScannedTime/mockOlderLastScannedTime/mockNewerLastScannedTime simulate the compliance
	// operator's last-scanned-timestamp annotation, which is what LastStartedTime is parsed from
	// (see internaltov2storage.ComplianceOperatorCheckResult). Real check results always carry this
	// once a check has actually run, so tests use realistic non-nil values instead of the nil
	// first-receipt case.
	mockLastScannedTime, _      = protocompat.ParseRFC3339NanoTimestamp("2024-06-01T00:00:00.000000000Z")
	mockOlderLastScannedTime, _ = protocompat.ParseRFC3339NanoTimestamp("2024-01-01T00:00:00.000000000Z")
	mockNewerLastScannedTime, _ = protocompat.ParseRFC3339NanoTimestamp("2024-12-01T00:00:00.000000000Z")
)

func annotationsWithLastScanned(ts *timestamppb.Timestamp) map[string]string {
	return map[string]string{
		internaltov2storage.LastScannedAnnotationKey: ts.AsTime().Format(time.RFC3339Nano),
	}
}

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite

	v2ResultDS *v2ResultMocks.MockDataStore
	clusterDS  *clusterMocks.MockDataStore
	reportMgr  *reportMgrMocks.MockManager
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
	suite.reportMgr = reportMgrMocks.NewMockManager(suite.mockCtrl)
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestRunCreate() {
	ctx := context.Background()

	suite.clusterDS.EXPECT().GetClusterName(ctx, fixtureconsts.Cluster1).Return("cluster1", true, nil).Times(1)
	suite.v2ResultDS.EXPECT().UpsertResult(ctx, getTestRec(fixtureconsts.Cluster1)).Return(nil).Times(1)
	suite.reportMgr.EXPECT().HandleResult(gomock.Any(), gomock.Any()).Times(1)
	pipeline := NewPipeline(suite.v2ResultDS, suite.clusterDS, suite.reportMgr)

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

// TestRunCreateMarksCurrentWhenFreshnessEnabled covers the first time a result is ever seen (no
// prior row): a non-nil LastStartedTime cannot be a stale replay, so it is marked CURRENT.
func (suite *PipelineTestSuite) TestRunCreateMarksCurrentWhenFreshnessEnabled() {
	suite.T().Setenv(features.ComplianceSurfaceStaleData.EnvVar(), "true")
	if !features.ComplianceSurfaceStaleData.Enabled() {
		suite.T().Skip("Skip test when ComplianceSurfaceStaleData disabled")
	}
	ctx := context.Background()

	expected := getTestRec(fixtureconsts.Cluster1)
	expected.Annotations = annotationsWithLastScanned(mockLastScannedTime)
	expected.LastStartedTime = mockLastScannedTime
	expected.DataFreshness = storage.ComplianceOperatorCheckResultV2_CURRENT

	suite.clusterDS.EXPECT().GetClusterName(ctx, fixtureconsts.Cluster1).Return("cluster1", true, nil).Times(1)
	suite.v2ResultDS.EXPECT().GetComplianceCheckResult(ctx, id).Return(nil, false, nil).Times(1)
	suite.v2ResultDS.EXPECT().UpsertResult(ctx, expected).Return(nil).Times(1)
	suite.reportMgr.EXPECT().HandleResult(gomock.Any(), gomock.Any()).Times(1)
	pipeline := NewPipeline(suite.v2ResultDS, suite.clusterDS, suite.reportMgr)

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
						Annotations:  annotationsWithLastScanned(mockLastScannedTime),
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

// TestRunCreateMarksCurrentWhenNewerThanStored covers a genuinely new scan run landing on top of
// an existing (possibly stale) row: the incoming LastStartedTime is newer than what is stored, so
// it is marked CURRENT regardless of the row's previous freshness.
func (suite *PipelineTestSuite) TestRunCreateMarksCurrentWhenNewerThanStored() {
	suite.T().Setenv(features.ComplianceSurfaceStaleData.EnvVar(), "true")
	if !features.ComplianceSurfaceStaleData.Enabled() {
		suite.T().Skip("Skip test when ComplianceSurfaceStaleData disabled")
	}
	ctx := context.Background()

	stored := getTestRec(fixtureconsts.Cluster1)
	stored.LastStartedTime = mockOlderLastScannedTime
	stored.DataFreshness = storage.ComplianceOperatorCheckResultV2_STALE

	expected := getTestRec(fixtureconsts.Cluster1)
	expected.Annotations = annotationsWithLastScanned(mockNewerLastScannedTime)
	expected.LastStartedTime = mockNewerLastScannedTime
	expected.DataFreshness = storage.ComplianceOperatorCheckResultV2_CURRENT

	suite.clusterDS.EXPECT().GetClusterName(ctx, fixtureconsts.Cluster1).Return("cluster1", true, nil).Times(1)
	suite.v2ResultDS.EXPECT().GetComplianceCheckResult(ctx, id).Return(stored, true, nil).Times(1)
	suite.v2ResultDS.EXPECT().UpsertResult(ctx, expected).Return(nil).Times(1)
	suite.reportMgr.EXPECT().HandleResult(gomock.Any(), gomock.Any()).Times(1)
	pipeline := NewPipeline(suite.v2ResultDS, suite.clusterDS, suite.reportMgr)

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
						Annotations:  annotationsWithLastScanned(mockNewerLastScannedTime),
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

// TestRunCreateDoesNotMarkCurrentOnStaleReplay is the regression test for the gap found in review:
// sensor resends every existing CheckResult CR on reconnect regardless of whether the compliance
// operator actually rescanned (compliance resources are not deduplicated), and a misbehaving
// operator could republish a result the same way. In both cases the incoming LastStartedTime is
// not newer than what is already stored, so the receive hook must NOT mark it CURRENT -- that
// would launder stale data as fresh, with no sweeper in this slice to correct it later. The
// existing row's freshness must be carried over unchanged instead of defaulting to UNKNOWN.
func (suite *PipelineTestSuite) TestRunCreateDoesNotMarkCurrentOnStaleReplay() {
	suite.T().Setenv(features.ComplianceSurfaceStaleData.EnvVar(), "true")
	if !features.ComplianceSurfaceStaleData.Enabled() {
		suite.T().Skip("Skip test when ComplianceSurfaceStaleData disabled")
	}
	ctx := context.Background()

	stored := getTestRec(fixtureconsts.Cluster1)
	stored.LastStartedTime = mockLastScannedTime
	stored.DataFreshness = storage.ComplianceOperatorCheckResultV2_STALE

	// The replayed message carries the SAME LastStartedTime as what is already stored: no new
	// scan actually ran.
	expected := getTestRec(fixtureconsts.Cluster1)
	expected.Annotations = annotationsWithLastScanned(mockLastScannedTime)
	expected.LastStartedTime = mockLastScannedTime
	expected.DataFreshness = storage.ComplianceOperatorCheckResultV2_STALE

	suite.clusterDS.EXPECT().GetClusterName(ctx, fixtureconsts.Cluster1).Return("cluster1", true, nil).Times(1)
	suite.v2ResultDS.EXPECT().GetComplianceCheckResult(ctx, id).Return(stored, true, nil).Times(1)
	suite.v2ResultDS.EXPECT().UpsertResult(ctx, expected).Return(nil).Times(1)
	suite.reportMgr.EXPECT().HandleResult(gomock.Any(), gomock.Any()).Times(1)
	pipeline := NewPipeline(suite.v2ResultDS, suite.clusterDS, suite.reportMgr)

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
						Annotations:  annotationsWithLastScanned(mockLastScannedTime),
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
	pipeline := NewPipeline(suite.v2ResultDS, suite.clusterDS, suite.reportMgr)

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
		ScanRefId:      internaltov2storage.BuildNameRefID(clusterID, mockScanName),
		RuleRefId:      internaltov2storage.BuildNameRefID(clusterID, ""),
	}
}
