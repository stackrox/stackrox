//go:build sql_integration

package postgres

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	blobDatastore "github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/central/reports/common"
	configDatastore "github.com/stackrox/rox/central/reports/config/datastore"
	historyDatastore "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

const (
	retentionDuration = time.Hour * 24 * 7
)

var (
	config1 = uuid.NewV4().String()
	config2 = uuid.NewV4().String()
	config3 = uuid.NewV4().String()
	config4 = uuid.NewV4().String()
)

type ReportHistoryPruningSuite struct {
	suite.Suite
	ctx       context.Context
	testDB    *pgtest.TestPostgres
	historyDS historyDatastore.DataStore
	configDS  configDatastore.DataStore
	blobDS    blobDatastore.Datastore
}

func TestReportHistoryPruning(t *testing.T) {
	suite.Run(t, new(ReportHistoryPruningSuite))
}

func (s *ReportHistoryPruningSuite) SetupSuite() {
	s.T().Setenv(features.VulnReportingEnhancements.EnvVar(), "true")
	if !features.VulnReportingEnhancements.Enabled() {
		s.T().Skip("Skip test when reporting enhancements are disabled")
		s.T().SkipNow()
	}

	s.testDB = pgtest.ForT(s.T())
	s.ctx = sac.WithAllAccess(context.Background())

	configDS := configDatastore.GetTestPostgresDataStore(s.T(), s.testDB)
	s.configDS = configDS
	s.historyDS = historyDatastore.GetTestPostgresDataStore(s.T(), s.testDB)
	s.blobDS = blobDatastore.NewTestDatastore(s.T(), s.testDB)

	for _, id := range []string{config1, config2, config3, config4} {
		_, err := s.configDS.AddReportConfiguration(s.ctx, newConfig(id))
		s.Require().NoError(err)
	}
}

func (s *ReportHistoryPruningSuite) TearDownTest() {
	tag, err := s.testDB.Exec(s.ctx, "TRUNCATE report_snapshots CASCADE")
	s.T().Log("report_snapshots", tag)
	s.NoError(err)

	tag, err = s.testDB.Exec(s.ctx, "TRUNCATE blobs CASCADE")
	s.T().Log("blobs", tag)
	s.NoError(err)
}

func (s *ReportHistoryPruningSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *ReportHistoryPruningSuite) TestMustNotDeleteLastSuccessfulJobOneDownload() {
	// All downloads; all delivered; outside retention window
	snapshots := []*storage.ReportSnapshot{
		newReportSnapshot(config1, "r1", 100*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config2, "r2", 100*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config3, "r3", 100*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config4, "r4", 100*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_DELIVERED),
	}
	fakeIDToActualID := s.prepareDataStores(snapshots, nil)
	s.pruneAndAssert(fakeIDToActualID, set.NewStringSet())
}

func (s *ReportHistoryPruningSuite) TestMustNotDeleteLastSuccessfulJobOneEmail() {
	// All emails; all delivered; outside retention window
	snapshots := []*storage.ReportSnapshot{
		newReportSnapshot(config1, "r1", 100*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config2, "r2", 100*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config3, "r3", 100*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config4, "r4", 100*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_DELIVERED),
	}
	fakeIDToActualID := s.prepareDataStores(snapshots, nil)
	s.pruneAndAssert(fakeIDToActualID, set.NewStringSet())
}

func (s *ReportHistoryPruningSuite) TestMustNotDeleteLastSuccessfulJobMixedMethod() {
	// Newer delivered downloads; older failed scheduled; outside retention window
	snapshots := []*storage.ReportSnapshot{
		newReportSnapshot(config1, "r1", 100*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_FAILURE),
		newReportSnapshot(config2, "r2", 100*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_FAILURE),
		newReportSnapshot(config3, "r3", 100*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_FAILURE),
		newReportSnapshot(config4, "r4", 100*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_FAILURE),

		newReportSnapshot(config1, "r5", 90*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config2, "r6", 90*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config3, "r7", 90*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config4, "r8", 90*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_DELIVERED),
	}
	fakeIDToActualID := s.prepareDataStores(snapshots, nil)
	expectedDeletions := set.NewStringSet("r1", "r2", "r3", "r4")
	s.pruneAndAssert(fakeIDToActualID, expectedDeletions)
}

