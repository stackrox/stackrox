package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/processbaseline/index"
	baselineSearch "github.com/stackrox/rox/central/processbaseline/search"
	"github.com/stackrox/rox/central/processbaseline/store"
	postgresStore "github.com/stackrox/rox/central/processbaseline/store/postgres"
	rocksdbStore "github.com/stackrox/rox/central/processbaseline/store/rocksdb"
	"github.com/stackrox/rox/central/processbaselineresults/datastore/mocks"
	indicatorMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestProcessBaselineDatastore(t *testing.T) {
	suite.Run(t, new(ProcessBaselineDataStoreTestSuite))
}

type ProcessBaselineDataStoreTestSuite struct {
	suite.Suite
	requestContext     context.Context
	datastore          DataStore
	storage            store.Store
	indexer            index.Indexer
	searcher           baselineSearch.Searcher
	indicatorMockStore *indicatorMocks.MockDataStore

	db   *rocksdb.RocksDB
	pool *pgxpool.Pool

	baselineResultsStore *mocks.MockDataStore

	mockCtrl *gomock.Controller
}

func (suite *ProcessBaselineDataStoreTestSuite) SetupTest() {
	suite.requestContext = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.ProcessWhitelist),
		),
	)
	var err error

	if features.PostgresDatastore.Enabled() {
		pgtestbase := pgtest.ForT(suite.T())
		suite.Require().NotNil(pgtestbase)
		suite.pool = pgtestbase.Pool
		suite.storage = postgresStore.New(suite.pool)
		suite.indexer = postgresStore.NewIndexer(suite.pool)
	} else {
		suite.db, err = rocksdb.NewTemp(suite.T().Name() + ".db")
		suite.Require().NoError(err)

		suite.storage, err = rocksdbStore.New(suite.db)
		suite.NoError(err)

		tmpIndex, err := globalindex.TempInitializeIndices("")
		suite.NoError(err)
		suite.indexer = index.New(tmpIndex)
	}

	suite.searcher, err = baselineSearch.New(suite.storage, suite.indexer)
	suite.NoError(err)

	suite.mockCtrl = gomock.NewController(suite.T())

	suite.baselineResultsStore = mocks.NewMockDataStore(suite.mockCtrl)
	suite.indicatorMockStore = indicatorMocks.NewMockDataStore(suite.mockCtrl)
	suite.datastore = New(suite.storage, suite.indexer, suite.searcher, suite.baselineResultsStore, suite.indicatorMockStore)
}

func (suite *ProcessBaselineDataStoreTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
	if features.PostgresDatastore.Enabled() {
		suite.pool.Close()
	} else {
		rocksdbtest.TearDownRocksDB(suite.db)
	}
}

func (suite *ProcessBaselineDataStoreTestSuite) mustSerializeKey(key *storage.ProcessBaselineKey) string {
	serialized, err := keyToID(key)
	suite.Require().NoError(err)
	return serialized
}

func (suite *ProcessBaselineDataStoreTestSuite) createAndStoreBaseline(key *storage.ProcessBaselineKey) *storage.ProcessBaseline {
	baseline := fixtures.GetProcessBaseline()
	baseline.Key = key
	suite.NotNil(baseline)
	id, err := suite.datastore.AddProcessBaseline(suite.requestContext, baseline)
	suite.NoError(err)
	suite.NotNil(id)
	suite.NotNil(baseline.Created)
	suite.Equal(baseline.Created, baseline.LastUpdate)
	suite.True(baseline.StackRoxLockedTimestamp.Compare(baseline.Created) >= 0)

	suite.Equal(suite.mustSerializeKey(key), id)
	suite.Equal(id, baseline.Id)
	return baseline
}

func (suite *ProcessBaselineDataStoreTestSuite) createAndStoreBaselines(keys ...*storage.ProcessBaselineKey) []*storage.ProcessBaseline {
	baselines := make([]*storage.ProcessBaseline, len(keys))
	for i, key := range keys {
		baselines[i] = suite.createAndStoreBaseline(key)
	}
	return baselines
}

