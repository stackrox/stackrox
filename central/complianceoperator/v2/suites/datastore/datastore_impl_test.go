//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	suiteStorage "github.com/stackrox/rox/central/complianceoperator/v2/suites/store/postgres"
	"github.com/stackrox/rox/central/convert/testutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	sacTestUtils "github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestComplianceSuiteDataStore(t *testing.T) {
	suite.Run(t, new(complianceSuiteDataStoreTestSuite))
}

type complianceSuiteDataStoreTestSuite struct {
	suite.Suite

	hasReadCtx            context.Context
	hasWriteCtx           context.Context
	noAccessCtx           context.Context
	testContexts          map[string]context.Context
	nonComplianceContexts map[string]context.Context

	dataStore DataStore
	storage   suiteStorage.Store
	db        *pgtest.TestPostgres
}

func (s *complianceSuiteDataStoreTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip tests when ComplianceEnhancements disabled")
		s.T().SkipNow()
	}
}

func (s *complianceSuiteDataStoreTestSuite) SetupTest() {
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.noAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.testContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Compliance)
	s.nonComplianceContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Deployment)

	s.db = pgtest.ForT(s.T())

	s.storage = suiteStorage.New(s.db)
	s.dataStore = GetTestPostgresDataStore(s.T(), s.db)
}

func (s *complianceSuiteDataStoreTestSuite) TestGetSuite() {
	// make sure we have nothing
	suiteIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(suiteIDs)
	suites := []*storage.ComplianceOperatorSuiteV2{
		s.getTestSuite(testconsts.Cluster1),
		s.getTestSuite(testconsts.Cluster2),
	}
	suiteIDs = []string{suites[0].Id, suites[1].Id}
	s.Require().NoError(s.dataStore.UpsertSuites(s.testContexts[sacTestUtils.UnrestrictedReadWriteCtx], suites))

	// Delayed clean up
	defer func() {
		s.Require().NoError(s.storage.DeleteMany(s.testContexts[sacTestUtils.UnrestrictedReadWriteCtx], suiteIDs))
	}()

	testCases := []struct {
		desc                string
		testContext         context.Context
		expectedRecordIndex set.FrozenIntSet
	}{
		{
			desc:                "GetSuite - Full access",
			testContext:         s.testContexts[sacTestUtils.UnrestrictedReadWriteCtx],
			expectedRecordIndex: set.NewFrozenIntSet(0, 1),
		},
		{
			desc:                "GetSuite - No access",
			testContext:         s.noAccessCtx,
			expectedRecordIndex: set.NewFrozenIntSet(),
		},
		{
			desc:                "GetSuite - Cluster 1 access",
			testContext:         s.testContexts[sacTestUtils.Cluster1ReadWriteCtx],
			expectedRecordIndex: set.NewFrozenIntSet(0),
		},
	}

	for _, tc := range testCases {
		for i, id := range suiteIDs {
			suite1, exists, err := s.dataStore.GetSuite(tc.testContext, id)
			s.Require().NoError(err)
			if tc.expectedRecordIndex.Contains(i) {
				s.Require().True(exists)
				protoassert.Equal(s.T(), suites[i], suite1)
			} else {
				s.Require().False(exists)
				s.Require().Nil(suite1)
			}
		}
	}
}

func (s *complianceSuiteDataStoreTestSuite) TestUpsertSuite() {
	// make sure we have nothing
	suiteIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(suiteIDs)

	testCases := []struct {
		desc                string
		suites              []*storage.ComplianceOperatorSuiteV2
		testContext         context.Context
		expectedRecordIndex set.FrozenIntSet
	}{
		{
			desc: "Write 3 clusters - Full access",
			suites: []*storage.ComplianceOperatorSuiteV2{
				s.getTestSuite(testconsts.Cluster1),
				s.getTestSuite(testconsts.Cluster2),
				s.getTestSuite(testconsts.Cluster3),
			},
			testContext:         s.testContexts[sacTestUtils.UnrestrictedReadWriteCtx],
			expectedRecordIndex: set.NewFrozenIntSet(0, 1, 2),
		},
		{
			desc: "Write 3 clusters - No access",
			suites: []*storage.ComplianceOperatorSuiteV2{
				s.getTestSuite(testconsts.Cluster1),
				s.getTestSuite(testconsts.Cluster2),
				s.getTestSuite(testconsts.Cluster3),
			},
			testContext:         s.noAccessCtx,
			expectedRecordIndex: set.NewFrozenIntSet(),
		},
		{
			desc: "Write 3 clusters - Cluster 1 access",
			suites: []*storage.ComplianceOperatorSuiteV2{
				s.getTestSuite(testconsts.Cluster1),
				s.getTestSuite(testconsts.Cluster2),
				s.getTestSuite(testconsts.Cluster3),
			},
			testContext:         s.testContexts[sacTestUtils.Cluster1ReadWriteCtx],
			expectedRecordIndex: set.NewFrozenIntSet(0),
		},
	}

	for _, tc := range testCases {
		for index, suite := range tc.suites {
			if tc.expectedRecordIndex.Contains(index) {
				s.Require().NoError(s.dataStore.UpsertSuite(tc.testContext, suite))
			} else {
				s.Require().Error(s.dataStore.UpsertSuite(tc.testContext, suite), "access to resource denied")
			}
		}

		count, err := s.storage.Count(s.hasReadCtx, search.EmptyQuery())
		s.Require().NoError(err)
		s.Require().Equal(tc.expectedRecordIndex.Cardinality(), count)

		// Clean up
		for _, suite := range tc.suites {
			s.Require().NoError(s.dataStore.DeleteSuite(s.hasWriteCtx, suite.GetId()))
		}
	}
}