func (s *ReportHistoryPruningSuite) TestMustNotDeleteLastSuccessfulJobMixedMethod2() {
	// Older delivered downloads; newer failed scheduled; outside retention window
	snapshots := []*storage.ReportSnapshot{
		newReportSnapshot(config1, "r1", 100*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config2, "r2", 100*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config3, "r3", 100*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config4, "r4", 100*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_DELIVERED),

		newReportSnapshot(config1, "r5", 90*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_FAILURE),
		newReportSnapshot(config2, "r6", 90*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_FAILURE),
		newReportSnapshot(config3, "r7", 90*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_FAILURE),
		newReportSnapshot(config4, "r8", 90*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_FAILURE),
	}
	fakeIDToActualID := s.prepareDataStores(snapshots, nil)
	expectedDeletions := set.NewStringSet("r5", "r6", "r7", "r8")
	s.pruneAndAssert(fakeIDToActualID, expectedDeletions)
}

func (s *ReportHistoryPruningSuite) TestMustNotDeleteLastSuccessfulJobMixedMethod3() {
	// On-demand and scheduled; outside retention window
	snapshots := []*storage.ReportSnapshot{
		newReportSnapshot(config1, "r1", 100*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config2, "r2", 100*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_FAILURE),
		newReportSnapshot(config3, "r3", 100*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config4, "r4", 100*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_FAILURE),

		newReportSnapshot(config1, "r5", 90*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_FAILURE),
		newReportSnapshot(config2, "r6", 90*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config3, "r7", 90*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_FAILURE),
		newReportSnapshot(config4, "r8", 90*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_DELIVERED),
	}
	onDemandEmails := []*storage.ReportSnapshot{
		newReportSnapshot(config1, "r9", 80*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_FAILURE),
		newReportSnapshot(config2, "r10", 80*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config3, "r11", 80*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_FAILURE),
		newReportSnapshot(config4, "r12", 80*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_DELIVERED),
	}
	for _, onDemandEmail := range onDemandEmails {
		onDemandEmail.ReportStatus.ReportRequestType = storage.ReportStatus_ON_DEMAND
	}
	fakeIDToActualID := s.prepareDataStores(snapshots, nil)
	expectedDeletions := set.NewStringSet("r2", "r4", "r5", "r7", "r9", "r11")
	s.pruneAndAssert(fakeIDToActualID, expectedDeletions)
}

func (s *ReportHistoryPruningSuite) TestMustNotDeleteLastSuccessfulJobMultipleEmails() {
	// All emails; all delivered; multiple; outside retention window
	snapshots := []*storage.ReportSnapshot{
		newReportSnapshot(config1, "r1", 100*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config2, "r2", 100*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config3, "r3", 100*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config4, "r4", 100*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_DELIVERED),

		newReportSnapshot(config1, "r5", 90*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config2, "r6", 90*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config3, "r7", 90*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config4, "r8", 90*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_DELIVERED),
	}
	fakeIDToActualID := s.prepareDataStores(snapshots, nil)
	expectedDeletions := set.NewStringSet("r1", "r2", "r3", "r4")
	s.pruneAndAssert(fakeIDToActualID, expectedDeletions)
}

func (s *ReportHistoryPruningSuite) TestMustNotDeleteUnfinishedOldJobs() {
	// Older than retention window; preparing and waiting
	snapshots := []*storage.ReportSnapshot{
		newReportSnapshot(config1, "r1", 100*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_PREPARING),
		newReportSnapshot(config2, "r2", 100*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_WAITING),
		newReportSnapshot(config3, "r3", 100*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_PREPARING),
		newReportSnapshot(config4, "r4", 100*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_WAITING),
	}
	fakeIDToActualID := s.prepareDataStores(snapshots, nil)
	expectedDeletions := set.NewStringSet()
	s.pruneAndAssert(fakeIDToActualID, expectedDeletions)
}

func (s *ReportHistoryPruningSuite) TestMustNotDeleteUnfinishedJobsInRetentionWindow() {
	// Within retention window; preparing and waiting
	snapshots := []*storage.ReportSnapshot{
		newReportSnapshot(config1, "r1", 1*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_PREPARING),
		newReportSnapshot(config2, "r2", 2*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_WAITING),
		newReportSnapshot(config3, "r3", 3*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_PREPARING),
		newReportSnapshot(config4, "r41", 4*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_WAITING),
		newReportSnapshot(config1, "r42", 2*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_GENERATED),
	}
	fakeIDToActualID := s.prepareDataStores(snapshots, nil)
	expectedDeletions := set.NewStringSet()
	s.pruneAndAssert(fakeIDToActualID, expectedDeletions)
}