func (suite *ProcessBaselineDataStoreTestSuite) createAndStoreBaselineWithRandomKey() *storage.ProcessBaseline {
	return suite.createAndStoreBaseline(&storage.ProcessBaselineKey{
		DeploymentId:  uuid.NewV4().String(),
		ContainerName: uuid.NewV4().String(),
		ClusterId:     uuid.NewV4().String(),
		Namespace:     uuid.NewV4().String(),
	})
}

func (suite *ProcessBaselineDataStoreTestSuite) doGet(key *storage.ProcessBaselineKey, shouldExist bool, equals *storage.ProcessBaseline) *storage.ProcessBaseline {
	baseline, exists, err := suite.datastore.GetProcessBaseline(suite.requestContext, key)
	suite.NoError(err)
	if shouldExist {
		suite.True(exists)
		suite.NotNil(baseline)
		if equals != nil {
			suite.Equal(equals, baseline)
		}
	} else {
		suite.Nil(baseline)
		suite.False(exists)
	}
	return baseline
}

func (suite *ProcessBaselineDataStoreTestSuite) testUpdate(key *storage.ProcessBaselineKey, addProcesses []string, removeProcesses []string, auto bool, expectedResults set.StringSet) *storage.ProcessBaseline {
	updated, err := suite.datastore.UpdateProcessBaselineElements(suite.requestContext, key, fixtures.MakeBaselineItems(addProcesses...), fixtures.MakeBaselineItems(removeProcesses...), auto)
	suite.NoError(err)
	suite.NotNil(updated)
	suite.True(updated.GetLastUpdate().Compare(updated.GetCreated()) > 0)
	suite.NotNil(updated.Elements)
	suite.Equal(expectedResults.Cardinality(), len(updated.Elements))
	actualResults := set.NewStringSet()
	for _, process := range updated.Elements {
		actualResults.Add(process.GetElement().GetProcessName())
	}
	suite.Equal(expectedResults, actualResults)
	return updated
}

func (suite *ProcessBaselineDataStoreTestSuite) TestGetById() {
	suite.doGet(&storage.ProcessBaselineKey{DeploymentId: "FAKE", ContainerName: "whatever", ClusterId: "whatever", Namespace: "whatever"}, false, nil)

	key := &storage.ProcessBaselineKey{
		DeploymentId:  "blah",
		ContainerName: "container",
		ClusterId:     "cluster1",
		Namespace:     "namespace",
	}
	baseline := suite.createAndStoreBaseline(key)
	suite.doGet(key, true, baseline)
}

func (suite *ProcessBaselineDataStoreTestSuite) TestRemoveProcessBaseline() {
	baseline := suite.createAndStoreBaselineWithRandomKey()
	key := baseline.GetKey()
	suite.doGet(baseline.GetKey(), true, baseline)
	suite.baselineResultsStore.EXPECT().DeleteBaselineResults(suite.requestContext, key.GetDeploymentId()).Return(nil)
	err := suite.datastore.RemoveProcessBaseline(suite.requestContext, key)
	suite.NoError(err)
	suite.doGet(key, false, nil)
}

func (suite *ProcessBaselineDataStoreTestSuite) TestLockAndUnlockBaseline() {
	baseline := suite.createAndStoreBaselineWithRandomKey()
	key := baseline.GetKey()
	suite.Nil(baseline.GetUserLockedTimestamp())
	updatedBaseline, err := suite.datastore.UserLockProcessBaseline(suite.requestContext, key, true)
	suite.NoError(err)
	suite.NotNil(updatedBaseline.GetUserLockedTimestamp())
	suite.doGet(key, true, updatedBaseline)
	suite.True(updatedBaseline.GetLastUpdate().Compare(updatedBaseline.GetCreated()) > 0)

	updatedBaseline, err = suite.datastore.UserLockProcessBaseline(suite.requestContext, key, false)
	suite.NoError(err)
	suite.Nil(updatedBaseline.GetUserLockedTimestamp())
	suite.doGet(key, true, updatedBaseline)
	suite.True(updatedBaseline.GetLastUpdate().Compare(updatedBaseline.GetCreated()) > 0)
}

