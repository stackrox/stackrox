//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	checkresultsSearch "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/search"
	checkResultsStorage "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/store/postgres"
	"github.com/stackrox/rox/central/convert/internaltov2storage"
	apiV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
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
	expectedClusterScanCounts = []*ResourceResultCountByClusterScan{
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

	expectedInconsistentClusterScanCounts = []*ResourceResultCountByClusterScan{
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
			InfoCount:          1,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  0,
			ClusterID:          testconsts.Cluster3,
			ClusterName:        "cluster3",
			ScanConfigName:     "scanConfig2",
		},
	}

	expectedCluster2And3ScanCounts = []*ResourceResultCountByClusterScan{
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

	expectedCluster2OnlyScanCounts = []*ResourceResultCountByClusterScan{
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

	expectedClusterCounts = []*ResultStatusCountByCluster{
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
		},
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          1,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  1,
			ClusterID:          testconsts.Cluster3,
			ClusterName:        "cluster3",
		},
	}

	expectedClusterCountsByInconsistent = []*ResultStatusCountByCluster{
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          1,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  1,
			ClusterID:          testconsts.Cluster3,
			ClusterName:        "cluster3",
		},
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
		},
	}

	expectedCluster2And3Counts = []*ResultStatusCountByCluster{
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
		},
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          1,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  1,
			ClusterID:          testconsts.Cluster3,
			ClusterName:        "cluster3",
		},
	}

	expectedCluster2OnlyCounts = []*ResultStatusCountByCluster{
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
		},
	}

	expectedProfileCounts = []*ResourceResultCountByProfile{
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          1,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  5,
			ProfileName:        "ocp4-cis-node",
		},
	}

	expectedProfileCountsCluster2 = []*ResourceResultCountByProfile{
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          0,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  3,
			ProfileName:        "ocp4-cis-node",
		},
	}

	expectedProfileCountsCluster2And3 = []*ResourceResultCountByProfile{
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          1,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  4,
			ProfileName:        "ocp4-cis-node",
		},
	}

	expectedProfileResultsByInconsistent = []*ResourceResultsByProfile{
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          1,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  0,
			ProfileName:        "ocp4-cis-node",
			CheckName:          "test-check-2",
			RuleName:           "test-rule",
		},
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          0,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  5,
			ProfileName:        "ocp4-cis-node",
			CheckName:          "test-check",
			RuleName:           "test-rule",
		},
	}

	expectedProfileResults = []*ResourceResultsByProfile{
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          0,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  5,
			ProfileName:        "ocp4-cis-node",
			CheckName:          "test-check",
			RuleName:           "test-rule",
		},
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          1,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  0,
			ProfileName:        "ocp4-cis-node",
			CheckName:          "test-check-2",
			RuleName:           "test-rule",
		},
	}

	expectedProfileResultsCluster2 = []*ResourceResultsByProfile{
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          0,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  3,
			ProfileName:        "ocp4-cis-node",
			CheckName:          "test-check",
			RuleName:           "test-rule",
		},
	}

	expectedProfileResultsCluster2And3 = []*ResourceResultsByProfile{
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          0,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  4,
			ProfileName:        "ocp4-cis-node",
			CheckName:          "test-check",
			RuleName:           "test-rule",
		},
		{
			PassCount:          0,
			FailCount:          0,
			ErrorCount:         0,
			InfoCount:          1,
			ManualCount:        0,
			NotApplicableCount: 0,
			InconsistentCount:  0,
			ProfileName:        "ocp4-cis-node",
			CheckName:          "test-check-2",
			RuleName:           "test-rule",
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
	configStorage := checkResultsStorage.New(s.db)
	s.searcher = checkresultsSearch.New(configStorage)
	s.dataStore = New(s.storage, s.db, s.searcher)
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

	count, err := s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(len(ids), count)

	// upsert with read context
	s.Require().Error(s.dataStore.UpsertResult(s.hasReadCtx, rec2))

	retrieveRec1, found, err := s.storage.Get(s.hasReadCtx, rec1.GetId())
	s.Require().NoError(err)
	s.Require().True(found)
	protoassert.Equal(s.T(), rec1, retrieveRec1)
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

	count, err := s.storage.Count(s.hasReadCtx, search.EmptyQuery())
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
		{
			desc:          "Check name query - Full Access",
			query:         search.NewQueryBuilder().AddStrings(search.ComplianceOperatorCheckName, "test-check-2").ProtoQuery(),
			scopeKey:      testutils.UnrestrictedReadCtx,
			expectedCount: 1,
		},
		{
			desc: "Check name query and cluster 3 - Cluster 3 Access",
			query: search.NewQueryBuilder().AddStrings(search.ComplianceOperatorCheckName, "test-check-2").
				AddStrings(search.ClusterID, testconsts.Cluster3).ProtoQuery(),
			scopeKey:      testutils.Cluster3ReadWriteCtx,
			expectedCount: 1,
		},
		{
			desc: "Check name query and cluster 2 - Full Access",
			query: search.NewQueryBuilder().AddStrings(search.ComplianceOperatorCheckName, "test-check-2").
				AddStrings(search.ClusterID, testconsts.Cluster2).ProtoQuery(),
			scopeKey:      testutils.UnrestrictedReadCtx,
			expectedCount: 0,
		},
		{
			desc: "Check name query and cluster 3 and scan config name - Cluster 3 Access",
			query: search.NewQueryBuilder().AddStrings(search.ComplianceOperatorCheckName, "test-check-2").
				AddStrings(search.ClusterID, testconsts.Cluster3).
				AddStrings(search.ComplianceOperatorScanConfigName, "scanConfig2").ProtoQuery(),
			scopeKey:      testutils.Cluster3ReadWriteCtx,
			expectedCount: 1,
		},
		{
			desc: "Check name query and cluster 3 and scan config name - Cluster 3 Access",
			query: search.NewQueryBuilder().AddStrings(search.ComplianceOperatorCheckName, "test-check-2").
				AddStrings(search.ClusterID, testconsts.Cluster3).
				AddStrings(search.ComplianceOperatorScanConfigName, "scanConfig1").ProtoQuery(),
			scopeKey:      testutils.Cluster3ReadWriteCtx,
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		results, err := s.dataStore.SearchComplianceCheckResults(s.testContexts[tc.scopeKey], tc.query)
		s.Require().NoError(err)
		s.Require().Equal(tc.expectedCount, len(results))
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
		s.Require().NoError(err)
		s.Require().Equal(tc.expectedCount, count)
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
			expectedResults: expectedClusterScanCounts,
		},
		{
			desc:            "Empty query - Full access - sort on custom field",
			query:           search.NewQueryBuilder().WithPagination(search.NewPagination().AddSortOption(search.NewSortOption(search.ComplianceInconsistentCount).Reversed(true))).ProtoQuery(),
			scopeKey:        testutils.UnrestrictedReadCtx,
			expectedResults: expectedInconsistentClusterScanCounts,
		},
		{
			desc:            "Empty query - Only cluster 2 access",
			query:           search.NewQueryBuilder().ProtoQuery(),
			scopeKey:        testutils.Cluster2ReadWriteCtx,
			expectedResults: expectedCluster2OnlyScanCounts,
		},
		{
			desc:            "Cluster 2 query - Only cluster 2 access",
			query:           search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).ProtoQuery(),
			scopeKey:        testutils.Cluster2ReadWriteCtx,
			expectedResults: expectedCluster2OnlyScanCounts,
		},
		{
			desc: "Cluster 2 and 3 query - Only cluster 2 access",
			query: search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).
				AddStrings(search.ClusterID, testconsts.Cluster3).ProtoQuery(),
			scopeKey:        testutils.Cluster2ReadWriteCtx,
			expectedResults: expectedCluster2OnlyScanCounts,
		},
		{
			desc: "Cluster 2 and 3 query - Full Access",
			query: search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).
				AddStrings(search.ClusterID, testconsts.Cluster3).ProtoQuery(),
			scopeKey:        testutils.UnrestrictedReadCtx,
			expectedResults: expectedCluster2And3ScanCounts,
		},
	}

	for _, tc := range testCases {
		results, err := s.dataStore.ComplianceCheckResultStats(s.testContexts[tc.scopeKey], tc.query)
		s.Require().NoError(err)
		s.Require().Equal(tc.expectedResults, results)
	}
}

