//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
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

func TestRiskDataStoreSAC(t *testing.T) {
	suite.Run(t, new(riskDatastoreSACSuite))
}

type riskDatastoreSACSuite struct {
	suite.Suite

	pool postgres.DB

	datastore DataStore

	optionsMap searchPkg.OptionsMap

	testContexts map[string]context.Context
	testRiskIDs  []string
}

func (s *riskDatastoreSACSuite) SetupSuite() {
	pgtestbase := pgtest.ForT(s.T())
	s.Require().NotNil(pgtestbase)
	s.pool = pgtestbase.DB
	s.datastore = GetTestPostgresDataStore(s.T(), s.pool)
	s.optionsMap = schema.RisksSchema.OptionsMap

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(),
		resources.DeploymentExtension)
}

func (s *riskDatastoreSACSuite) TearDownSuite() {
	s.pool.Close()
}

func (s *riskDatastoreSACSuite) SetupTest() {
	s.testRiskIDs = make([]string, 0)

	risks := fixtures.GetSACTestResourceSet(fixtures.GetScopedRisk)

	for i := range risks {
		err := s.datastore.UpsertRisk(s.testContexts[testutils.UnrestrictedReadWriteCtx], risks[i])
		s.Require().NoError(err)
	}

	for _, risk := range risks {
		s.testRiskIDs = append(s.testRiskIDs, risk.GetSubject().GetId())
	}
}

func (s *riskDatastoreSACSuite) TearDownTest() {
	for _, id := range s.testRiskIDs {
		s.deleteRisk(id)
	}
}

func (s *riskDatastoreSACSuite) deleteRisk(id string) {
	s.Require().NoError(s.datastore.RemoveRisk(s.testContexts[testutils.UnrestrictedReadWriteCtx], id,
		storage.RiskSubjectType_DEPLOYMENT))
}

func (s *riskDatastoreSACSuite) TestUpsertRisk() {
	cases := testutils.GenericGlobalSACUpsertTestCases(s.T(), testutils.VerbUpsert)

	for name, c := range cases {
		s.Run(name, func() {
			risk := fixtures.GetScopedRisk(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testRiskIDs = append(s.testRiskIDs, risk.GetSubject().GetId())
			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.UpsertRisk(ctx, risk)
			defer s.deleteRisk(risk.GetSubject().GetId())
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *riskDatastoreSACSuite) TestGetRisk() {
	risk := fixtures.GetScopedRisk(uuid.NewV4().String(), testconsts.Cluster2,
		testconsts.NamespaceB)
	err := s.datastore.UpsertRisk(s.testContexts[testutils.UnrestrictedReadWriteCtx], risk)
	s.Require().NoError(err)
	s.testRiskIDs = append(s.testRiskIDs, risk.GetSubject().GetId())

	cases := testutils.GenericGlobalSACGetTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			res, found, err := s.datastore.GetRisk(ctx, risk.GetSubject().GetId(), storage.RiskSubjectType_DEPLOYMENT)
			s.Require().NoError(err)
			if c.ExpectedFound {
				s.Require().True(found)
				s.Equal(*risk, *res)
			} else {
				s.False(found)
				s.Nil(res)
			}
		})
	}
}

func (s *riskDatastoreSACSuite) TestGetRiskForDeployment() {
	risk := fixtures.GetScopedRisk(uuid.NewV4().String(), testconsts.Cluster2,
		testconsts.NamespaceB)
	err := s.datastore.UpsertRisk(s.testContexts[testutils.UnrestrictedReadWriteCtx], risk)
	s.Require().NoError(err)
	s.testRiskIDs = append(s.testRiskIDs, risk.GetSubject().GetId())

	d := &storage.Deployment{
		Id:        risk.GetSubject().GetId(),
		ClusterId: testconsts.Cluster2,
		Namespace: testconsts.NamespaceB,
	}

	cases := testutils.GenericNamespaceSACGetTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			res, found, err := s.datastore.GetRiskForDeployment(ctx, d)
			s.Require().NoError(err)
			if c.ExpectedFound {
				s.Require().True(found)
				s.Equal(*risk, *res)
			} else {
				s.False(found)
				s.Nil(res)
			}
		})
	}
}

func (s *riskDatastoreSACSuite) TestRemoveRisk() {
	cases := testutils.GenericGlobalSACDeleteTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			risk := fixtures.GetScopedRisk(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testRiskIDs = append(s.testRiskIDs, risk.GetSubject().GetId())

			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.UpsertRisk(s.testContexts[testutils.UnrestrictedReadWriteCtx], risk)
			s.Require().NoError(err)
			defer s.deleteRisk(risk.GetId())

			err = s.datastore.RemoveRisk(ctx, risk.GetSubject().GetId(), storage.RiskSubjectType_DEPLOYMENT)
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *riskDatastoreSACSuite) runSearchRawTest(c testutils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.SearchRawRisks(ctx, nil)
	s.Require().NoError(err)
	resultObjs := make([]sac.NamespaceScopedObject, 0, len(results))
	for i := range results {
		resultObjs = append(resultObjs, results[i].Subject)
	}
	resultCounts := testutils.CountSearchResultObjectsPerClusterAndNamespace(s.T(), resultObjs)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)
}

func (s *riskDatastoreSACSuite) runSearchTest(c testutils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.Search(ctx, nil)
	s.Require().NoError(err)
	resultObjects := make([]sac.NamespaceScopedObject, 0, len(results))
	for _, r := range results {
		subjectType, subjectID, err := GetIDParts(r.ID)
		if err != nil {
			continue
		}
		obj, found, err := s.datastore.GetRisk(s.testContexts[testutils.UnrestrictedReadCtx], subjectID, subjectType)
		if found && err == nil {
			resultObjects = append(resultObjects, obj.GetSubject())
		}
	}
	resultCounts := testutils.CountSearchResultObjectsPerClusterAndNamespace(s.T(), resultObjects)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)
}

func (s *riskDatastoreSACSuite) TestScopedSearch() {
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *riskDatastoreSACSuite) TestUnrestrictedSearch() {
	for name, c := range testutils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *riskDatastoreSACSuite) TestScopeSearchRaw() {
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchRawTest(c)
		})
	}
}

func (s *riskDatastoreSACSuite) TestUnrestrictedSearchRaw() {
	for name, c := range testutils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchRawTest(c)
		})
	}
}