func (suite *ProcessBaselineDataStoreTestSuite) TestUpdateProcessBaseline() {
	baseline := fixtures.GetProcessBaselineWithKey()
	baseline.Elements = nil // Fixture gives a single process but we want to test updates
	suite.NotNil(baseline)
	key := baseline.GetKey()
	id, err := suite.datastore.AddProcessBaseline(suite.requestContext, baseline)
	suite.NoError(err)
	suite.NotNil(id)

	processName := []string{"Some process name"}
	processNameSet := set.NewStringSet(processName...)
	otherProcess := []string{"Some other process"}
	otherProcessSet := set.NewStringSet(otherProcess...)
	updated := suite.testUpdate(key, processName, nil, true, processNameSet)
	suite.True(updated.Elements[0].Auto)

	updated = suite.testUpdate(key, processName, nil, false, processNameSet)
	suite.False(updated.Elements[0].Auto)

	updated = suite.testUpdate(key, otherProcess, processName, true, otherProcessSet)
	suite.True(updated.Elements[0].Auto)

	multiAdd := []string{"a", "b", "c"}
	multiAddExpected := set.NewStringSet(multiAdd...)
	updated = suite.testUpdate(key, multiAdd, otherProcess, false, multiAddExpected)
	for _, process := range updated.Elements {
		suite.False(process.Auto)
	}
}

func (suite *ProcessBaselineDataStoreTestSuite) TestUpsertProcessBaseline() {
	key := fixtures.GetBaselineKey()
	firstProcess := "Joseph Rules"
	newItem := []*storage.BaselineItem{{Item: &storage.BaselineItem_ProcessName{ProcessName: firstProcess}}}
	baseline, err := suite.datastore.UpsertProcessBaseline(suite.requestContext, key, newItem, true, false)
	suite.NoError(err)
	suite.Equal(1, len(baseline.GetElements()))
	suite.Equal(firstProcess, baseline.GetElements()[0].GetElement().GetProcessName())
	suite.Equal(key, baseline.GetKey())
	suite.True(baseline.GetLastUpdate().Compare(baseline.GetCreated()) == 0)

	secondProcess := "Joseph is the Best"
	newItem = []*storage.BaselineItem{{Item: &storage.BaselineItem_ProcessName{ProcessName: secondProcess}}}
	baseline, err = suite.datastore.UpsertProcessBaseline(suite.requestContext, key, newItem, true, false)
	suite.NoError(err)
	suite.Equal(2, len(baseline.GetElements()))
	processNames := make([]string, 0, 2)
	for _, element := range baseline.GetElements() {
		processNames = append(processNames, element.GetElement().GetProcessName())
	}
	suite.ElementsMatch([]string{firstProcess, secondProcess}, processNames)
	suite.Equal(key, baseline.GetKey())
	suite.True(baseline.GetLastUpdate().Compare(baseline.GetCreated()) > 0)
}

func makeItemList(elementList []*storage.BaselineElement) []*storage.BaselineItem {
	itemList := make([]*storage.BaselineItem, len(elementList))
	for i, element := range elementList {
		itemList[i] = element.GetElement()
	}
	return itemList
}