func (s *complianceCheckResultDataStoreTestSuite) TestComplianceClusterStats() {
	s.setupTestData()
	testCases := []struct {
		desc            string
		query           *apiV1.Query
		scopeKey        string
		expectedResults []*ResultStatusCountByCluster
	}{
		{
			desc:            "Empty query - Full access",
			query:           search.NewQueryBuilder().ProtoQuery(),
			scopeKey:        testutils.UnrestrictedReadCtx,
			expectedResults: expectedClusterCounts,
		},
		{
			desc:            "Empty query - Full access - sort by inconsistent count",
			query:           search.NewQueryBuilder().WithPagination(search.NewPagination().AddSortOption(search.NewSortOption(search.ComplianceInconsistentCount))).ProtoQuery(),
			scopeKey:        testutils.UnrestrictedReadCtx,
			expectedResults: expectedClusterCountsByInconsistent,
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
		results, err := s.dataStore.ComplianceClusterStats(s.testContexts[tc.scopeKey], tc.query)
		s.Require().NoError(err)
		s.Require().Equal(tc.expectedResults, results)
	}
}

func (s *complianceCheckResultDataStoreTestSuite) TestCountByFieldCluster() {
	s.setupTestData()
	testCases := []struct {
		desc          string
		query         *apiV1.Query
		scopeKey      string
		expectedCount int
	}{
		{
			desc:          "Empty query - Full access",
			query:         search.NewQueryBuilder().ProtoQuery(),
			scopeKey:      testutils.UnrestrictedReadCtx,
			expectedCount: 3,
		},
		{
			desc:          "Empty query - Only cluster 2 access",
			query:         search.NewQueryBuilder().ProtoQuery(),
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedCount: 1,
		},
		{
			desc:          "Cluster 2 query - Only cluster 2 access",
			query:         search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).ProtoQuery(),
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedCount: 1,
		},
		{
			desc: "Cluster 2 and 3 query - Only cluster 2 access",
			query: search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).
				AddStrings(search.ClusterID, testconsts.Cluster3).ProtoQuery(),
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedCount: 1,
		},
		{
			desc: "Cluster 2 and 3 query - Full Access",
			query: search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).
				AddStrings(search.ClusterID, testconsts.Cluster3).ProtoQuery(),
			scopeKey:      testutils.UnrestrictedReadCtx,
			expectedCount: 2,
		},
	}

	for _, tc := range testCases {
		results, err := s.dataStore.CountByField(s.testContexts[tc.scopeKey], tc.query, search.ClusterID)
		s.Require().NoError(err)
		s.Require().Equal(tc.expectedCount, results)
	}
}

