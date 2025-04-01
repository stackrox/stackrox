//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	reportStorage "github.com/stackrox/rox/central/complianceoperator/v2/report/store/postgres"
	scanConfigDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	uuidStub1           = "933cf32f-d387-4787-8835-65857b5fdbfd"
	uuidStub2           = "8f9850a8-b615-4a12-a3da-7b057bf3aeba"
	uuidNonExisting     = "1e52b778-63f2-4eab-aa81-c9b6381ceb02"
	uuidScanConfigStub1 = "001cf32f-d387-4787-8835-65857b5fdbfd"
	uuidScanConfigStub2 = "002cf32f-d387-4787-8835-65857b5fdbfd"
)

func TestComplianceReportSnapshotDataStore(t *testing.T) {
	suite.Run(t, new(complianceReportSnapshotDataStoreSuite))
}

type complianceReportSnapshotDataStoreSuite struct {
	suite.Suite

	hasReadCtx  context.Context
	hasWriteCtx context.Context
	noAccessCtx context.Context

	datastore    DataStore
	storage      reportStorage.Store
	db           *pgtest.TestPostgres
	scanConfigDB scanConfigDS.DataStore
}

func (s *complianceReportSnapshotDataStoreSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skipf("Skip test when %s is disabled", features.ComplianceEnhancements.EnvVar())
		s.T().SkipNow()
	}
}

func (s *complianceReportSnapshotDataStoreSuite) SetupTest() {
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.noAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())

	s.db = pgtest.ForT(s.T())
	s.storage = reportStorage.New(s.db)
	s.datastore = New(s.storage)

	s.scanConfigDB = scanConfigDS.GetTestPostgresDataStore(s.T(), s.db)
	require.NoError(s.T(), s.scanConfigDB.UpsertScanConfiguration(s.hasWriteCtx, &storage.ComplianceOperatorScanConfigurationV2{Id: uuidScanConfigStub1, ScanConfigName: uuidScanConfigStub1}))
	require.NoError(s.T(), s.scanConfigDB.UpsertScanConfiguration(s.hasWriteCtx, &storage.ComplianceOperatorScanConfigurationV2{Id: uuidScanConfigStub2, ScanConfigName: uuidScanConfigStub2}))
}

func (s *complianceReportSnapshotDataStoreSuite) TestUpsertReport() {
	// make sure we have nothing
	reportIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(reportIDs)

	status := getStatus(storage.ComplianceOperatorReportStatus_PREPARING, timestamp.Now().Protobuf(), nil, "", storage.ComplianceOperatorReportStatus_SCHEDULED, storage.ComplianceOperatorReportStatus_EMAIL)
	user := getUser("u-1", "user-1")
	r1 := getTestReport(uuidStub1, uuidScanConfigStub1, status, user)
	r2 := getTestReport(uuidStub2, uuidScanConfigStub2, status, user)

	s.Require().NoError(s.datastore.UpsertSnapshot(s.hasWriteCtx, r1))
	s.Require().NoError(s.datastore.UpsertSnapshot(s.hasWriteCtx, r2))

	count, err := s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(2, count)

	s.Require().Error(s.datastore.UpsertSnapshot(s.hasReadCtx, r1))

	retR1, found, err := s.storage.Get(s.hasReadCtx, r1.GetReportId())
	s.Require().NoError(err)
	s.Require().True(found)
	assertReports(s.T(), r1, retR1)

	retR2, found, err := s.storage.Get(s.hasReadCtx, r2.GetReportId())
	s.Require().NoError(err)
	s.Require().True(found)
	assertReports(s.T(), r2, retR2)
}

