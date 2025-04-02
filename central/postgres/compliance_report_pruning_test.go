//go:build sql_integration

package postgres

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	blobDS "github.com/stackrox/rox/central/blob/datastore"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	scanConfigDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	"github.com/stackrox/rox/central/reports/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

const (
	day = time.Hour * 24
)

var (
	report1  = uuid.NewV4().String()
	report2  = uuid.NewV4().String()
	report3  = uuid.NewV4().String()
	report4  = uuid.NewV4().String()
	report5  = uuid.NewV4().String()
	report6  = uuid.NewV4().String()
	report7  = uuid.NewV4().String()
	report8  = uuid.NewV4().String()
	report9  = uuid.NewV4().String()
	report10 = uuid.NewV4().String()
	report11 = uuid.NewV4().String()
	report12 = uuid.NewV4().String()

	idToHumanReadable = map[string]string{
		report1:  "report-1",
		report2:  "report-2",
		report3:  "report-3",
		report4:  "report-4",
		report5:  "report-5",
		report6:  "report-6",
		report7:  "report-7",
		report8:  "report-8",
		report9:  "report-9",
		report10: "report-10",
		report11: "report-11",
		report12: "report-12",
	}

	blobData = []byte("some-test-date")
)

type ComplianceReportPruningSuite struct {
	suite.Suite
	ctx          context.Context
	db           *pgtest.TestPostgres
	snapshotDS   snapshotDS.DataStore
	scanConfigDS scanConfigDS.DataStore
	blobDS       blobDS.Datastore
}

func TestComplianceReportPruning(t *testing.T) {
	suite.Run(t, new(ComplianceReportPruningSuite))
}

var _ suite.SetupAllSuite = (*ComplianceReportPruningSuite)(nil)
var _ suite.TearDownTestSuite = (*ComplianceReportPruningSuite)(nil)

func (s *ComplianceReportPruningSuite) SetupSuite() {
	s.db = pgtest.ForT(s.T())
	s.ctx = sac.WithAllAccess(context.Background())

	s.scanConfigDS = scanConfigDS.GetTestPostgresDataStore(s.T(), s.db)
	s.snapshotDS = snapshotDS.GetTestPostgresDataStore(s.T(), s.db)
	s.blobDS = blobDS.NewTestDatastore(s.T(), s.db)

	for _, id := range []string{config1, config2, config3, config4} {
		s.Require().NoError(s.scanConfigDS.UpsertScanConfiguration(s.ctx, newScanConfig(id)))
	}
}

func (s *ComplianceReportPruningSuite) TearDownTest() {
	tag, err := s.db.Exec(s.ctx, fmt.Sprintf("TRUNCATE %s CASCADE", schema.ComplianceOperatorReportSnapshotV2TableName))
	s.T().Logf("%s %v", schema.ComplianceOperatorReportSnapshotV2TableName, tag)
	s.Require().NoError(err)

	tag, err = s.db.Exec(s.ctx, fmt.Sprintf("TRUNCATE %s CASCADE", schema.BlobsTableName))
	s.T().Logf("%s %v", schema.BlobsTableName, tag)
	s.Require().NoError(err)
}

func (s *ComplianceReportPruningSuite) Test_MustNotDeleteLastSuccessfulJobOneDownload() {
	// All downloads; all delivered; outside retention window
	// These should not be deleted because they are the last successful job for each Scan Configuration
	snapshots := []*storage.ComplianceOperatorReportSnapshotV2{
		newComplianceReportSnapshot(report1, config1, 100*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report2, config2, 100*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report3, config3, 100*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report4, config4, 100*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),
	}
	expectedDeletions := set.NewStringSet()
	for _, snapshot := range snapshots {
		s.Require().NoError(s.snapshotDS.UpsertSnapshot(s.ctx, snapshot))
	}
	s.PruneAndAssert(snapshots, expectedDeletions)
}

