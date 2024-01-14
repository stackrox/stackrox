//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	checkresultsSearch "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/search"
	checkResultsStorage "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/store/postgres"
	apiV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	maxPaginationLimit = 1000
)

var (
	expectedClusterCounts = []*ResourceResultCountByClusterScan{
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          0,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  1,
			ClusterID:          testconsts.Cluster1,
			ClusterName:        "cluster1",
			ScanConfigName:     "scanConfig1",
		},
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          0,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  3,
			ClusterID:          testconsts.Cluster2,
			ClusterName:        "cluster2",
			ScanConfigName:     "scanConfig1",
		},
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          0,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  1,
			ClusterID:          testconsts.Cluster3,
			ClusterName:        "cluster3",
			ScanConfigName:     "scanConfig1",
		},
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          1,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  0,
			ClusterID:          testconsts.Cluster3,
			ClusterName:        "cluster3",
			ScanConfigName:     "scanConfig2",
		},
	}

	expectedCluster2And3Counts = []*ResourceResultCountByClusterScan{
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          0,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  3,
			ClusterID:          testconsts.Cluster2,
			ClusterName:        "cluster2",
			ScanConfigName:     "scanConfig1",
		},
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          0,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  1,
			ClusterID:          testconsts.Cluster3,
			ClusterName:        "cluster3",
			ScanConfigName:     "scanConfig1",
		},
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          1,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  0,
			ClusterID:          testconsts.Cluster3,
			ClusterName:        "cluster3",
			ScanConfigName:     "scanConfig2",
		},
	}

	expectedCluster2OnlyCounts = []*ResourceResultCountByClusterScan{
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          0,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  3,
			ClusterID:          testconsts.Cluster2,
			ClusterName:        "cluster2",
			ScanConfigName:     "scanConfig1",
		},
	}
)

func TestComplianceCheckResultDataStore(t *testing.T) {
	suite.Run(t, new(complianceCheckResultDataStoreTestSuite))
}

type complianceCheckResultDataStoreTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	hasReadCtx   context.Context
	hasWriteCtx  context.Context
	noAccessCtx  context.Context
	testContexts map[string]context.Context

	dataStore DataStore
	storage   checkResultsStorage.Store
	db        *pgtest.TestPostgres
	searcher  checkresultsSearch.Searcher
}

func (s *complianceCheckResultDataStoreTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip tests when ComplianceEnhancements disabled")
		s.T().SkipNow()
	}
}

func (s *complianceCheckResultDataStoreTestSuite) SetupTest() {
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.noAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Compliance)

	s.mockCtrl = gomock.NewController(s.T())

	s.db = pgtest.ForT(s.T())

	s.storage = checkResultsStorage.New(s.db)
	indexer := checkResultsStorage.NewIndexer(s.db)
	configStorage := checkResultsStorage.New(s.db)
	s.searcher = checkresultsSearch.New(configStorage, indexer)
	s.dataStore = New(s.storage, s.db, s.searcher)
}

func (s *complianceCheckResultDataStoreTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *complianceCheckResultDataStoreTestSuite) TestUpsertResult() {
	// make sure we have nothing
	checkResultIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(checkResultIDs)

	rec1 := getTestRec(testconsts.Cluster1)
	rec2 := getTestRec(testconsts.Cluster2)
	ids := []string{rec1.GetId(), rec2.GetId()}

	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec1))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec2))

	count, err := s.storage.Count(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Equal(len(ids), count)

	// upsert with read context
	s.Require().Error(s.dataStore.UpsertResult(s.hasReadCtx, rec2))

	retrieveRec1, found, err := s.storage.Get(s.hasReadCtx, rec1.GetId())
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(rec1, retrieveRec1)
}

func (s *complianceCheckResultDataStoreTestSuite) TestDeleteResult() {
	// make sure we have nothing
	checkResultIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(checkResultIDs)

	rec1 := getTestRec(testconsts.Cluster1)
	rec2 := getTestRec(testconsts.Cluster2)
	ids := []string{rec1.GetId(), rec2.GetId()}

	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec1))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec2))

	count, err := s.storage.Count(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Equal(len(ids), count)

	// Try to delete with wrong context
	s.Require().Error(s.dataStore.DeleteResult(s.hasReadCtx, rec1.GetId()))

	// Now delete rec1
	s.Require().NoError(s.dataStore.DeleteResult(s.hasWriteCtx, rec1.GetId()))
	retrieveRec1, found, err := s.storage.Get(s.hasReadCtx, rec1.GetId())
	s.Require().NoError(err)
	s.Require().False(found)
	s.Require().Nil(retrieveRec1)
}

