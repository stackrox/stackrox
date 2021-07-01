package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/risk/datastore/internal/index"
	"github.com/stackrox/rox/central/risk/datastore/internal/search"
	"github.com/stackrox/rox/central/risk/datastore/internal/store"
	rocksdbStore "github.com/stackrox/rox/central/risk/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestRiskDataStore(t *testing.T) {
	suite.Run(t, new(RiskDataStoreTestSuite))
}

type RiskDataStoreTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index

	db *rocksdb.RocksDB

	indexer   index.Indexer
	searcher  search.Searcher
	storage   store.Store
	datastore DataStore

	hasReadCtx  context.Context
	hasWriteCtx context.Context
}

func (suite *RiskDataStoreTestSuite) SetupSuite() {
	var err error
	suite.bleveIndex, err = globalindex.TempInitializeIndices("")
	suite.Require().NoError(err)

	db, err := rocksdb.NewTemp(suite.T().Name() + ".db")
	suite.Require().NoError(err)

	suite.db = db

	suite.storage = rocksdbStore.New(db)
	suite.indexer = index.New(suite.bleveIndex)
	suite.searcher = search.New(suite.storage, suite.indexer)
	suite.datastore, err = New(suite.storage, suite.indexer, suite.searcher)
	suite.Require().NoError(err)

	suite.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Risk)))
	suite.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Risk)))
}

func (suite *RiskDataStoreTestSuite) TearDownSuite() {
	suite.NoError(suite.bleveIndex.Close())
	rocksdbtest.TearDownRocksDB(suite.db)
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
			sac.ResourceScopeKeys(resources.Risk),
			sac.ClusterScopeKeys("FakeClusterID"),
			sac.NamespaceScopeKeys("FakeNS")))

	scopedAccessForDifferentNamespace := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Risk),
			sac.ClusterScopeKeys("FakeClusterID"),
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