func (s *complianceSuiteDataStoreTestSuite) TestUpsertSuites() {
	// make sure we have nothing
	allSuiteIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(allSuiteIDs)
	suites := []*storage.ComplianceOperatorSuiteV2{
		s.getTestSuite(testconsts.Cluster1),
		s.getTestSuite(testconsts.Cluster2),
		s.getTestSuite(testconsts.Cluster3),
	}
	allSuiteIDs = []string{suites[0].Id, suites[1].Id, suites[2].Id}

	testCases := []struct {
		desc        string
		testContext context.Context
		hasError    bool
	}{
		{
			desc:        "UpsertSuites - Full access",
			testContext: s.testContexts[sacTestUtils.UnrestrictedReadWriteCtx],
			hasError:    false,
		},
		{
			desc:        "UpsertSuites - No access",
			testContext: s.noAccessCtx,
			hasError:    true,
		},
		{
			desc:        "UpsertSuites - Cluster 1 access",
			testContext: s.testContexts[sacTestUtils.Cluster1ReadWriteCtx],
			hasError:    true,
		},
	}

	for _, tc := range testCases {
		err := s.dataStore.UpsertSuites(tc.testContext, suites)
		if tc.hasError {
			s.Require().Error(err)
			count, err := s.storage.Count(s.hasReadCtx, search.EmptyQuery())
			s.Require().NoError(err)
			s.Require().Zero(count)
		} else {
			s.Require().NoError(err)
			count, err := s.storage.Count(s.hasReadCtx, search.EmptyQuery())
			s.Require().NoError(err)
			s.Require().Equal(len(allSuiteIDs), count)
			ids, err := s.storage.GetIDs(s.hasReadCtx)
			s.Require().NoError(err)
			s.Require().ElementsMatch(allSuiteIDs, ids)
		}

		// Clean up
		s.Require().NoError(s.storage.DeleteMany(s.testContexts[sacTestUtils.UnrestrictedReadWriteCtx], allSuiteIDs))
	}
}

func (s *complianceSuiteDataStoreTestSuite) TestDeleteSuiteByCluster() {
	suite := s.getTestSuite(testconsts.Cluster1)
	s.Require().NoError(s.dataStore.UpsertSuite(s.hasWriteCtx, suite))

	count, err := s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(1, count)

	s.Require().NoError(s.dataStore.DeleteSuitesByCluster(s.hasWriteCtx, testconsts.Cluster1))

	count, err = s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(0, count)
}