func (s *complianceCheckResultDataStoreTestSuite) TestSearchCheckResults() {
	s.setupTestData()

	count, err := s.storage.Count(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Equal(6, count)

	// Search results of fake cluster, should return 0
	searchResults, err := s.dataStore.SearchComplianceCheckResults(s.hasReadCtx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, fixtureconsts.ClusterFake2).ProtoQuery())
	s.Require().NoError(err)
	s.Require().Equal(0, len(searchResults))

	// Search results of cluster 2 should return 3 records.
	searchResults, err = s.dataStore.SearchComplianceCheckResults(s.hasReadCtx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, testconsts.Cluster2).ProtoQuery())
	s.Require().NoError(err)
	s.Require().Equal(3, len(searchResults))

	// Search with no access should return err
	searchResults, err = s.dataStore.SearchComplianceCheckResults(s.noAccessCtx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, fixtureconsts.Cluster2).ProtoQuery())
	s.Require().NoError(err)
	s.Require().Equal(0, len(searchResults))
}

func (s *complianceCheckResultDataStoreTestSuite) TestCheckResultStats() {
	s.setupTestData()

	// Counts by Scan Config by Cluster
	query := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, testconsts.Cluster2).
		AddExactMatches(search.ClusterID, testconsts.Cluster3).ProtoQuery()

	results, err := s.dataStore.ComplianceCheckResultStats(s.hasReadCtx, query)
	s.Require().NoError(err)
	s.Require().Equal(expectedCluster2And3Counts, results)

	// Counts with no access should return error
	results, err = s.dataStore.ComplianceCheckResultStats(s.noAccessCtx, query)
	s.Require().NoError(err)
	s.Require().Equal(0, len(results))
}

func (s *complianceCheckResultDataStoreTestSuite) TestCountScanResults() {
	s.setupTestData()
	q := search.NewQueryBuilder().ProtoQuery()
	count, err := s.dataStore.CountCheckResults(s.hasReadCtx, q)
	s.NoError(err)
	s.Equal(6, count)
}

func (s *complianceCheckResultDataStoreTestSuite) TestSearchResultsSac() {
	s.setupTestData()
	testCases := []struct {
		desc          string
		query         *apiV1.Query
		scopeKey      string
		expectedCount int
	}{
		{
			desc:          "Empty query - Full access",
			query:         search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			scopeKey:      testutils.UnrestrictedReadCtx,
			expectedCount: 6,
		},
		{
			desc:          "Empty query - Only cluster 2 access",
			query:         search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedCount: 3,
		},
		{
			desc: "Cluster 2 query - Only cluster 2 access",
			query: search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).
				WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedCount: 3,
		},
		{
			desc: "Cluster 1 and 2 query - Only cluster 2 access",
			query: search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).
				AddStrings(search.ClusterID, testconsts.Cluster1).
				WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedCount: 3,
		},
	}

	for _, tc := range testCases {
		results, err := s.dataStore.SearchComplianceCheckResults(s.testContexts[tc.scopeKey], tc.query)
		s.NoError(err)
		s.Equal(tc.expectedCount, len(results))
	}
}

func (s *complianceCheckResultDataStoreTestSuite) TestCountResultsSac() {
	s.setupTestData()
	testCases := []struct {
		desc          string
		query         *apiV1.Query
		scopeKey      string
		expectedCount int
	}{
		{
			desc:          "Empty query - Full access",
			query:         search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			scopeKey:      testutils.UnrestrictedReadCtx,
			expectedCount: 6,
		},
		{
			desc:          "Empty query - Only cluster 2 access",
			query:         search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedCount: 3,
		},
		{
			desc: "Cluster 2 query - Only cluster 2 access",
			query: search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).
				WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedCount: 3,
		},
		{
			desc: "Cluster 1 and 2 query - Only cluster 2 access",
			query: search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).
				AddStrings(search.ClusterID, testconsts.Cluster1).
				WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedCount: 3,
		},
	}

	for _, tc := range testCases {
		count, err := s.dataStore.CountCheckResults(s.testContexts[tc.scopeKey], tc.query)
		s.NoError(err)
		s.Equal(tc.expectedCount, count)
	}
}