func (s *ComplianceReportPruningSuite) Test_MustNotDeleteLastSuccessfulJobOnDemandOneEmail() {
	// All emails; all delivered; all on demand; outside retention window
	// These should not be deleted because they are the last successful job for each Scan Configuration
	snapshots := []*storage.ComplianceOperatorReportSnapshotV2{
		newComplianceReportSnapshot(report1, config1, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report2, config2, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report3, config3, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report4, config4, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, true),
	}
	expectedDeletions := set.NewStringSet()
	for _, snapshot := range snapshots {
		s.Require().NoError(s.snapshotDS.UpsertSnapshot(s.ctx, snapshot))
	}
	s.PruneAndAssert(snapshots, expectedDeletions)
}

func (s *ComplianceReportPruningSuite) Test_MustNotDeleteLastSuccessfulJobScheduledOneEmail() {
	// All emails; all delivered; all scheduled; outside retention window
	// These should not be deleted because they are the last successful job for each Scan Configuration
	snapshots := []*storage.ComplianceOperatorReportSnapshotV2{
		newComplianceReportSnapshot(report1, config1, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, false),
		newComplianceReportSnapshot(report2, config2, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, false),
		newComplianceReportSnapshot(report3, config3, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, false),
		newComplianceReportSnapshot(report4, config4, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, false),
	}
	expectedDeletions := set.NewStringSet()
	for _, snapshot := range snapshots {
		s.Require().NoError(s.snapshotDS.UpsertSnapshot(s.ctx, snapshot))
	}
	s.PruneAndAssert(snapshots, expectedDeletions)
}

func (s *ComplianceReportPruningSuite) Test_MustNotDeleteLastSuccessfulJobNewerDownloadsOlderFailedEmails() {
	// Newer delivered downloads; older failed scheduled; outside retention window
	snapshots := []*storage.ComplianceOperatorReportSnapshotV2{
		newComplianceReportSnapshot(report1, config1, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_FAILURE, false),
		newComplianceReportSnapshot(report2, config2, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_FAILURE, false),
		newComplianceReportSnapshot(report3, config3, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_FAILURE, false),
		newComplianceReportSnapshot(report4, config4, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_FAILURE, false),

		newComplianceReportSnapshot(report5, config1, 10*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report6, config2, 10*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report7, config3, 10*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report8, config4, 10*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),
	}
	expectedDeletions := set.NewStringSet(report1, report2, report3, report4)
	for _, snapshot := range snapshots {
		s.Require().NoError(s.snapshotDS.UpsertSnapshot(s.ctx, snapshot))
	}
	s.PruneAndAssert(snapshots, expectedDeletions)
}

func newScanConfig(id string) *storage.ComplianceOperatorScanConfigurationV2 {
	return &storage.ComplianceOperatorScanConfigurationV2{
		Id:             id,
		ScanConfigName: id,
	}
}

func (s *ComplianceReportPruningSuite) Test_MustNotDeleteLastSuccessfulJobOlderDownloadsNewerFailedEmails() {
	// Older delivered downloads; newer failed scheduled; outside retention window
	snapshots := []*storage.ComplianceOperatorReportSnapshotV2{
		newComplianceReportSnapshot(report1, config1, 100*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report2, config2, 100*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report3, config3, 100*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report4, config4, 100*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),

		newComplianceReportSnapshot(report5, config1, 10*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_FAILURE, false),
		newComplianceReportSnapshot(report6, config2, 10*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_FAILURE, false),
		newComplianceReportSnapshot(report7, config3, 10*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_FAILURE, false),
		newComplianceReportSnapshot(report8, config4, 10*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_FAILURE, false),
	}
	expectedDeletions := set.NewStringSet(report5, report6, report7, report8)
	for _, snapshot := range snapshots {
		s.Require().NoError(s.snapshotDS.UpsertSnapshot(s.ctx, snapshot))
	}
	s.PruneAndAssert(snapshots, expectedDeletions)
}

