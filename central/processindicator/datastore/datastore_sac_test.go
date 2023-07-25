//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	sacTestUtils "github.com/stackrox/rox/pkg/sac/testutils"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestProcessIndicatorDataStoreSAC(t *testing.T) {
	suite.Run(t, new(processIndicatorDatastoreSACSuite))
}

type processIndicatorDatastoreSACSuite struct {
	suite.Suite

	pool postgres.DB

	datastore DataStore

	optionsMap searchPkg.OptionsMap

	testContexts            map[string]context.Context
	testProcessIndicatorIDs []string
}

func (s *processIndicatorDatastoreSACSuite) SetupSuite() {
	var err error

	pgtestbase := pgtest.ForT(s.T())
	s.Require().NotNil(pgtestbase)
	s.pool = pgtestbase.DB
	s.datastore, err = GetTestPostgresDataStore(s.T(), s.pool)
	s.Require().NoError(err)
	s.optionsMap = schema.ProcessIndicatorsSchema.OptionsMap

	s.testContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(),
		resources.DeploymentExtension)
}

func (s *processIndicatorDatastoreSACSuite) TearDownSuite() {
	s.pool.Close()
}

func (s *processIndicatorDatastoreSACSuite) SetupTest() {
	s.testProcessIndicatorIDs = make([]string, 0)

	processIndicators := fixtures.GetSACTestResourceSet(fixtures.GetScopedProcessIndicator)
	err := s.datastore.AddProcessIndicators(s.testContexts[sacTestUtils.UnrestrictedReadWriteCtx], processIndicators...)
	s.Require().NoError(err)

	for _, pi := range processIndicators {
		s.testProcessIndicatorIDs = append(s.testProcessIndicatorIDs, pi.GetId())
	}
}

func (s *processIndicatorDatastoreSACSuite) TearDownTest() {
	err := s.datastore.RemoveProcessIndicators(s.testContexts[sacTestUtils.UnrestrictedReadWriteCtx],
		s.testProcessIndicatorIDs)
	s.Require().NoError(err)
}

func (s *processIndicatorDatastoreSACSuite) deleteProcessIndicator(id string) {
	s.Require().NoError(s.datastore.RemoveProcessIndicators(s.testContexts[sacTestUtils.UnrestrictedReadWriteCtx],
		[]string{id}))
}

func (s *processIndicatorDatastoreSACSuite) TestAddProcessIndicators() {
	cases := sacTestUtils.GenericGlobalSACUpsertTestCases(s.T(), sacTestUtils.VerbAdd)

	for name, c := range cases {
		s.Run(name, func() {
			processIndicator := fixtures.GetScopedProcessIndicator(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testProcessIndicatorIDs = append(s.testProcessIndicatorIDs, processIndicator.GetId())
			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.AddProcessIndicators(ctx, processIndicator)
			defer s.deleteProcessIndicator(processIndicator.GetId())
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}

}

func (s *processIndicatorDatastoreSACSuite) TestGetProcessIndicator() {
	processIndicator := fixtures.GetScopedProcessIndicator(uuid.NewV4().String(), testconsts.Cluster2,
		testconsts.NamespaceB)
	err := s.datastore.AddProcessIndicators(s.testContexts[sacTestUtils.UnrestrictedReadWriteCtx], processIndicator)
	s.Require().NoError(err)
	s.testProcessIndicatorIDs = append(s.testProcessIndicatorIDs, processIndicator.GetId())

	cases := sacTestUtils.GenericNamespaceSACGetTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			res, found, err := s.datastore.GetProcessIndicator(ctx, processIndicator.GetId())
			s.Require().NoError(err)
			if c.ExpectedFound {
				s.True(found)
				s.Equal(*processIndicator, *res)
			} else {
				s.False(found)
				s.Nil(res)
			}
		})
	}
}

func (s *processIndicatorDatastoreSACSuite) TestRemoveProcessIndicators() {
	cases := sacTestUtils.GenericGlobalSACDeleteTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			processIndicator := fixtures.GetScopedProcessIndicator(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testProcessIndicatorIDs = append(s.testProcessIndicatorIDs, processIndicator.GetId())

			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.AddProcessIndicators(s.testContexts[sacTestUtils.UnrestrictedReadWriteCtx], processIndicator)
			s.Require().NoError(err)
			defer s.deleteProcessIndicator(processIndicator.GetId())

			err = s.datastore.RemoveProcessIndicators(ctx, []string{processIndicator.GetId()})
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *processIndicatorDatastoreSACSuite) TestScopedSearch() {
	for name, c := range sacTestUtils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *processIndicatorDatastoreSACSuite) TestUnrestrictedSearch() {
	for name, c := range sacTestUtils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *processIndicatorDatastoreSACSuite) TestScopeSearchRaw() {
	for name, c := range sacTestUtils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchRawTest(c)
		})
	}
}

func (s *processIndicatorDatastoreSACSuite) TestUnrestrictedSearchRaw() {
	for name, c := range sacTestUtils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchRawTest(c)
		})
	}
}

func (s *processIndicatorDatastoreSACSuite) runSearchRawTest(c sacTestUtils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.SearchRawProcessIndicators(ctx, searchPkg.NewQueryBuilder().AddStrings(searchPkg.ProcessID, searchPkg.WildcardString).ProtoQuery())
	s.Require().NoError(err)
	resultObjs := make([]sac.NamespaceScopedObject, 0, len(results))
	for i := range results {
		resultObjs = append(resultObjs, results[i])
	}
	resultCounts := sacTestUtils.CountSearchResultObjectsPerClusterAndNamespace(s.T(), resultObjs)
	sacTestUtils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)
}

func (s *processIndicatorDatastoreSACSuite) runSearchTest(c sacTestUtils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.Search(ctx, nil)
	s.Require().NoError(err)
	resultObjects := make([]sac.NamespaceScopedObject, 0, len(results))
	for _, r := range results {
		obj, found, err := s.datastore.GetProcessIndicator(s.testContexts[sacTestUtils.UnrestrictedReadCtx], r.ID)
		if found && err == nil {
			resultObjects = append(resultObjects, obj)
		}
	}
	resultCounts := sacTestUtils.CountSearchResultObjectsPerClusterAndNamespace(s.T(), resultObjects)
	sacTestUtils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)
}