func (suite *ProcessBaselineDataStoreTestSuite) TestGraveyard() {
	baseline := suite.createAndStoreBaselineWithRandomKey()
	itemList := makeItemList(baseline.GetElements())
	suite.NotEmpty(itemList)
	suite.Empty(baseline.GetElementGraveyard())
	updatedBaseline, err := suite.datastore.UpdateProcessBaselineElements(suite.requestContext, baseline.GetKey(), nil, itemList, true)
	// The elements should have been removed from the process baseline and put in the graveyard
	suite.NoError(err)
	suite.ElementsMatch(baseline.GetElements(), updatedBaseline.GetElementGraveyard())

	updatedBaseline, err = suite.datastore.UpdateProcessBaselineElements(suite.requestContext, baseline.GetKey(), itemList, nil, true)
	suite.NoError(err)
	// The elements should NOT be added back on to the process baseline because they are in the graveyard and auto = true
	suite.Empty(updatedBaseline.GetElements())
	suite.ElementsMatch(baseline.GetElements(), updatedBaseline.GetElementGraveyard())

	updatedBaseline, err = suite.datastore.UpdateProcessBaselineElements(suite.requestContext, baseline.GetKey(), itemList, nil, false)
	suite.NoError(err)
	// The elements SHOULD be added back on to the process baseline because auto = false
	suite.Empty(updatedBaseline.GetElementGraveyard())
	updatedItems := makeItemList(updatedBaseline.GetElements())
	suite.ElementsMatch(itemList, updatedItems)
}

func (suite *ProcessBaselineDataStoreTestSuite) doQuery(q *v1.Query, len int) {
	result, err := suite.datastore.SearchRawProcessBaselines(suite.requestContext, q)
	suite.NoError(err)
	suite.Len(result, len)
}

func (suite *ProcessBaselineDataStoreTestSuite) TestRemoveByDeployment() {
	dep1 := "1"
	key1 := &storage.ProcessBaselineKey{DeploymentId: dep1, ContainerName: "1", ClusterId: "1", Namespace: "1"}
	key2 := &storage.ProcessBaselineKey{DeploymentId: dep1, ContainerName: "2", ClusterId: "1", Namespace: "2"}
	key3 := &storage.ProcessBaselineKey{DeploymentId: "2", ContainerName: "1", ClusterId: "1", Namespace: "3"}
	suite.createAndStoreBaselines(key1, key2, key3)

	queryDep1 := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.DeploymentID, dep1).ProtoQuery()
	queryDep2 := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.DeploymentID, "2").ProtoQuery()
	suite.doQuery(queryDep1, 2)
	suite.doQuery(queryDep2, 1)
	suite.doGet(key1, true, nil)
	suite.doGet(key2, true, nil)
	suite.doGet(key3, true, nil)

	suite.baselineResultsStore.EXPECT().DeleteBaselineResults(suite.requestContext, dep1).Return(nil)
	err := suite.datastore.RemoveProcessBaselinesByDeployment(suite.requestContext, dep1)
	suite.NoError(err)

	suite.doQuery(queryDep1, 0)
	suite.doQuery(queryDep2, 1)
	suite.doGet(key1, false, nil)
	suite.doGet(key2, false, nil)
	suite.doGet(key3, true, nil)
}

func (suite *ProcessBaselineDataStoreTestSuite) TestIDToKeyConversion() {
	key := &storage.ProcessBaselineKey{
		DeploymentId:  "blah",
		ContainerName: "container",
		ClusterId:     "cluster1",
		Namespace:     "namespace",
	}

	id, err := keyToID(key)
	suite.NoError(err)
	resKey, err := IDToKey(id)
	suite.NoError(err)
	suite.NotNil(resKey)
	suite.Equal(*key, *resKey)
}

