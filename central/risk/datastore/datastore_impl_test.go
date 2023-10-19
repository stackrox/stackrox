//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/risk/datastore/internal/index"
	"github.com/stackrox/rox/central/risk/datastore/internal/search"
	"github.com/stackrox/rox/central/risk/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestRiskDataStore(t *testing.T) {
	suite.Run(t, new(RiskDataStoreTestSuite))
}

type RiskDataStoreTestSuite struct {
	suite.Suite

	indexer   index.Indexer
	searcher  search.Searcher
	storage   store.Store
	datastore DataStore

	pool postgres.DB

	optionsMap searchPkg.OptionsMap

	hasReadCtx  context.Context
	hasWriteCtx context.Context
}

func (suite *RiskDataStoreTestSuite) SetupSuite() {
	pgtestbase := pgtest.ForT(suite.T())
	suite.Require().NotNil(pgtestbase)
	suite.pool = pgtestbase.DB
	suite.datastore = GetTestPostgresDataStore(suite.T(), suite.pool)

	suite.optionsMap = schema.RisksSchema.OptionsMap

	suite.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension)))
	suite.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension)))
}

func (suite *RiskDataStoreTestSuite) TearDownSuite() {
	suite.pool.Close()
}

func (suite *RiskDataStoreTestSuite) TestRiskDataStore() {
	risk := fixtures.GetRisk()
	deployment := &storage.Deployment{
		Id:        risk.GetSubject().GetId(),
		Namespace: risk.GetSubject().GetNamespace(),
		ClusterId: risk.GetSubject().GetClusterId(),
	}

	testCases := map[string]func() (*storage.Risk, bool, error){
		"GetRisk": func() (*storage.Risk, bool, error) {
			return suite.datastore.GetRisk(suite.hasReadCtx, risk.GetSubject().GetId(), risk.GetSubject().GetType())
		},
		"GetRiskForDeployment": func() (*storage.Risk, bool, error) {
			return suite.datastore.GetRiskForDeployment(suite.hasReadCtx, deployment)
		},
	}
	for name, getRisk := range testCases {
		suite.Run(name, func() {
			err := suite.datastore.UpsertRisk(suite.hasWriteCtx, risk)
			suite.Require().NoError(err)

			result, found, err := getRisk()
			suite.Require().NoError(err)
			suite.Require().True(found)
			suite.Require().NotNil(result)

			err = suite.datastore.RemoveRisk(suite.hasWriteCtx, risk.GetSubject().GetId(), risk.GetSubject().GetType())
			suite.Require().NoError(err)

			result, found, err = getRisk()
			suite.Require().NoError(err)
			suite.Require().False(found)
			suite.Require().Nil(result)

		})
	}

	scopedAccess := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension),
			sac.ClusterScopeKeys(fixtureconsts.Cluster1),
			sac.NamespaceScopeKeys(fixtureconsts.Namespace1)))

	scopedAccessForDifferentNamespace := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension),
			sac.ClusterScopeKeys(fixtureconsts.Cluster1),
			sac.NamespaceScopeKeys("DifferentNS")))

	suite.Run("GetRiskForDeployment with scoped access", func() {
		err := suite.datastore.UpsertRisk(suite.hasWriteCtx, risk)
		suite.Require().NoError(err)

		result, found, err := suite.datastore.GetRiskForDeployment(scopedAccess, deployment)
		suite.Require().NoError(err)
		suite.Require().True(found)
		suite.Require().NotNil(result)
	})

	testCasesForScopedAccess := map[string]func() (*storage.Risk, bool, error){
		"GetRiskForDeployment with access to different namespace": func() (*storage.Risk, bool, error) {
			return suite.datastore.GetRiskForDeployment(scopedAccessForDifferentNamespace, deployment)
		},
		"GetRisk with scoped access": func() (*storage.Risk, bool, error) {
			return suite.datastore.GetRisk(scopedAccess, risk.GetSubject().GetId(), risk.GetSubject().GetType())
		},
		"GetRisk with scoped access for different namespace": func() (*storage.Risk, bool, error) {
			return suite.datastore.GetRisk(scopedAccessForDifferentNamespace, risk.GetSubject().GetId(), risk.GetSubject().GetType())
		},
	}
	for name, getRisk := range testCasesForScopedAccess {
		suite.Run(name, func() {
			err := suite.datastore.UpsertRisk(suite.hasWriteCtx, risk)
			suite.Require().NoError(err)

			result, found, err := getRisk()
			suite.Require().NoError(err)
			suite.Require().False(found)
			suite.Require().Nil(result)
		})
	}
}
