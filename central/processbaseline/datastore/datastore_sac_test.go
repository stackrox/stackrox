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
	"github.com/stackrox/rox/pkg/sac/testutils"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestProcessBaselineDatastoreSAC(t *testing.T) {
	suite.Run(t, new(processBaselineSACTestSuite))
}

type processBaselineSACTestSuite struct {
	suite.Suite

	pool postgres.DB

	datastore DataStore

	optionsMap searchPkg.OptionsMap

	testContexts map[string]context.Context

	testProcessBaselineIDs []string
}

func (s *processBaselineSACTestSuite) SetupSuite() {
	var err error
	pgtestbase := pgtest.ForT(s.T())
	s.Require().NotNil(pgtestbase)
	s.pool = pgtestbase.DB
	s.datastore, err = GetTestPostgresDataStore(s.T(), s.pool)
	s.Require().NoError(err)
	s.optionsMap = schema.ProcessBaselinesSchema.OptionsMap

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(),
		resources.DeploymentExtension)
}

func (s *processBaselineSACTestSuite) TearDownSuite() {
	s.pool.Close()
}

func (s *processBaselineSACTestSuite) SetupTest() {
	s.testProcessBaselineIDs = make([]string, 0)

	processBaselines := fixtures.GetSACTestResourceSet(fixtures.GetScopedProcessBaseline)

	for i := range processBaselines {
		_, err := s.datastore.AddProcessBaseline(s.testContexts[testutils.UnrestrictedReadWriteCtx],
			processBaselines[i])
		s.Require().NoError(err)
	}

	for _, rb := range processBaselines {
		s.testProcessBaselineIDs = append(s.testProcessBaselineIDs, rb.GetId())
	}
}

func (s *processBaselineSACTestSuite) TearDownTest() {
	s.Require().NoError(s.datastore.RemoveProcessBaselinesByIDs(s.testContexts[testutils.UnrestrictedReadWriteCtx],
		s.testProcessBaselineIDs))
}

func (s *processBaselineSACTestSuite) deleteProcessBaseline(id string) {
	if id != "" {
		s.Require().NoError(s.datastore.RemoveProcessBaselinesByIDs(s.testContexts[testutils.UnrestrictedReadWriteCtx],
			[]string{id}))
	}
}

func (s *processBaselineSACTestSuite) TestAddProcessBaseline() {
	cases := testutils.GenericNamespaceSACUpsertTestCases(s.T(), testutils.VerbAdd)

	for name, c := range cases {
		s.Run(name, func() {
			processBaseline := fixtures.GetScopedProcessBaseline(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			ctx := s.testContexts[c.ScopeKey]
			key, err := s.datastore.AddProcessBaseline(ctx, processBaseline)
			defer s.deleteProcessBaseline(key)
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *processBaselineSACTestSuite) TestUpsertProcessBaseline() {
	cases := testutils.GenericNamespaceSACUpsertTestCases(s.T(), testutils.VerbUpsert)

	for name, c := range cases {
		s.Run(name, func() {
			processBaseline := fixtures.GetScopedProcessBaseline(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			ctx := s.testContexts[c.ScopeKey]
			processBaseline, err := s.datastore.UpsertProcessBaseline(ctx, processBaseline.GetKey(), nil, false, false)
			defer s.deleteProcessBaseline(processBaseline.GetId())
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *processBaselineSACTestSuite) TestUpdateProcessBaselineElements() {
	processBaseline := fixtures.GetScopedProcessBaseline(uuid.NewV4().String(), testconsts.Cluster2,
		testconsts.NamespaceB)
	_, err := s.datastore.AddProcessBaseline(s.testContexts[testutils.UnrestrictedReadWriteCtx], processBaseline)
	s.Require().NoError(err)
	s.testProcessBaselineIDs = append(s.testProcessBaselineIDs, processBaseline.GetId())

	cases := testutils.GenericNamespaceSACUpsertTestCases(s.T(), testutils.VerbUpdate)

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			_, err := s.datastore.UpdateProcessBaselineElements(
				ctx, processBaseline.GetKey(), nil, nil, false)
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *processBaselineSACTestSuite) TestGetProcessBaseline() {
	processBaseline := fixtures.GetScopedProcessBaseline(uuid.NewV4().String(), testconsts.Cluster2,
		testconsts.NamespaceB)
	_, err := s.datastore.AddProcessBaseline(s.testContexts[testutils.UnrestrictedReadWriteCtx], processBaseline)
	s.Require().NoError(err)
	s.testProcessBaselineIDs = append(s.testProcessBaselineIDs, processBaseline.GetId())

	cases := testutils.GenericNamespaceSACGetTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			res, found, err := s.datastore.GetProcessBaseline(ctx, processBaseline.GetKey())
			s.Require().NoError(err)
			if c.ExpectedFound {
				s.Require().True(found)
				s.Equal(*processBaseline, *res)
			} else {
				s.False(found)
				s.Nil(res)
			}
		})
	}
}

func (s *processBaselineSACTestSuite) TestRemoveProcessBaseline() {
	cases := testutils.GenericNamespaceSACDeleteTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			processBaseline := fixtures.GetScopedProcessBaseline(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			_, err := s.datastore.AddProcessBaseline(s.testContexts[testutils.UnrestrictedReadWriteCtx], processBaseline)
			s.Require().NoError(err)
			s.testProcessBaselineIDs = append(s.testProcessBaselineIDs, processBaseline.GetId())
			defer s.deleteProcessBaseline(processBaseline.GetId())

			ctx := s.testContexts[c.ScopeKey]
			err = s.datastore.RemoveProcessBaseline(ctx, processBaseline.GetKey())
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *processBaselineSACTestSuite) runSearchRawTest(c testutils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.SearchRawProcessBaselines(ctx, nil)
	s.Require().NoError(err)
	resultObjs := make([]sac.NamespaceScopedObject, 0, len(results))
	for i := range results {
		resultObjs = append(resultObjs, results[i].GetKey())
	}
	resultCounts := testutils.CountSearchResultObjectsPerClusterAndNamespace(s.T(), resultObjs)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)
}

func (s *processBaselineSACTestSuite) runSearchTest(c testutils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.Search(ctx, nil)
	s.Require().NoError(err)
	resultObjects := make([]sac.NamespaceScopedObject, 0, len(results))
	for _, r := range results {
		key, err := IDToKey(r.ID)
		if err != nil {
			continue
		}
		resultObjects = append(resultObjects, key)
	}
	resultCounts := testutils.CountSearchResultObjectsPerClusterAndNamespace(s.T(), resultObjects)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)
}

func (s *processBaselineSACTestSuite) TestScopedSearch() {
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *processBaselineSACTestSuite) TestUnrestrictedSearch() {
	for name, c := range testutils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *processBaselineSACTestSuite) TestScopeSearchRaw() {
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchRawTest(c)
		})
	}
}

func (s *processBaselineSACTestSuite) TestUnrestrictedSearchRaw() {
	for name, c := range testutils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchRawTest(c)
		})
	}
}
