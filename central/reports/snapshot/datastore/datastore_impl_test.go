//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
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
	s.testDB = pgtest.ForT(s.T())
	s.datastore = GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.reportConfigStore = reportConfigDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.notifierDataStore = notifierDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)

	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration)))
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
	resultSnap, found, err := s.datastore.Get(noAccessCtx, snap.GetReportId())
	s.NoError(err)
	s.False(found)
	s.Nil(resultSnap)

	// Test Get: returns report with read access
	resultSnap, found, err = s.datastore.Get(s.ctx, snap.GetReportId())
	s.NoError(err)
	s.True(found)
	s.Equal(snap.GetReportId(), resultSnap.GetReportId())

	// Test Search: Without read access
	results, err := s.datastore.Search(noAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Nil(results)

	// Test Search: With read access
	results, err = s.datastore.Search(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(1, len(results))
	s.Equal(snap.GetReportId(), results[0].ID)

	// Test Search: Search by run state
	failedReportSnap := fixtures.GetReportSnapshot()
	failedReportSnap.ReportStatus.RunState = storage.ReportStatus_FAILURE
	failedReportSnap.ReportConfigurationId = configID
	failedReportSnap.ReportId, err = s.datastore.AddReportSnapshot(s.ctx, failedReportSnap)
	s.NoError(err)

	results, err = s.datastore.Search(s.ctx, search.MatchFieldQuery(search.ReportState.String(), storage.ReportStatus_FAILURE.String(), false))
	s.NoError(err)
	s.Equal(1, len(results))
	s.Equal(failedReportSnap.GetReportId(), results[0].ID)

	// Test Count: returns 0 without read access
	count, err := s.datastore.Count(noAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(0, count)

	// Test Count: return true count with read access
	count, err = s.datastore.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(2, count)

	// Test Exists: returns false without read access
	exists, err := s.datastore.Exists(noAccessCtx, snap.GetReportId())
	s.NoError(err)
	s.False(exists)

	// Test Exists: returns correct value with read access
	exists, err = s.datastore.Exists(s.ctx, snap.GetReportId())
	s.NoError(err)
	s.True(exists)

	// Test GetMany: returns no reports without read access
	reportIDs := []string{snap.GetReportId(), failedReportSnap.GetReportId()}
	snaps, err := s.datastore.GetMany(noAccessCtx, reportIDs)
	s.NoError(err)
	s.Nil(snaps)

	// Test GetMany: returns requested reports with read access
	snaps, err = s.datastore.GetMany(s.ctx, reportIDs)
	s.NoError(err)
	s.Equal(len(reportIDs), len(snaps))

	// Test DeleteReportSnapshot: returns error without write access
	err = s.datastore.DeleteReportSnapshot(noAccessCtx, snap.GetReportId())
	s.Error(err)

	// Test DeleteReportSnapshot: successfully deletes with write access
	err = s.datastore.DeleteReportSnapshot(s.ctx, snap.GetReportId())
	s.NoError(err)
	resultSnap, found, err = s.datastore.Get(s.ctx, snap.GetReportId())
	s.NoError(err)
	s.False(found)
	s.Nil(resultSnap)

	// Test AddReportSnapshot: error with invalid config ID
	snap = fixtures.GetReportSnapshot()
	snap.ReportConfigurationId = uuid.NewDummy().String()
	_, err = s.datastore.AddReportSnapshot(s.ctx, snap)
	s.Error(err)

	// Test AddReportSnapshot: success with NO config ID
	snap = fixtures.GetReportSnapshot()
	snap.ReportConfigurationId = ""
	snap.ReportId, err = s.datastore.AddReportSnapshot(s.ctx, snap)
	s.NoError(err)

	// Test Get: returns report with read access
	resultSnap, found, err = s.datastore.Get(s.ctx, snap.GetReportId())
	s.NoError(err)
	s.True(found)
	s.Equal(snap.GetReportId(), resultSnap.GetReportId())
}

func (s *ReportSnapshotDatastoreTestSuite) storeNotifier(name string) *storage.NotifierConfiguration_Id {
	allCtx := sac.WithAllAccess(context.Background())

	id, err := s.notifierDataStore.AddNotifier(allCtx, &storage.Notifier{Name: name})
	s.Require().NoError(err)
	return &storage.NotifierConfiguration_Id{Id: id}
}

func (s *ReportSnapshotDatastoreTestSuite) cleanupReportSnapshots(ctx context.Context) {
	results, err := s.datastore.SearchResults(ctx, search.EmptyQuery())
	s.NoError(err)
	for i := range results {
		s.NoError(s.datastore.DeleteReportSnapshot(ctx, results[i].GetId()))
	}
	// Verify cleanup
	results, err = s.datastore.SearchResults(ctx, search.EmptyQuery())
	s.NoError(err)
	s.Empty(results)

}

func (s *ReportSnapshotDatastoreTestSuite) TestSearchResults() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration)))

	s.cleanupReportSnapshots(ctx)

	// Create test report snapshots
	snapshot1 := fixtures.GetReportSnapshot()
	snapshot1.ReportConfigurationId = ""
	id1, err := s.datastore.AddReportSnapshot(ctx, snapshot1)
	s.NoError(err)
	s.NotEmpty(id1)

	snapshot2 := fixtures.GetReportSnapshot()
	snapshot2.ReportConfigurationId = ""
	snapshot2.ReportStatus.RunState = storage.ReportStatus_FAILURE
	id2, err := s.datastore.AddReportSnapshot(ctx, snapshot2)
	s.NoError(err)
	s.NotEmpty(id2)

	snapshot3 := fixtures.GetReportSnapshot()
	snapshot3.ReportConfigurationId = ""
	snapshot3.ReportStatus.RunState = storage.ReportStatus_DELIVERED
	id3, err := s.datastore.AddReportSnapshot(ctx, snapshot3)
	s.NoError(err)
	s.NotEmpty(id3)

	// Define test cases
	testCases := []struct {
		name          string
		query         *v1.Query
		expectedCount int
		expectedIDs   []string
	}{
		{
			name:          "empty query returns all snapshots with names populated",
			query:         search.EmptyQuery(),
			expectedCount: 3,
			expectedIDs:   []string{id1, id2, id3},
		},
		{
			name:          "nil query defaults to empty query",
			query:         nil,
			expectedCount: 3,
			expectedIDs:   []string{id1, id2, id3},
		},
		{
			name:          "query by run state - FAILURE",
			query:         search.MatchFieldQuery(search.ReportState.String(), storage.ReportStatus_FAILURE.String(), false),
			expectedCount: 1,
			expectedIDs:   []string{id2},
		},
		{
			name:          "query by run state - DELIVERED",
			query:         search.MatchFieldQuery(search.ReportState.String(), storage.ReportStatus_DELIVERED.String(), false),
			expectedCount: 1,
			expectedIDs:   []string{id3},
		},
		{
			name:          "query with no matches returns empty",
			query:         search.NewQueryBuilder().AddExactMatches(search.ReportName, "nonexistent-report").ProtoQuery(),
			expectedCount: 0,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			results, err := s.datastore.SearchResults(ctx, tc.query)
			s.NoError(err)

			actualIDs := make([]string, 0, len(results))
			for _, result := range results {
				actualIDs = append(actualIDs, result.GetId())
				// Verify name is populated (should equal ID for report snapshots)
				s.NotEmpty(result.GetId())
				s.Equal(result.GetId(), result.GetName(), "Name should equal ID for report snapshots")
				s.Equal(v1.SearchCategory_REPORT_SNAPSHOT, result.GetCategory())
				s.Empty(result.GetLocation(), "Location should be empty for report snapshots")
			}

			if len(tc.expectedIDs) > 0 {
				s.ElementsMatch(tc.expectedIDs, actualIDs)
			}
		})
	}

	// Clean up
	s.cleanupReportSnapshots(ctx)
}