func (s *complianceCheckResultDataStoreTestSuite) TestCountByFieldProfile() {
	s.setupTestData()
	testCases := []struct {
		desc          string
		query         *apiV1.Query
		scopeKey      string
		expectedCount int
	}{
		{
			desc:          "Empty query - Full access",
			query:         search.NewQueryBuilder().ProtoQuery(),
			scopeKey:      testutils.UnrestrictedReadCtx,
			expectedCount: 1,
		},
		{
			desc:          "Empty query - Only cluster 2 access",
			query:         search.NewQueryBuilder().ProtoQuery(),
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedCount: 1,
		},
		{
			desc:          "Cluster 2 query - Only cluster 2 access",
			query:         search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).ProtoQuery(),
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedCount: 1,
		},
		{
			desc: "Cluster 2 and 3 query - Only cluster 2 access",
			query: search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).
				AddStrings(search.ClusterID, testconsts.Cluster3).ProtoQuery(),
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedCount: 1,
		},
		{
			desc: "Cluster 2 and 3 query - Full Access",
			query: search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).
				AddStrings(search.ClusterID, testconsts.Cluster3).ProtoQuery(),
			scopeKey:      testutils.UnrestrictedReadCtx,
			expectedCount: 1,
		},
	}

	for _, tc := range testCases {
		results, err := s.dataStore.CountByField(s.testContexts[tc.scopeKey], tc.query, search.ComplianceOperatorProfileName)
		s.Require().NoError(err)
		s.Require().Equal(tc.expectedCount, results)
	}
}