func (s *complianceCheckResultDataStoreTestSuite) TestResultsStatsSac() {
	s.setupTestData()
	testCases := []struct {
		desc            string
		query           *apiV1.Query
		scopeKey        string
		expectedResults []*ResourceResultCountByClusterScan
	}{
		{
			desc:            "Empty query - Full access",
			query:           search.NewQueryBuilder().ProtoQuery(),
			scopeKey:        testutils.UnrestrictedReadCtx,
			expectedResults: expectedClusterCounts,
		},
		{
			desc:            "Empty query - Only cluster 2 access",
			query:           search.NewQueryBuilder().ProtoQuery(),
			scopeKey:        testutils.Cluster2ReadWriteCtx,
			expectedResults: expectedCluster2OnlyCounts,
		},
		{
			desc:            "Cluster 2 query - Only cluster 2 access",
			query:           search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).ProtoQuery(),
			scopeKey:        testutils.Cluster2ReadWriteCtx,
			expectedResults: expectedCluster2OnlyCounts,
		},
		{
			desc: "Cluster 2 and 3 query - Only cluster 2 access",
			query: search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).
				AddStrings(search.ClusterID, testconsts.Cluster3).ProtoQuery(),
			scopeKey:        testutils.Cluster2ReadWriteCtx,
			expectedResults: expectedCluster2OnlyCounts,
		},
		{
			desc: "Cluster 2 and 3 query - Full Access",
			query: search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).
				AddStrings(search.ClusterID, testconsts.Cluster3).ProtoQuery(),
			scopeKey:        testutils.UnrestrictedReadCtx,
			expectedResults: expectedCluster2And3Counts,
		},
	}

	for _, tc := range testCases {
		results, err := s.dataStore.ComplianceCheckResultStats(s.testContexts[tc.scopeKey], tc.query)
		s.NoError(err)
		s.Equal(tc.expectedResults, results)
	}
}

func (s *complianceCheckResultDataStoreTestSuite) setupTestData() {
	// make sure we have nothing
	checkResultIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(checkResultIDs)

	_, err = s.db.DB.Exec(context.Background(), "insert into compliance_operator_scan_configuration_v2 (id, scanconfigname) values ($1, $2)", fixtureconsts.ComplianceScanConfigID1, "scan 1")
	s.Require().NoError(err)
	_, err = s.db.DB.Exec(context.Background(), "insert into compliance_operator_scan_configuration_v2 (id, scanconfigname) values ($1, $2)", fixtureconsts.ComplianceScanConfigID2, "scan 2")
	s.Require().NoError(err)

	_, err = s.db.DB.Exec(context.Background(), "insert into clusters (id, name) values ($1, $2)", testconsts.Cluster1, "cluster1")
	s.Require().NoError(err)
	_, err = s.db.DB.Exec(context.Background(), "insert into clusters (id, name) values ($1, $2)", testconsts.Cluster2, "cluster2")
	s.Require().NoError(err)
	_, err = s.db.DB.Exec(context.Background(), "insert into clusters (id, name) values ($1, $2)", testconsts.Cluster3, "cluster3")
	s.Require().NoError(err)

	rec1 := getTestRec(testconsts.Cluster1)
	rec2 := getTestRec(testconsts.Cluster2)
	rec3 := getTestRec(testconsts.Cluster2)
	rec4 := getTestRec(testconsts.Cluster2)
	rec5 := getTestRec(testconsts.Cluster3)
	rec6 := getTestRec2(testconsts.Cluster3)

	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec1))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec2))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec3))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec4))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec5))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec6))
}

func getTestRec(clusterID string) *storage.ComplianceOperatorCheckResultV2 {
	return &storage.ComplianceOperatorCheckResultV2{
		Id:             uuid.NewV4().String(),
		CheckId:        uuid.NewV4().String(),
		CheckName:      "test-check",
		ClusterId:      clusterID,
		Status:         storage.ComplianceOperatorCheckResultV2_INCONSISTENT,
		Severity:       storage.RuleSeverity_HIGH_RULE_SEVERITY,
		Description:    "this is a test",
		Instructions:   "this is a test",
		Labels:         nil,
		Annotations:    nil,
		CreatedTime:    types.TimestampNow(),
		ScanName:       uuid.NewV4().String(),
		ScanConfigName: "scanConfig1",
	}
}

func getTestRec2(clusterID string) *storage.ComplianceOperatorCheckResultV2 {
	return &storage.ComplianceOperatorCheckResultV2{
		Id:             uuid.NewV4().String(),
		CheckId:        uuid.NewV4().String(),
		CheckName:      "test-check-2",
		ClusterId:      clusterID,
		Status:         storage.ComplianceOperatorCheckResultV2_INFO,
		Severity:       storage.RuleSeverity_INFO_RULE_SEVERITY,
		Description:    "this is a test",
		Instructions:   "this is a test",
		Labels:         nil,
		Annotations:    nil,
		CreatedTime:    types.TimestampNow(),
		ScanName:       uuid.NewV4().String(),
		ScanConfigName: "scanConfig2",
	}
}