func (suite *ProcessBaselineDataStoreTestSuite) TestBuildUnlockedProcessBaseline() {
	key := fixtures.GetBaselineKey()
	indicators :=
		[]*storage.ProcessIndicator{
			{
				Signal: &storage.ProcessSignal{
					ExecFilePath: "/bin/not-apt-get",
					Args:         "install nmap",
				},
				ContainerName: key.GetContainerName(),
			},
			{
				Signal: &storage.ProcessSignal{
					ExecFilePath: "/bin/apt-get",
					Args:         "install nmap",
				},
				ContainerName: key.GetContainerName(),
			},
			{
				Signal: &storage.ProcessSignal{
					ExecFilePath: "/bin/curl",
					Args:         "badssl.com",
				},
				ContainerName: key.GetContainerName(),
			},
		}

	suite.indicatorMockStore.EXPECT().SearchRawProcessIndicators(suite.requestContext, gomock.Any()).Return(indicators, nil)

	baseline, err := suite.datastore.CreateUnlockedProcessBaseline(suite.requestContext, key)
	suite.NoError(err)

	suite.Equal(key, baseline.GetKey())
	suite.True(baseline.GetLastUpdate().Compare(baseline.GetCreated()) == 0)
	suite.True(baseline.UserLockedTimestamp == nil)
	suite.True(baseline.Elements != nil)

}

func (suite *ProcessBaselineDataStoreTestSuite) TestBuildUnlockedProcessBaselineDupesRemoved() {
	key := fixtures.GetBaselineKey()
	indicators :=
		[]*storage.ProcessIndicator{
			{
				Signal: &storage.ProcessSignal{
					ExecFilePath: "/bin/not-apt-get",
					Args:         "install nmap",
				},
				ContainerName: key.GetContainerName(),
			},
			{
				Signal: &storage.ProcessSignal{
					ExecFilePath: "/bin/apt-get",
					Args:         "install nmap",
				},
				ContainerName: key.GetContainerName(),
			},
			{
				Signal: &storage.ProcessSignal{
					ExecFilePath: "/bin/curl",
					Args:         "badssl.com",
				},
				ContainerName: key.GetContainerName(),
			},
			{
				Signal: &storage.ProcessSignal{
					ExecFilePath: "/bin/apt-get",
					Args:         "install nmap",
				},
				ContainerName: key.GetContainerName(),
			},
			{
				Signal: &storage.ProcessSignal{
					ExecFilePath: "/bin/curl",
					Args:         "badssl.com",
				},
				ContainerName: key.GetContainerName(),
			},
		}

	suite.indicatorMockStore.EXPECT().SearchRawProcessIndicators(suite.requestContext, gomock.Any()).Return(indicators, nil)

	baseline, err := suite.datastore.CreateUnlockedProcessBaseline(suite.requestContext, key)
	suite.NoError(err)

	suite.Equal(key, baseline.GetKey())
	suite.True(baseline.GetLastUpdate().Compare(baseline.GetCreated()) == 0)
	suite.True(baseline.UserLockedTimestamp == nil)
	suite.True(baseline.Elements != nil)
	suite.True(len(baseline.Elements) == len(indicators)-2)

}

func (suite *ProcessBaselineDataStoreTestSuite) TestBuildUnlockedProcessBaselineNoProcesses() {
	key := fixtures.GetBaselineKey()

	suite.indicatorMockStore.EXPECT().SearchRawProcessIndicators(suite.requestContext, gomock.Any())

	baseline, err := suite.datastore.CreateUnlockedProcessBaseline(suite.requestContext, key)
	suite.NoError(err)

	suite.Equal(key, baseline.GetKey())
	suite.True(baseline.GetLastUpdate().Compare(baseline.GetCreated()) == 0)
	suite.True(baseline.UserLockedTimestamp == nil)
	suite.True(baseline.Elements == nil || len(baseline.Elements) == 0)

}

func (suite *ProcessBaselineDataStoreTestSuite) TestClearProcessBaselines() {
	key := fixtures.GetBaselineKey()
	baseline := suite.createAndStoreBaseline(key)
	suite.True(baseline.Elements != nil)

	ids := []string{baseline.Id}
	err := suite.datastore.ClearProcessBaselines(suite.requestContext, ids)
	suite.True(err == nil)
	baseline, exists, err := suite.datastore.GetProcessBaseline(suite.requestContext, key)
	suite.True(exists)
	suite.True(baseline.Elements == nil)
	suite.True(err == nil)
}