func (s *complianceCheckResultDataStoreTestSuite) TestCountByFieldCheck() {
	s.setupTestData()
	testCases := []struct {
		desc          string
		query         *apiV1.Query
		scopeKey      string
		expectedCount int
	}{
		{
			desc:          "Empty query - Full access",
			query:         search.NewQueryBuilder().ProtoQuery(),
			scopeKey:      testutils.UnrestrictedReadCtx,
			expectedCount: 2,
		},
		{
			desc:          "Empty query - Only cluster 2 access",
			query:         search.NewQueryBuilder().ProtoQuery(),
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedCount: 1,
		},
		{
			desc:          "Cluster 2 query - Only cluster 2 access",
			query:         search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).ProtoQuery(),
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedCount: 1,
		},
		{
			desc: "Cluster 2 and 3 query - Only cluster 2 access",
			query: search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).
				AddStrings(search.ClusterID, testconsts.Cluster3).ProtoQuery(),
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedCount: 1,
		},
		{
			desc: "Cluster 2 and 3 query - Full Access",
			query: search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).
				AddStrings(search.ClusterID, testconsts.Cluster3).ProtoQuery(),
			scopeKey:      testutils.UnrestrictedReadCtx,
			expectedCount: 2,
		},
	}

	for _, tc := range testCases {
		results, err := s.dataStore.CountByField(s.testContexts[tc.scopeKey], tc.query, search.ComplianceOperatorCheckName)
		s.Require().NoError(err)
		s.Require().Equal(tc.expectedCount, results)
	}
}

func (s *complianceCheckResultDataStoreTestSuite) TestCountByFieldScanConfig() {
	s.setupTestData()
	testCases := []struct {
		desc          string
		query         *apiV1.Query
		scopeKey      string
		expectedCount int
	}{
		{
			desc:          "Empty query - Full access",
			query:         search.NewQueryBuilder().ProtoQuery(),
			scopeKey:      testutils.UnrestrictedReadCtx,
			expectedCount: 2,
		},
		{
			desc:          "Empty query - Only cluster 2 access",
			query:         search.NewQueryBuilder().ProtoQuery(),
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedCount: 1,
		},
		{
			desc:          "Cluster 2 query - Only cluster 2 access",
			query:         search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).ProtoQuery(),
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedCount: 1,
		},
		{
			desc: "Cluster 2 and 3 query - Only cluster 2 access",
			query: search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).
				AddStrings(search.ClusterID, testconsts.Cluster3).ProtoQuery(),
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedCount: 1,
		},
		{
			desc: "Cluster 2 and 3 query - Full Access",
			query: search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).
				AddStrings(search.ClusterID, testconsts.Cluster3).ProtoQuery(),
			scopeKey:      testutils.UnrestrictedReadCtx,
			expectedCount: 2,
		},
	}

	for _, tc := range testCases {
		results, err := s.dataStore.CountByField(s.testContexts[tc.scopeKey], tc.query, search.ComplianceOperatorScanConfigName)
		s.Require().NoError(err)
		s.Require().Equal(tc.expectedCount, results)
	}
}

func (s *complianceCheckResultDataStoreTestSuite) TestGetComplianceCheckResult() {
	s.setupTestData()

	rec1 := getTestRec(testconsts.Cluster1)
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec1))

	testCases := []struct {
		desc             string
		id               string
		scopeKey         string
		expectedResponse *storage.ComplianceOperatorCheckResultV2
	}{
		{
			desc:             "ID exists with cluster access",
			id:               rec1.GetId(),
			scopeKey:         testutils.UnrestrictedReadCtx,
			expectedResponse: rec1,
		},
		{
			desc:             "ID exists -- wrong cluster access",
			id:               rec1.GetId(),
			scopeKey:         testutils.Cluster2ReadWriteCtx,
			expectedResponse: nil,
		},
		{
			desc:             "ID does not exist",
			id:               uuid.NewV4().String(),
			scopeKey:         testutils.UnrestrictedReadCtx,
			expectedResponse: nil,
		},
	}

	for _, tc := range testCases {
		result, found, err := s.dataStore.GetComplianceCheckResult(s.testContexts[tc.scopeKey], tc.id)
		s.Require().NoError(err)
		protoassert.Equal(s.T(), tc.expectedResponse, result)
		s.Require().NotEqual(tc.expectedResponse == nil, found)
	}
}

