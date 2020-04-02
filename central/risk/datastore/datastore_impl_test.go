package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/risk/datastore/internal/index"
	"github.com/stackrox/rox/central/risk/datastore/internal/search"
	"github.com/stackrox/rox/central/risk/datastore/internal/store"
	"github.com/stackrox/rox/central/risk/datastore/internal/store/bolt"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestRiskDataStore(t *testing.T) {
	suite.Run(t, new(RiskDataStoreTestSuite))
}

type RiskDataStoreTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index

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

	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	suite.Require().NoError(err)

	suite.storage, _ = bolt.New(db)
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
}

func (suite *RiskDataStoreTestSuite) TestRiskDataStore() {
	risk := fixtures.GetRisk()
	err := suite.datastore.UpsertRisk(suite.hasWriteCtx, risk)
	suite.Require().NoError(err)

	result, found, err := suite.datastore.GetRisk(suite.hasReadCtx, risk.GetSubject().GetId(), risk.GetSubject().GetType())
	suite.Require().NoError(err)
	suite.Require().True(found)
	suite.Require().NotNil(result)

	err = suite.datastore.RemoveRisk(suite.hasWriteCtx, risk.GetSubject().GetId(), risk.GetSubject().GetType())
	suite.Require().NoError(err)

	result, found, err = suite.datastore.GetRisk(suite.hasReadCtx, risk.GetSubject().GetId(), risk.GetSubject().GetType())
	suite.Require().NoError(err)
	suite.Require().False(found)
	suite.Require().Nil(result)
}