func (s *ReportHistoryPruningSuite) TestMustNotDeleteDeliveredJobsInRetentionWindow() {
	// Within retention window; delivered
	snapshots := []*storage.ReportSnapshot{
		newReportSnapshot(config1, "r1", 1*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config2, "r2", 2*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config3, "r3", 3*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config4, "r4", 4*24*time.Hour, storage.ReportStatus_EMAIL, storage.ReportStatus_DELIVERED),
	}
	fakeIDToActualID := s.prepareDataStores(snapshots, nil)
	expectedDeletions := set.NewStringSet()
	s.pruneAndAssert(fakeIDToActualID, expectedDeletions)
}

func (s *ReportHistoryPruningSuite) TestMustNotDeleteIfBlobsExist() {
	// All old downloads; blob exists
	snapshots := []*storage.ReportSnapshot{
		newReportSnapshot(config2, "r21", 32*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config2, "r22", 32*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config3, "r31", 64*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_DELIVERED),
		newReportSnapshot(config2, "r32", 32*24*time.Hour, storage.ReportStatus_DOWNLOAD, storage.ReportStatus_DELIVERED),
	}
	blobs := []*storage.Blob{
		newBlob(common.GetReportBlobPath(config2, "r21"), []byte("test-blob")),
		newBlob(common.GetReportBlobPath(config2, "r22"), []byte("test-blob")),
		newBlob(common.GetReportBlobPath(config3, "r31"), []byte("test-blob")),
		newBlob(common.GetReportBlobPath(config3, "r32"), []byte("test-blob")),
	}
	fakeIDToActualID := s.prepareDataStores(snapshots, blobs)
	expectedDeletions := set.NewStringSet("r5", "r6", "r7", "r8")
	s.pruneAndAssert(fakeIDToActualID, expectedDeletions)
}

func (s *ReportHistoryPruningSuite) prepareDataStores(snapshots []*storage.ReportSnapshot, blobs []*storage.Blob) map[string]string {
	fakeIDToActualID := make(map[string]string)
	for _, snapshot := range snapshots {
		fakeID := snapshot.GetReportId()
		snapshot.ReportId = ""
		actualID := s.storeSnapshots(snapshot)
		fakeIDToActualID[fakeID] = actualID
	}

	for _, blob := range blobs {
		data := bytes.NewBuffer([]byte("test-blob"))
		s.Nil(s.blobDS.Upsert(s.ctx, blob, data))
	}

	return fakeIDToActualID
}

func (s *ReportHistoryPruningSuite) storeSnapshots(snapshot *storage.ReportSnapshot) string {
	reportID, err := s.historyDS.AddReportSnapshot(s.ctx, snapshot)
	s.Require().Nil(err)
	return reportID
}

func (s *ReportHistoryPruningSuite) pruneAndAssert(fakeIDToActualID map[string]string, expectedDeletions set.StringSet) {
	PruneReportHistory(s.ctx, s.testDB, retentionDuration)

	for fakeID, actualID := range fakeIDToActualID {
		found, err := s.historyDS.Exists(s.ctx, actualID)
		s.Nil(err)
		if expectedDeletions.Contains(fakeID) {
			s.False(found)
		} else {
			s.True(found)
		}
	}
}

// region  helper functions

func newConfig(id string) *storage.ReportConfiguration {
	return &storage.ReportConfiguration{
		Id:   id,
		Name: id,
	}
}

func newReportSnapshot(
	configID string,
	reportID string,
	age time.Duration,
	notificationMethod storage.ReportStatus_NotificationMethod,
	runState storage.ReportStatus_RunState,
) *storage.ReportSnapshot {
	var runMethod storage.ReportStatus_RunMethod
	if notificationMethod == storage.ReportStatus_EMAIL {
		runMethod = storage.ReportStatus_SCHEDULED
	} else {
		runMethod = storage.ReportStatus_ON_DEMAND
	}
	return &storage.ReportSnapshot{
		ReportId:              reportID,
		ReportConfigurationId: configID,
		ReportStatus: &storage.ReportStatus{
			RunState:                 runState,
			ReportRequestType:        runMethod,
			CompletedAt:              timestampNowMinus(age),
			ReportNotificationMethod: notificationMethod,
		},
	}
}

func newBlob(name string, data []byte) *storage.Blob {
	return &storage.Blob{
		Name:         name,
		ModifiedTime: types.TimestampNow(),
		Length:       int64(len(data)),
	}
}

// endregion  helper functions