func (s *complianceCheckResultDataStoreTestSuite) TestWalkByQueryCheckResult() {
	s.setupTestData()
	rec1 := getTestRec(testconsts.Cluster1)
	rec1.CheckName = "test-check1"
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec1))
	rec2 := getTestRec(testconsts.Cluster2)
	rec2.CheckName = "test-check2"
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec2))
	parsedQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanConfigName, rec1.GetScanConfigName()).
		AddExactMatches(search.ClusterID, rec1.ClusterId).
		ProtoQuery()
	type repResults struct {
		ClusterName string
		CheckName   string
		Status      string
	}
	results, err := s.dataStore.SearchComplianceCheckResults(s.testContexts[testutils.UnrestrictedReadWriteCtx], parsedQuery)
	s.Require().NoError(err)
	s.Require().NotEmpty(results)
	resultsQuery := []*repResults{}

	err = s.dataStore.WalkByQuery(s.testContexts[testutils.UnrestrictedReadCtx], parsedQuery, func(c *storage.ComplianceOperatorCheckResultV2) error {
		res := &repResults{
			ClusterName: c.GetClusterName(),
			CheckName:   c.GetCheckName(),
			Status:      c.GetStatus().String(),
		}
		resultsQuery = append(resultsQuery, res)
		return nil
	})
	s.Require().NoError(err)
	s.Require().NotEmpty(resultsQuery)
	s.Require().Equal(resultsQuery[0].CheckName, rec1.GetCheckName())
}

func (s *complianceCheckResultDataStoreTestSuite) TestComplianceProfileResultStats() {
	s.setupTestData()
	testCases := []struct {
		desc            string
		query           *apiV1.Query
		scopeKey        string
		expectedResults []*ResourceResultCountByProfile
	}{
		{
			desc:            "Empty query - Full access",
			query:           search.NewQueryBuilder().ProtoQuery(),
			scopeKey:        testutils.UnrestrictedReadCtx,
			expectedResults: expectedProfileCounts,
		},
		{
			desc:            "Empty query - Full access - Custom Sort",
			query:           search.NewQueryBuilder().WithPagination(search.NewPagination().AddSortOption(search.NewSortOption(search.ComplianceInconsistentCount).Reversed(true))).ProtoQuery(),
			scopeKey:        testutils.UnrestrictedReadCtx,
			expectedResults: expectedProfileCounts,
		},
		{
			desc:            "Empty query - Only cluster 2 access",
			query:           search.NewQueryBuilder().ProtoQuery(),
			scopeKey:        testutils.Cluster2ReadWriteCtx,
			expectedResults: expectedProfileCountsCluster2,
		},
		{
			desc:            "Cluster 2 query - Only cluster 2 access",
			query:           search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).ProtoQuery(),
			scopeKey:        testutils.Cluster2ReadWriteCtx,
			expectedResults: expectedProfileCountsCluster2,
		},
		{
			desc: "Cluster 2 and 3 query - Only cluster 2 access",
			query: search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).
				AddStrings(search.ClusterID, testconsts.Cluster3).ProtoQuery(),
			scopeKey:        testutils.Cluster2ReadWriteCtx,
			expectedResults: expectedProfileCountsCluster2,
		},
		{
			desc: "Cluster 2 and 3 query - Full Access",
			query: search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).
				AddStrings(search.ClusterID, testconsts.Cluster3).ProtoQuery(),
			scopeKey:        testutils.UnrestrictedReadCtx,
			expectedResults: expectedProfileCountsCluster2And3,
		},
	}

	for _, tc := range testCases {
		results, err := s.dataStore.ComplianceProfileResultStats(s.testContexts[tc.scopeKey], tc.query)
		s.Require().NoError(err)
		s.Require().Equal(tc.expectedResults, results)
	}
}

