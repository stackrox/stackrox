//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestReportSnapshotDatastore(t *testing.T) {
	suite.Run(t, new(ReportSnapshotDatastoreTestSuite))
}

type ReportSnapshotDatastoreTestSuite struct {
	suite.Suite

	testDB            *pgtest.TestPostgres
	datastore         DataStore
	reportConfigStore reportConfigDS.DataStore
	notifierDataStore notifierDS.DataStore
	ctx               context.Context
}

func (s *ReportSnapshotDatastoreTestSuite) SetupSuite() {
	s.T().Setenv(features.VulnReportingEnhancements.EnvVar(), "true")
	if !features.VulnReportingEnhancements.Enabled() {
		s.T().Skip("Skip tests when ROX_VULN_MGMT_REPORTING_ENHANCEMENTS disabled")
		s.T().SkipNow()
	}

	s.testDB = pgtest.ForT(s.T())
	s.datastore = GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.reportConfigStore = reportConfigDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.notifierDataStore = notifierDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)

	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration)))
}

func (s *ReportSnapshotDatastoreTestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *ReportSnapshotDatastoreTestSuite) TestReportMetadataWorkflows() {
	reportConfig := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
	reportConfig.Id = ""
	// Add all required notifiers to the database.
	for i, n := range reportConfig.GetNotifiers() {
		reportConfig.Notifiers[i].Ref = s.storeNotifier(n.GetId())
	}

	configID, err := s.reportConfigStore.AddReportConfiguration(s.ctx, reportConfig)
	s.NoError(err)

	noAccessCtx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())

	// Test AddReportSnapshot: error without write access
	snap := fixtures.GetReportSnapshot()
	snap.ReportConfigurationId = configID
	_, err = s.datastore.AddReportSnapshot(noAccessCtx, snap)
	s.Error(err)

	// Test AddReportSnapshot: no error with write access
	snap.ReportId, err = s.datastore.AddReportSnapshot(s.ctx, snap)
	s.NoError(err)

	// Test UpdateReportSnapshot: no error with write access
	snap.ReportStatus.RunState = storage.ReportStatus_DELIVERED
	err = s.datastore.UpdateReportSnapshot(s.ctx, snap)
	s.NoError(err)

	// Test Get: no result without read access
	resultSnap, found, err := s.datastore.Get(noAccessCtx, snap.ReportId)
	s.NoError(err)
	s.False(found)
	s.Nil(resultSnap)

	// Test Get: returns report with read access
	resultSnap, found, err = s.datastore.Get(s.ctx, snap.ReportId)
	s.NoError(err)
	s.True(found)
	s.Equal(snap.ReportId, resultSnap.ReportId)

	// Test Search: Without read access
	results, err := s.datastore.Search(noAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Nil(results)

	// Test Search: With read access
	results, err = s.datastore.Search(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(1, len(results))
	s.Equal(snap.ReportId, results[0].ID)

	// Test Search: Search by run state
	failedReportSnap := fixtures.GetReportSnapshot()
	failedReportSnap.ReportStatus.RunState = storage.ReportStatus_FAILURE
	failedReportSnap.ReportConfigurationId = configID
	failedReportSnap.ReportId, err = s.datastore.AddReportSnapshot(s.ctx, failedReportSnap)
	s.NoError(err)

	results, err = s.datastore.Search(s.ctx, search.MatchFieldQuery(search.ReportState.String(), storage.ReportStatus_FAILURE.String(), false))
	s.NoError(err)
	s.Equal(1, len(results))
	s.Equal(failedReportSnap.ReportId, results[0].ID)

	// Test Count: returns 0 without read access
	count, err := s.datastore.Count(noAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(0, count)

	// Test Count: return true count with read access
	count, err = s.datastore.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(2, count)

	// Test Exists: returns false without read access
	exists, err := s.datastore.Exists(noAccessCtx, snap.ReportId)
	s.NoError(err)
	s.False(exists)

	// Test Exists: returns correct value with read access
	exists, err = s.datastore.Exists(s.ctx, snap.ReportId)
	s.NoError(err)
	s.True(exists)

	// Test GetMany: returns no reports without read access
	reportIDs := []string{snap.ReportId, failedReportSnap.ReportId}
	snaps, err := s.datastore.GetMany(noAccessCtx, reportIDs)
	s.NoError(err)
	s.Nil(snaps)

	// Test GetMany: returns requested reports with read access
	snaps, err = s.datastore.GetMany(s.ctx, reportIDs)
	s.NoError(err)
	s.Equal(len(reportIDs), len(snaps))

	// Test DeleteReportSnapshot: returns error without write access
	err = s.datastore.DeleteReportSnapshot(noAccessCtx, snap.ReportId)
	s.Error(err)

	// Test DeleteReportSnapshot: successfully deletes with write access
	err = s.datastore.DeleteReportSnapshot(s.ctx, snap.ReportId)
	s.NoError(err)
	resultSnap, found, err = s.datastore.Get(s.ctx, snap.ReportId)
	s.NoError(err)
	s.False(found)
	s.Nil(resultSnap)
}

func (s *ReportSnapshotDatastoreTestSuite) storeNotifier(name string) *storage.NotifierConfiguration_Id {
	allCtx := sac.WithAllAccess(context.Background())

	id, err := s.notifierDataStore.AddNotifier(allCtx, &storage.Notifier{Name: name})
	s.Require().NoError(err)
	return &storage.NotifierConfiguration_Id{Id: id}
}