func (s *ComplianceReportPruningSuite) Test_MustNotDeleteLastSuccessfulJobOldMixedEmailsNewerMixedEmails() {
	// Older mixed scheduled; newer mixed scheduled; even newer mixed on demand emails; outside retention window
	snapshots := []*storage.ComplianceOperatorReportSnapshotV2{
		newComplianceReportSnapshot(report1, config1, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, false),
		newComplianceReportSnapshot(report2, config2, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_FAILURE, false),
		newComplianceReportSnapshot(report3, config3, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, false),
		newComplianceReportSnapshot(report4, config4, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_FAILURE, false),

		newComplianceReportSnapshot(report5, config1, 20*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_FAILURE, false),
		newComplianceReportSnapshot(report6, config2, 20*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, false),
		newComplianceReportSnapshot(report7, config3, 20*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_FAILURE, false),
		newComplianceReportSnapshot(report8, config4, 20*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, false),

		newComplianceReportSnapshot(report9, config1, 10*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_FAILURE, true),
		newComplianceReportSnapshot(report10, config2, 10*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report11, config3, 10*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_FAILURE, true),
		newComplianceReportSnapshot(report12, config4, 10*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, true),
	}
	expectedDeletions := set.NewStringSet(report2, report4, report5, report7, report9, report11)
	for _, snapshot := range snapshots {
		s.Require().NoError(s.snapshotDS.UpsertSnapshot(s.ctx, snapshot))
	}
	s.PruneAndAssert(snapshots, expectedDeletions)
}

func (s *ComplianceReportPruningSuite) Test_MustNotDeleteLastSuccessfulJobAllEmails() {
	// All emails; all scheduled; all delivered; outside retention window
	snapshots := []*storage.ComplianceOperatorReportSnapshotV2{
		newComplianceReportSnapshot(report1, config1, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, false),
		newComplianceReportSnapshot(report2, config2, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, false),
		newComplianceReportSnapshot(report3, config3, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, false),
		newComplianceReportSnapshot(report4, config4, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, false),

		newComplianceReportSnapshot(report5, config1, 20*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, false),
		newComplianceReportSnapshot(report6, config2, 20*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, false),
		newComplianceReportSnapshot(report7, config3, 20*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, false),
		newComplianceReportSnapshot(report8, config4, 20*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, false),
	}
	expectedDeletions := set.NewStringSet(report1, report2, report3, report4)
	for _, snapshot := range snapshots {
		s.Require().NoError(s.snapshotDS.UpsertSnapshot(s.ctx, snapshot))
	}
	s.PruneAndAssert(snapshots, expectedDeletions)
}

func (s *ComplianceReportPruningSuite) Test_MustNotDeleteLastSuccessfulJobAllDownloads() {
	// All downloads; some delivered; outside retention window
	snapshots := []*storage.ComplianceOperatorReportSnapshotV2{
		newComplianceReportSnapshot(report1, config1, 100*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report2, config2, 100*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report3, config3, 100*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report4, config4, 100*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),

		newComplianceReportSnapshot(report5, config1, 20*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report6, config2, 20*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report7, config3, 20*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report8, config4, 20*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),
	}
	expectedDeletions := set.NewStringSet(report1, report2, report3, report4)
	for _, snapshot := range snapshots {
		s.Require().NoError(s.snapshotDS.UpsertSnapshot(s.ctx, snapshot))
	}
	s.PruneAndAssert(snapshots, expectedDeletions)
}

func (s *ComplianceReportPruningSuite) Test_MustNotDeleteUnfinishedJobs() {
	// All unfinished; outside retention window
	snapshots := []*storage.ComplianceOperatorReportSnapshotV2{
		newComplianceReportSnapshot(report1, config1, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_WAITING, true),
		newComplianceReportSnapshot(report2, config2, 100*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_WAITING, true),
		newComplianceReportSnapshot(report3, config3, 100*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_PREPARING, true),
		newComplianceReportSnapshot(report4, config4, 100*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_PREPARING, true),
	}
	expectedDeletions := set.NewStringSet()
	for _, snapshot := range snapshots {
		s.Require().NoError(s.snapshotDS.UpsertSnapshot(s.ctx, snapshot))
	}
	s.PruneAndAssert(snapshots, expectedDeletions)
}