func (s *complianceCheckResultDataStoreTestSuite) TestComplianceProfileResults() {
	s.setupTestData()
	testCases := []struct {
		desc            string
		query           *apiV1.Query
		scopeKey        string
		expectedResults []*ResourceResultsByProfile
	}{
		{
			desc:            "Empty query - Full access",
			query:           search.NewQueryBuilder().ProtoQuery(),
			scopeKey:        testutils.UnrestrictedReadCtx,
			expectedResults: expectedProfileResults,
		},
		{
			desc:            "Empty query - Full access - sort on custom field",
			query:           search.NewQueryBuilder().WithPagination(search.NewPagination().AddSortOption(search.NewSortOption(search.ComplianceInconsistentCount).Reversed(false))).ProtoQuery(),
			scopeKey:        testutils.UnrestrictedReadCtx,
			expectedResults: expectedProfileResultsByInconsistent,
		},
		{
			desc:            "Empty query - Only cluster 2 access",
			query:           search.NewQueryBuilder().ProtoQuery(),
			scopeKey:        testutils.Cluster2ReadWriteCtx,
			expectedResults: expectedProfileResultsCluster2,
		},
		{
			desc:            "Cluster 2 query - Only cluster 2 access",
			query:           search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).ProtoQuery(),
			scopeKey:        testutils.Cluster2ReadWriteCtx,
			expectedResults: expectedProfileResultsCluster2,
		},
		{
			desc: "Cluster 2 and 3 query - Only cluster 2 access",
			query: search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).
				AddStrings(search.ClusterID, testconsts.Cluster3).ProtoQuery(),
			scopeKey:        testutils.Cluster2ReadWriteCtx,
			expectedResults: expectedProfileResultsCluster2,
		},
		{
			desc: "Cluster 2 and 3 query - Full Access",
			query: search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).
				AddStrings(search.ClusterID, testconsts.Cluster3).ProtoQuery(),
			scopeKey:        testutils.UnrestrictedReadCtx,
			expectedResults: expectedProfileResultsCluster2And3,
		},
	}

	for _, tc := range testCases {
		results, err := s.dataStore.ComplianceProfileResults(s.testContexts[tc.scopeKey], tc.query)
		s.Require().NoError(err)
		s.Require().Equal(tc.expectedResults, results)
	}
}

