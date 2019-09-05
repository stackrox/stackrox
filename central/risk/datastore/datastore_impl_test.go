package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/central/risk/datastore/internal/index"
	"github.com/stackrox/rox/central/risk/datastore/internal/search"
	"github.com/stackrox/rox/central/risk/datastore/internal/store"
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

	suite.storage, _ = store.New(db)
	suite.indexer = index.New(suite.bleveIndex)
	suite.searcher = search.New(suite.storage, suite.indexer)
	suite.datastore, err = New(suite.storage, suite.indexer, suite.searcher)
	suite.Require().NoError(err)

	suite.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Risk, resources.Deployment)))
	suite.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Risk)))
}

func (suite *RiskDataStoreTestSuite) TearDownSuite() {
	suite.NoError(suite.bleveIndex.Close())
}

func (suite *RiskDataStoreTestSuite) TestRiskDataStore() {
	risk := fixtures.GetRisk()
	err := suite.datastore.UpsertRisk(suite.hasWriteCtx, risk)
	suite.Require().NoError(err)

	result, found, err := suite.datastore.GetRisk(suite.hasReadCtx, risk.GetEntity().GetId(), risk.GetEntity().GetType(), true)
	suite.Require().NoError(err)
	suite.Require().True(found)
	suite.Require().NotNil(result)

	err = suite.datastore.RemoveRisk(suite.hasWriteCtx, risk.GetEntity().GetId(), risk.GetEntity().GetType())
	suite.Require().NoError(err)

	result, found, _ = suite.datastore.GetRisk(suite.hasReadCtx, risk.GetEntity().GetId(), risk.GetEntity().GetType(), true)
	suite.Require().False(found)
	suite.Require().Nil(result)
}

func (suite *RiskDataStoreTestSuite) TestRiskAggregation() {
	risks := []*storage.Risk{
		{
			Score: 10,
			Entity: &storage.RiskEntityMeta{
				Id:        "FakeID1",
				Namespace: "FakeNS1",
				ClusterId: "FakeClusterID",
				Type:      storage.RiskEntityType_DEPLOYMENT,
			},
			Results: []*storage.Risk_Result{
				{Name: "BLAH"},
			},
		},
		{
			Score: 4,
			Entity: &storage.RiskEntityMeta{
				Id:        "FakeID2",
				Namespace: "FakeNS2",
				ClusterId: "FakeClusterID",
				Type:      storage.RiskEntityType_DEPLOYMENT,
			},
			Results: []*storage.Risk_Result{
				{Name: "BLAH"},
			},
		},
		{
			Score: 2,
			Entity: &storage.RiskEntityMeta{
				Id:        "FakeID3",
				Namespace: "FakeNS",
				ClusterId: "FakeClusterID",
				Type:      storage.RiskEntityType_DEPLOYMENT,
			},
			Results: []*storage.Risk_Result{
				{Name: "BLAH1"},
				{Name: "BLAH2"},
			},
		},
		{
			Score: 2,
			Entity: &storage.RiskEntityMeta{
				Id:        "FakeID",
				Namespace: "FakeNS",
				ClusterId: "FakeClusterID1",
				Type:      storage.RiskEntityType_DEPLOYMENT,
			},
			Results: []*storage.Risk_Result{},
		},
	}

	for _, risk := range risks {
		err := suite.datastore.UpsertRisk(suite.hasWriteCtx, risk)
		suite.Require().NoError(err)
	}

	for _, risk := range risks {
		result, found, err := suite.datastore.GetRisk(suite.hasReadCtx, risk.GetEntity().GetId(), risk.GetEntity().GetType(), true)
		suite.Require().NoError(err)
		suite.Require().True(found)
		suite.Require().NotNil(result)
	}

	actualRisk, found, err := suite.datastore.GetRisk(suite.hasReadCtx, "FakeClusterID", storage.RiskEntityType_CLUSTER, true)
	suite.Require().NoError(err)
	suite.Require().True(found)
	suite.Require().NotNil(actualRisk)
	suite.Require().Empty(actualRisk.GetResults())
	suite.Require().EqualValues(float32(16), actualRisk.GetAggregateScore())
}

func (suite *RiskDataStoreTestSuite) TestRankerUpdates() {
	parentRisk1 := &storage.Risk{
		Id:    "deployment:parent1",
		Score: 10,
		Entity: &storage.RiskEntityMeta{
			Id:   "parent1",
			Type: storage.RiskEntityType_DEPLOYMENT,
		},
		Results: []*storage.Risk_Result{
			{Name: "BLAH"},
		},
	}
	parentRisk2 := &storage.Risk{
		Id:    "deployment:parent2",
		Score: 4,
		Entity: &storage.RiskEntityMeta{
			Id:   "parent2",
			Type: storage.RiskEntityType_DEPLOYMENT,
		},
		Results: []*storage.Risk_Result{
			{Name: "BLAH"},
		},
	}
	childRisk1 := &storage.Risk{
		Id:    "deployment:child1",
		Score: 2,
		Entity: &storage.RiskEntityMeta{
			Id:   "child1",
			Type: storage.RiskEntityType_DEPLOYMENT,
		},
		Results: []*storage.Risk_Result{
			{Name: "BLAH1"},
			{Name: "BLAH2"},
		},
	}
	childRisk2 := &storage.Risk{
		Id:    "deployment:child2",
		Score: 2,
		Entity: &storage.RiskEntityMeta{
			Id:   "child2",
			Type: storage.RiskEntityType_DEPLOYMENT,
		},
		Results: []*storage.Risk_Result{},
	}

	suite.datastore.AddRiskDependencies(parentRisk1.GetId(), childRisk1.GetId())
	suite.datastore.AddRiskDependencies(parentRisk2.GetId(), childRisk1.GetId(), childRisk2.GetId())

	err := suite.datastore.UpsertRisk(suite.hasWriteCtx, parentRisk1)
	suite.Require().NoError(err)
	err = suite.datastore.UpsertRisk(suite.hasWriteCtx, parentRisk2)
	suite.Require().NoError(err)
	err = suite.datastore.UpsertRisk(suite.hasWriteCtx, childRisk1)
	suite.Require().NoError(err)
	err = suite.datastore.UpsertRisk(suite.hasWriteCtx, childRisk2)
	suite.Require().NoError(err)

	deploymentRanker := ranking.DeploymentRanker()

	rank := deploymentRanker.GetRankForID(parentRisk1.GetEntity().GetId())
	suite.EqualValues(int64(1), rank)
	rank = deploymentRanker.GetRankForID(parentRisk2.GetEntity().GetId())
	suite.EqualValues(int64(2), rank)
	rank = deploymentRanker.GetRankForID(childRisk1.GetEntity().GetId())
	suite.EqualValues(int64(3), rank)
	rank = deploymentRanker.GetRankForID(childRisk2.GetEntity().GetId())
	suite.EqualValues(int64(3), rank)

	score := deploymentRanker.GetScoreForID(parentRisk1.GetEntity().GetId())
	suite.EqualValues(12, score)
	score = deploymentRanker.GetScoreForID(parentRisk2.GetEntity().GetId())
	suite.EqualValues(8, score)
	score = deploymentRanker.GetScoreForID(childRisk1.GetEntity().GetId())
	suite.EqualValues(2, score)
	score = deploymentRanker.GetScoreForID(childRisk2.GetEntity().GetId())
	suite.EqualValues(2, score)
}