func (s *complianceSuiteDataStoreTestSuite) TestDeleteSuite() {
	// make sure we have nothing
	suiteIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(suiteIDs)

	testCases := []struct {
		desc                string
		suites              []*storage.ComplianceOperatorSuiteV2
		testContext         context.Context
		expectedRecordIndex set.FrozenIntSet
	}{
		{
			desc: "Write 3 clusters - Full access",
			suites: []*storage.ComplianceOperatorSuiteV2{
				s.getTestSuite(testconsts.Cluster1),
				s.getTestSuite(testconsts.Cluster2),
				s.getTestSuite(testconsts.Cluster3),
			},
			testContext:         s.testContexts[sacTestUtils.UnrestrictedReadWriteCtx],
			expectedRecordIndex: set.NewFrozenIntSet(0, 1, 2),
		},
		{
			desc: "Write 3 clusters - No access",
			suites: []*storage.ComplianceOperatorSuiteV2{
				s.getTestSuite(testconsts.Cluster1),
				s.getTestSuite(testconsts.Cluster2),
				s.getTestSuite(testconsts.Cluster3),
			},
			testContext:         s.noAccessCtx,
			expectedRecordIndex: set.NewFrozenIntSet(),
		},
		{
			desc: "Write 3 clusters - Cluster 1 access",
			suites: []*storage.ComplianceOperatorSuiteV2{
				s.getTestSuite(testconsts.Cluster1),
				s.getTestSuite(testconsts.Cluster2),
				s.getTestSuite(testconsts.Cluster3),
			},
			testContext:         s.testContexts[sacTestUtils.Cluster1ReadWriteCtx],
			expectedRecordIndex: set.NewFrozenIntSet(0),
		},
	}

	for _, tc := range testCases {
		for _, suite := range tc.suites {
			s.Require().NoError(s.dataStore.UpsertSuite(s.hasWriteCtx, suite))
		}

		for _, suite := range tc.suites {
			s.Require().NoError(s.dataStore.DeleteSuite(tc.testContext, suite.GetId()))
		}

		count, err := s.storage.Count(s.hasReadCtx, search.EmptyQuery())
		s.Require().NoError(err)
		// If we could not delete the suite then they will remain.
		s.Require().Equal(len(tc.suites)-tc.expectedRecordIndex.Cardinality(), count)

		// Clean up
		for _, suite := range tc.suites {
			s.Require().NoError(s.dataStore.DeleteSuite(s.hasWriteCtx, suite.GetId()))
		}
	}
}

func (s *complianceSuiteDataStoreTestSuite) TestGetSuitesByCluster() {
	// make sure we have nothing
	suiteIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(suiteIDs)

	testSuite1 := s.getTestSuite(testconsts.Cluster1)
	testSuite2 := s.getTestSuite(testconsts.Cluster1)
	testSuite3 := s.getTestSuite(testconsts.Cluster2)

	s.Require().NoError(s.dataStore.UpsertSuite(s.hasWriteCtx, testSuite1))
	s.Require().NoError(s.dataStore.UpsertSuite(s.hasWriteCtx, testSuite2))
	s.Require().NoError(s.dataStore.UpsertSuite(s.hasWriteCtx, testSuite3))

	count, err := s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(3, count)

	testCases := []struct {
		desc            string
		clusterID       string
		testContext     context.Context
		expectedResults []*storage.ComplianceOperatorSuiteV2
		expectedCount   int
	}{
		{
			desc:            "Suites exist - Full access",
			clusterID:       testconsts.Cluster1,
			testContext:     s.testContexts[sacTestUtils.UnrestrictedReadCtx],
			expectedResults: []*storage.ComplianceOperatorSuiteV2{testSuite1, testSuite2},
			expectedCount:   2,
		},
		{
			desc:            "Suites exist - Cluster 1 access",
			clusterID:       testconsts.Cluster1,
			testContext:     s.testContexts[sacTestUtils.Cluster1ReadWriteCtx],
			expectedResults: []*storage.ComplianceOperatorSuiteV2{testSuite1, testSuite2},
			expectedCount:   2,
		},
		{
			desc:            "Suites exist - Cluster 2 access",
			clusterID:       testconsts.Cluster1,
			testContext:     s.testContexts[sacTestUtils.Cluster2ReadWriteCtx],
			expectedResults: nil,
			expectedCount:   0,
		},
		{
			desc:            "Suites exists - No compliance access",
			clusterID:       testconsts.Cluster1,
			testContext:     s.nonComplianceContexts[sacTestUtils.UnrestrictedReadCtx],
			expectedResults: nil,
			expectedCount:   0,
		},
		{
			desc:            "Suite does not exist - Full access",
			clusterID:       fixtureconsts.ClusterFake1,
			testContext:     s.testContexts[sacTestUtils.UnrestrictedReadCtx],
			expectedResults: nil,
			expectedCount:   0,
		},
	}
	for _, tc := range testCases {
		retrievedObjects, err := s.dataStore.GetSuitesByCluster(tc.testContext, tc.clusterID)
		s.Require().NoError(err)
		s.Require().Equal(tc.expectedCount, len(retrievedObjects))
		protoassert.SlicesEqual(s.T(), tc.expectedResults, retrievedObjects)
	}
}

func (s *complianceSuiteDataStoreTestSuite) getTestSuite(clusterID string) *storage.ComplianceOperatorSuiteV2 {
	suite := testutils.GetSuiteStorage(s.T())
	suite.ClusterId = clusterID
	suite.Id = uuid.NewV4().String()
	return suite
}