func (s *complianceCheckResultDataStoreTestSuite) TestDeleteResultsByScanConfigAndCluster() {
	scanResults1 := getTestRec(testconsts.Cluster1)
	scanResults2 := getTestRec(testconsts.Cluster2)
	scanResults3 := getTestRec(testconsts.Cluster3)
	scanResults4 := getTestRec2(testconsts.Cluster1)
	scanResults5 := getTestRec2(testconsts.Cluster2)

	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, scanResults1))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, scanResults2))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, scanResults3))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, scanResults4))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, scanResults5))

	err := s.dataStore.DeleteResultsByScanConfigAndCluster(s.hasWriteCtx, "", []string{scanResults2.GetClusterId()})
	s.NoError(err)
	actualResultsCount, err := s.dataStore.CountCheckResults(s.hasWriteCtx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(5, actualResultsCount, "Empty scan schedule name does not delete any records")

	err = s.dataStore.DeleteResultsByScanConfigAndCluster(s.hasWriteCtx, scanResults2.GetScanConfigName(), []string{})
	s.NoError(err)
	actualResultsCount, err = s.dataStore.CountCheckResults(s.hasWriteCtx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(5, actualResultsCount, "Empty scan list of clusters does not delete any records")

	err = s.dataStore.DeleteResultsByScanConfigAndCluster(s.hasWriteCtx, scanResults2.GetScanConfigName(), []string{scanResults2.GetClusterId(), scanResults3.GetClusterId()})
	s.NoError(err)
	actualResultsCount, err = s.dataStore.CountCheckResults(s.hasWriteCtx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(3, actualResultsCount, "2 records should be removed")
}

func (s *complianceCheckResultDataStoreTestSuite) TestDeleteResultsByScanConfigAndProfile() {
	scanConfigName1 := "scan-config1"
	scanConfigName2 := "scan-config2"

	ruleName1 := "rule1"
	ruleName2 := "rule2"
	ruleName3 := "rule3"

	scanResults111 := getTestRecWithRule(scanConfigName1, testconsts.Cluster1, ruleName1)
	scanResults121 := getTestRecWithRule(scanConfigName1, testconsts.Cluster2, ruleName1)
	scanResults112 := getTestRecWithRule(scanConfigName1, testconsts.Cluster1, ruleName2)
	scanResults122 := getTestRecWithRule(scanConfigName1, testconsts.Cluster2, ruleName2)
	scanResults213 := getTestRecWithRule(scanConfigName2, testconsts.Cluster1, ruleName3)

	// With current implementation this option is not possible because one profile cannot be used in two scan schedules.
	scanResults212 := getTestRecWithRule(scanConfigName2, testconsts.Cluster1, ruleName2)

	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, scanResults111))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, scanResults121))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, scanResults112))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, scanResults122))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, scanResults213))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, scanResults212))

	ruleName2Cluster1RefID := internaltov2storage.BuildNameRefID(testconsts.Cluster1, ruleName2)
	ruleName2Cluster2RefID := internaltov2storage.BuildNameRefID(testconsts.Cluster2, ruleName2)

	s.T().Log(ruleName2Cluster1RefID)
	s.T().Log(scanResults112.GetRuleRefId())

	err := s.dataStore.DeleteResultsByScanConfigAndRules(s.hasWriteCtx, "", []string{ruleName2Cluster1RefID, ruleName2Cluster2RefID})
	s.NoError(err)
	actualResultsCount, err := s.dataStore.CountCheckResults(s.hasWriteCtx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(6, actualResultsCount, "Empty scan schedule name does not delete any records")

	err = s.dataStore.DeleteResultsByScanConfigAndRules(s.hasWriteCtx, scanConfigName1, []string{})
	s.NoError(err)
	actualResultsCount, err = s.dataStore.CountCheckResults(s.hasWriteCtx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(6, actualResultsCount, "Empty scan list of clusters does not delete any records")

	err = s.dataStore.DeleteResultsByScanConfigAndRules(s.hasWriteCtx, scanConfigName1, []string{ruleName2Cluster1RefID, ruleName2Cluster2RefID})
	s.NoError(err)
	actualResultsCount, err = s.dataStore.CountCheckResults(s.hasWriteCtx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(4, actualResultsCount, "2 records should be removed")
}

func (s *complianceCheckResultDataStoreTestSuite) setupTestData() {
	// make sure we have nothing
	checkResultIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(checkResultIDs)

	_, err = s.db.DB.Exec(context.Background(), "insert into compliance_operator_scan_configuration_v2 (id, scanconfigname) values ($1, $2)", fixtureconsts.ComplianceScanConfigID1, "scanConfig1")
	s.Require().NoError(err)
	_, err = s.db.DB.Exec(context.Background(), "insert into compliance_operator_scan_configuration_v2 (id, scanconfigname) values ($1, $2)", fixtureconsts.ComplianceScanConfigID2, "scanConfig2")
	s.Require().NoError(err)

	_, err = s.db.DB.Exec(context.Background(), "insert into clusters (id, name) values ($1, $2)", testconsts.Cluster1, "cluster1")
	s.Require().NoError(err)
	_, err = s.db.DB.Exec(context.Background(), "insert into clusters (id, name) values ($1, $2)", testconsts.Cluster2, "cluster2")
	s.Require().NoError(err)
	_, err = s.db.DB.Exec(context.Background(), "insert into clusters (id, name) values ($1, $2)", testconsts.Cluster3, "cluster3")
	s.Require().NoError(err)

	recs := []*storage.ComplianceOperatorCheckResultV2{
		getTestRec(testconsts.Cluster1),
		getTestRec(testconsts.Cluster2),
		getTestRec(testconsts.Cluster2),
		getTestRec(testconsts.Cluster2),
		getTestRec(testconsts.Cluster3),
		getTestRec2(testconsts.Cluster3),
	}

	profileCluster := map[string]string{
		testconsts.Cluster1: internaltov2storage.BuildProfileRefID(testconsts.Cluster1, "ocp4-cis-node", "node"),
		testconsts.Cluster2: internaltov2storage.BuildProfileRefID(testconsts.Cluster2, "ocp4-cis-node", "node"),
		testconsts.Cluster3: internaltov2storage.BuildProfileRefID(testconsts.Cluster3, "ocp4-cis-node", "node"),
	}

	ruleCluster := map[string]string{
		testconsts.Cluster1: internaltov2storage.BuildNameRefID(testconsts.Cluster1, "test-rule"),
		testconsts.Cluster2: internaltov2storage.BuildNameRefID(testconsts.Cluster2, "test-rule"),
		testconsts.Cluster3: internaltov2storage.BuildNameRefID(testconsts.Cluster3, "test-rule"),
	}

	for k, v := range profileCluster {
		_, err = s.db.DB.Exec(context.Background(), "insert into compliance_operator_profile_v2 (id, profileid, name, producttype, clusterid, profilerefid) values ($1, $2, $3, $4, $5, $6)", uuid.NewV4().String(), "profile-1", "ocp4-cis-node", "node", k, v)

		s.Require().NoError(err)
	}

	for k, v := range ruleCluster {
		_, err = s.db.DB.Exec(context.Background(), "insert into compliance_operator_rule_v2 (id, name, clusterid, rulerefid) values ($1, $2, $3, $4)", uuid.NewV4().String(), "test-rule", k, v)
		s.Require().NoError(err)
	}

	for _, rec := range recs {
		s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec))

		_, err = s.db.DB.Exec(context.Background(), "insert into compliance_operator_scan_v2 (id, scanconfigname, scanname, profile_profilerefid, clusterid, scanrefid) values ($1, $2, $3, $4, $5, $6)", uuid.NewV4().String(), rec.GetScanConfigName(), rec.GetScanName(), profileCluster[rec.GetClusterId()], rec.GetClusterId(), rec.GetScanRefId())
		s.Require().NoError(err)
	}

}