func (s *ComplianceReportPruningSuite) Test_MustNotDeleteIfBlobExists() {
	// Blob still exits; outside retention window
	snapshots := []*storage.ComplianceOperatorReportSnapshotV2{
		newComplianceReportSnapshot(report1, config1, 100*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_GENERATED, true),
		newComplianceReportSnapshot(report2, config1, 100*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),
	}
	blobs := []*storage.Blob{
		newBlob(common.GetComplianceReportBlobPath(config1, report1), blobData),
		newBlob(common.GetComplianceReportBlobPath(config1, report2), blobData),
	}
	expectedDeletions := set.NewStringSet()
	for _, snapshot := range snapshots {
		s.Require().NoError(s.snapshotDS.UpsertSnapshot(s.ctx, snapshot))
	}
	for _, blob := range blobs {
		s.Require().NoError(s.blobDS.Upsert(s.ctx, blob, bytes.NewBuffer(blobData)))
	}
	s.PruneAndAssert(snapshots, expectedDeletions)
}

func (s *ComplianceReportPruningSuite) Test_MustNotDeleteJobsInTheRetentionWindow() {
	// Mixed jobs; all in the retention window
	snapshots := []*storage.ComplianceOperatorReportSnapshotV2{
		newComplianceReportSnapshot(report1, config1, 1*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_WAITING, false),
		newComplianceReportSnapshot(report2, config2, 1*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_WAITING, true),
		newComplianceReportSnapshot(report3, config3, 1*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_PREPARING, false),
		newComplianceReportSnapshot(report4, config4, 1*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_PREPARING, true),

		newComplianceReportSnapshot(report5, config1, 2*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, false),
		newComplianceReportSnapshot(report6, config2, 2*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report7, config3, 2*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_FAILURE, false),
		newComplianceReportSnapshot(report8, config4, 2*day, storage.ComplianceOperatorReportStatus_DOWNLOAD, storage.ComplianceOperatorReportStatus_FAILURE, true),

		newComplianceReportSnapshot(report9, config1, 3*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_WAITING, true),
		newComplianceReportSnapshot(report10, config2, 3*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_PREPARING, true),
		newComplianceReportSnapshot(report11, config3, 3*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_DELIVERED, true),
		newComplianceReportSnapshot(report12, config4, 3*day, storage.ComplianceOperatorReportStatus_EMAIL, storage.ComplianceOperatorReportStatus_FAILURE, true),
	}
	expectedDeletions := set.NewStringSet()
	for _, snapshot := range snapshots {
		s.Require().NoError(s.snapshotDS.UpsertSnapshot(s.ctx, snapshot))
	}
	s.PruneAndAssert(snapshots, expectedDeletions)
}

func (s *ComplianceReportPruningSuite) PruneAndAssert(snapshots []*storage.ComplianceOperatorReportSnapshotV2, expectedDeletions set.StringSet) {
	PruneComplianceReportHistory(s.ctx, s.db, retentionDuration)
	for _, snapshot := range snapshots {
		_, found, err := s.snapshotDS.GetSnapshot(s.ctx, snapshot.GetReportId())
		s.Require().NoError(err)
		s.Assert().Equal(expectedDeletions.Contains(snapshot.GetReportId()), !found, "report %s should be deleted == %t but found == %t", idToHumanReadable[snapshot.GetReportId()], expectedDeletions.Contains(snapshot.GetReportId()), found)
	}
}

func newComplianceReportSnapshot(
	id string,
	scanConfigID string,
	completedAt time.Duration,
	notificationMethod storage.ComplianceOperatorReportStatus_NotificationMethod,
	state storage.ComplianceOperatorReportStatus_RunState,
	onDemand bool,
) *storage.ComplianceOperatorReportSnapshotV2 {
	requestType := storage.ComplianceOperatorReportStatus_SCHEDULED
	if onDemand || notificationMethod == storage.ComplianceOperatorReportStatus_DOWNLOAD {
		requestType = storage.ComplianceOperatorReportStatus_ON_DEMAND
	}
	return &storage.ComplianceOperatorReportSnapshotV2{
		ReportId:            id,
		ScanConfigurationId: scanConfigID,
		ReportStatus: &storage.ComplianceOperatorReportStatus{
			RunState:                 state,
			ReportRequestType:        requestType,
			CompletedAt:              protoconv.NowMinus(completedAt),
			ReportNotificationMethod: notificationMethod,
		},
	}
}