func (s *complianceReportSnapshotDataStoreSuite) TestDeleteReport() {
	// make sure we have nothing
	reportIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(reportIDs)

	status := getStatus(storage.ComplianceOperatorReportStatus_PREPARING, timestamp.Now().Protobuf(), nil, "", storage.ComplianceOperatorReportStatus_SCHEDULED, storage.ComplianceOperatorReportStatus_EMAIL)
	user := getUser("u-1", "user-1")
	r1 := getTestReport(uuidStub1, uuidScanConfigStub1, status, user)
	r2 := getTestReport(uuidStub2, uuidScanConfigStub2, status, user)

	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, r1))
	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, r2))

	count, err := s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(2, count)

	s.Require().NoError(s.datastore.DeleteSnapshot(s.hasWriteCtx, r1.GetReportId()))

	count, err = s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(1, count)

	s.Require().NoError(s.datastore.DeleteSnapshot(s.noAccessCtx, r2.GetReportId()))

	count, err = s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(1, count)

	s.Require().NoError(s.datastore.DeleteSnapshot(s.hasWriteCtx, r2.GetReportId()))

	count, err = s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(0, count)
}
func (s *complianceReportSnapshotDataStoreSuite) TestGetReports() {
	// make sure we have nothing
	reportIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(reportIDs)

	status := getStatus(storage.ComplianceOperatorReportStatus_PREPARING, timestamp.Now().Protobuf(), nil, "", storage.ComplianceOperatorReportStatus_SCHEDULED, storage.ComplianceOperatorReportStatus_EMAIL)
	user := getUser("u-1", "user-1")
	reports := []*storage.ComplianceOperatorReportSnapshotV2{
		getTestReport(uuidStub1, uuidScanConfigStub1, status, user),
		getTestReport(uuidStub2, uuidScanConfigStub2, status, user),
	}

	for _, r := range reports {
		s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, r))
	}

	for _, r := range reports {
		retR, found, err := s.datastore.GetSnapshot(s.hasReadCtx, r.GetReportId())
		s.Require().NoError(err)
		s.Require().True(found)
		assertReports(s.T(), r, retR)
	}

	_, found, err := s.datastore.GetSnapshot(s.noAccessCtx, reports[0].GetReportId())
	s.Require().NoError(err)
	s.Require().False(found)

	_, found, err = s.datastore.GetSnapshot(s.hasReadCtx, uuidNonExisting)
	s.Require().NoError(err)
	s.Require().False(found)
}

func getTestReport(id string, scanConfigID string, status *storage.ComplianceOperatorReportStatus, user *storage.SlimUser) *storage.ComplianceOperatorReportSnapshotV2 {
	return &storage.ComplianceOperatorReportSnapshotV2{
		ReportId:            id,
		ScanConfigurationId: scanConfigID,
		Name:                fmt.Sprintf("name-%s", scanConfigID),
		Description:         fmt.Sprintf("description-%s", scanConfigID),
		ReportStatus:        status,
		User:                user,
	}
}

func getStatus(state storage.ComplianceOperatorReportStatus_RunState, started *timestamppb.Timestamp, completed *timestamppb.Timestamp, errorMsg string, reportType storage.ComplianceOperatorReportStatus_RunMethod, notification storage.ComplianceOperatorReportStatus_NotificationMethod) *storage.ComplianceOperatorReportStatus {
	return &storage.ComplianceOperatorReportStatus{
		RunState:                 state,
		StartedAt:                started,
		CompletedAt:              completed,
		ErrorMsg:                 errorMsg,
		ReportRequestType:        reportType,
		ReportNotificationMethod: notification,
	}
}

func getUser(id, name string) *storage.SlimUser {
	return &storage.SlimUser{
		Id:   id,
		Name: name,
	}
}

func assertReports(t *testing.T, expected *storage.ComplianceOperatorReportSnapshotV2, actual *storage.ComplianceOperatorReportSnapshotV2) {
	assert.Equal(t, expected.GetReportId(), actual.GetReportId())
	assert.Equal(t, expected.GetName(), actual.GetName())
	assert.Equal(t, expected.GetDescription(), actual.GetDescription())
	assert.Equal(t, expected.GetReportStatus().GetRunState(), actual.GetReportStatus().GetRunState())
	assert.Equal(t, expected.GetReportStatus().GetStartedAt(), actual.GetReportStatus().GetStartedAt())
	assert.Equal(t, expected.GetReportStatus().GetCompletedAt(), actual.GetReportStatus().GetCompletedAt())
	assert.Equal(t, expected.GetReportStatus().GetErrorMsg(), actual.GetReportStatus().GetErrorMsg())
	assert.Equal(t, expected.GetReportStatus().GetReportRequestType(), actual.GetReportStatus().GetReportRequestType())
	assert.Equal(t, expected.GetReportStatus().GetReportNotificationMethod(), actual.GetReportStatus().GetReportNotificationMethod())
	assert.Equal(t, expected.GetUser().GetId(), actual.GetUser().GetId())
	assert.Equal(t, expected.GetUser().GetName(), actual.GetUser().GetName())
}