func getTestRec(clusterID string) *storage.ComplianceOperatorCheckResultV2 {
	scanName := uuid.NewV4().String()
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
		CreatedTime:    protocompat.TimestampNow(),
		ScanName:       scanName,
		ScanConfigName: "scanConfig1",
		ScanRefId:      internaltov2storage.BuildNameRefID(clusterID, scanName),
		RuleRefId:      internaltov2storage.BuildNameRefID(clusterID, "test-rule"),
	}
}

func getTestRec2(clusterID string) *storage.ComplianceOperatorCheckResultV2 {
	scanName := uuid.NewV4().String()
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
		CreatedTime:    protocompat.TimestampNow(),
		ScanName:       scanName,
		ScanConfigName: "scanConfig2",
		ScanRefId:      internaltov2storage.BuildNameRefID(clusterID, scanName),
		RuleRefId:      internaltov2storage.BuildNameRefID(clusterID, "test-rule"),
	}
}

func getTestRecWithRule(scanConfigName string, clusterID string, rule string) *storage.ComplianceOperatorCheckResultV2 {
	scanName := uuid.NewV4().String()
	return &storage.ComplianceOperatorCheckResultV2{
		Id:             uuid.NewV4().String(),
		CheckId:        uuid.NewV4().String(),
		CheckName:      fmt.Sprintf("check-%s", scanConfigName),
		ClusterId:      clusterID,
		Status:         storage.ComplianceOperatorCheckResultV2_INFO,
		Severity:       storage.RuleSeverity_INFO_RULE_SEVERITY,
		CreatedTime:    protocompat.TimestampNow(),
		ScanName:       scanName,
		ScanConfigName: scanConfigName,
		ScanRefId:      internaltov2storage.BuildNameRefID(clusterID, scanName),
		RuleRefId:      internaltov2storage.BuildNameRefID(clusterID, rule),
	}
}
