package datastore

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/processindicator"
	"github.com/stackrox/rox/central/processindicator/index"
	indexMocks "github.com/stackrox/rox/central/processindicator/index/mocks"
	"github.com/stackrox/rox/central/processindicator/pruner"
	prunerMocks "github.com/stackrox/rox/central/processindicator/pruner/mocks"
	processSearch "github.com/stackrox/rox/central/processindicator/search"
	searchMocks "github.com/stackrox/rox/central/processindicator/search/mocks"
	"github.com/stackrox/rox/central/processindicator/store"
	badgerStore "github.com/stackrox/rox/central/processindicator/store/badger"
	storeMocks "github.com/stackrox/rox/central/processindicator/store/mocks"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestIndicatorDatastore(t *testing.T) {
	suite.Run(t, new(IndicatorDataStoreTestSuite))
}

type IndicatorDataStoreTestSuite struct {
	suite.Suite
	datastore DataStore
	storage   store.Store
	indexer   index.Indexer
	searcher  processSearch.Searcher

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	mockCtrl *gomock.Controller
}

func (suite *IndicatorDataStoreTestSuite) SetupTest() {
	suite.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	suite.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Indicator)))
	suite.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Indicator)))

	db, _, err := badgerhelper.NewTemp(testutils.DBFileName(suite))
	suite.NoError(err)
	suite.storage = badgerStore.New(db)

	tmpIndex, err := globalindex.TempInitializeIndices("")
	suite.NoError(err)
	suite.indexer = index.New(tmpIndex)

	suite.searcher = processSearch.New(suite.storage, suite.indexer)

	suite.mockCtrl = gomock.NewController(suite.T())
}

func (suite *IndicatorDataStoreTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *IndicatorDataStoreTestSuite) setupDataStoreNoPruning() {
	var err error
	suite.datastore, err = New(suite.storage, suite.indexer, suite.searcher, nil)
	suite.Require().NoError(err)
}

func (suite *IndicatorDataStoreTestSuite) setupDataStoreWithMocks() (*storeMocks.MockStore, *indexMocks.MockIndexer, *searchMocks.MockSearcher) {
	mockStorage := storeMocks.NewMockStore(suite.mockCtrl)
	mockStorage.EXPECT().GetKeysToIndex().Return(nil, nil)

	mockIndexer := indexMocks.NewMockIndexer(suite.mockCtrl)
	mockIndexer.EXPECT().NeedsInitialIndexing().Return(false, nil)

	mockSearcher := searchMocks.NewMockSearcher(suite.mockCtrl)
	var err error
	suite.datastore, err = New(mockStorage, mockIndexer, mockSearcher, nil)
	suite.Require().NoError(err)

	return mockStorage, mockIndexer, mockSearcher
}

func (suite *IndicatorDataStoreTestSuite) verifyIndicatorsAre(indicators ...*storage.ProcessIndicator) {
	indexResults, err := suite.indexer.Search(search.EmptyQuery())
	suite.NoError(err)
	suite.Len(indexResults, len(indicators))
	resultIDs := make([]string, 0, len(indexResults))
	for _, r := range indexResults {
		resultIDs = append(resultIDs, r.ID)
	}
	indicatorIDs := make([]string, 0, len(indicators))
	for _, i := range indicators {
		indicatorIDs = append(indicatorIDs, i.GetId())
	}
	suite.ElementsMatch(resultIDs, indicatorIDs)

	boltResults, err := suite.storage.GetProcessIndicators()
	suite.NoError(err)
	suite.Len(boltResults, len(indicators))
	for _, ind := range indicators {
		ind.DeploymentStateTs = 0
	}
	for _, ind := range boltResults {
		ind.DeploymentStateTs = 0
	}
	suite.ElementsMatch(boltResults, indicators)
}

func getIndicators() (indicators []*storage.ProcessIndicator, repeatIndicator *storage.ProcessIndicator) {
	repeatedSignal := &storage.ProcessSignal{
		Args:         "da_args",
		ContainerId:  "aa",
		Name:         "blah",
		ExecFilePath: "file",
	}

	indicators = []*storage.ProcessIndicator{
		{
			Id:           "id1",
			DeploymentId: "d1",

			Signal: repeatedSignal,
		},
		{
			Id:           "id2",
			DeploymentId: "d2",

			Signal: &storage.ProcessSignal{
				Name: "blah",
				Args: "args2",
			},
		},
	}

	repeatIndicator = &storage.ProcessIndicator{
		Id:           "id1",
		DeploymentId: "d1",
		Signal:       repeatedSignal,
	}
	return
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorBatchAdd() {
	suite.setupDataStoreNoPruning()

	indicators, repeatIndicator := getIndicators()
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, append(indicators, repeatIndicator)...))
	suite.verifyIndicatorsAre(indicators[1], repeatIndicator)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorBatchAddWithOldIndicator() {
	suite.setupDataStoreNoPruning()

	indicators, repeatIndicator := getIndicators()
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators[0]))
	suite.verifyIndicatorsAre(indicators[0])

	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators[1], repeatIndicator))
	suite.verifyIndicatorsAre(indicators[1], repeatIndicator)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorAddOneByOne() {
	suite.setupDataStoreNoPruning()

	indicators, repeatIndicator := getIndicators()
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators[0]))
	suite.verifyIndicatorsAre(indicators[0])

	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators[1]))
	suite.verifyIndicatorsAre(indicators...)

	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, repeatIndicator))
	suite.verifyIndicatorsAre(indicators[1], repeatIndicator)
}

func generateIndicatorsWithTS(deploymentTSs map[string]int64, containerIDs []string) []*storage.ProcessIndicator {
	var indicators []*storage.ProcessIndicator
	for d, ts := range deploymentTSs {
		for _, c := range containerIDs {
			indicators = append(indicators, &storage.ProcessIndicator{
				Id:           fmt.Sprintf("indicator_id_%s_%s", d, c),
				DeploymentId: d,
				Signal: &storage.ProcessSignal{
					ContainerId:  fmt.Sprintf("%s_%s", d, c),
					ExecFilePath: fmt.Sprintf("EXECFILE_%s_%s", d, c),
				},
				DeploymentStateTs: ts,
			})
		}
	}
	return indicators
}

func generateIndicators(deploymentIDs []string, containerIDs []string) []*storage.ProcessIndicator {
	var indicators []*storage.ProcessIndicator
	for _, d := range deploymentIDs {
		for _, c := range containerIDs {
			indicators = append(indicators, &storage.ProcessIndicator{
				Id:           fmt.Sprintf("indicator_id_%s_%s", d, c),
				DeploymentId: d,
				Signal: &storage.ProcessSignal{
					ContainerId:  fmt.Sprintf("%s_%s", d, c),
					ExecFilePath: fmt.Sprintf("EXECFILE_%s_%s", d, c),
				},
			})
		}
	}
	return indicators
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorRemovalByDeploymentID() {
	suite.setupDataStoreNoPruning()

	indicators := generateIndicators([]string{"d1", "d2"}, []string{"c1", "c2"})
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators...))
	suite.verifyIndicatorsAre(indicators...)

	suite.NoError(suite.datastore.RemoveProcessIndicatorsByDeployment(suite.hasWriteCtx, "d1"))
	suite.verifyIndicatorsAre(generateIndicators([]string{"d2"}, []string{"c1", "c2"})...)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorRemovalByDeploymentIDAgain() {
	suite.setupDataStoreNoPruning()

	indicators := generateIndicators([]string{"d1", "d2", "d3"}, []string{"c1", "c2", "c3"})
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators...))
	suite.verifyIndicatorsAre(indicators...)

	suite.NoError(suite.datastore.RemoveProcessIndicatorsByDeployment(suite.hasWriteCtx, "dnonexistent"))
	suite.verifyIndicatorsAre(indicators...)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorRemovalByContainerID() {
	suite.setupDataStoreNoPruning()

	indicators := generateIndicatorsWithTS(map[string]int64{"d1": 1234, "d2": 1234}, []string{"c1", "c2"})
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators...))
	suite.verifyIndicatorsAre(indicators...)

	deploy1 := &storage.Deployment{
		Id: "d1",
		Containers: []*storage.Container{
			{
				Instances: []*storage.ContainerInstance{
					{
						InstanceId: &storage.ContainerInstanceID{
							Id: "d1_c2",
						},
					},
				},
			},
		},
		StateTimestamp: 1235,
	}
	suite.NoError(suite.datastore.RemoveProcessIndicatorsOfStaleContainers(suite.hasWriteCtx, deploy1))
	suite.verifyIndicatorsAre(
		append(generateIndicators([]string{"d1"}, []string{"c2"}), generateIndicators([]string{"d2"}, []string{"c1", "c2"})...)...)

	deploy2 := &storage.Deployment{
		Id:             "d2",
		StateTimestamp: 1233,
	}
	suite.NoError(suite.datastore.RemoveProcessIndicatorsOfStaleContainers(suite.hasWriteCtx, deploy2))
	suite.verifyIndicatorsAre(
		append(generateIndicators([]string{"d1"}, []string{"c2"}), generateIndicators([]string{"d2"}, []string{"c1", "c2"})...)...)

	deploy2 = &storage.Deployment{
		Id: "d2",
		Containers: []*storage.Container{
			{
				Instances: []*storage.ContainerInstance{
					{
						InstanceId: &storage.ContainerInstanceID{
							Id: "d2_c2",
						},
					},
				},
			},
		},
		StateTimestamp: 1233,
	}
	suite.NoError(suite.datastore.RemoveProcessIndicatorsOfStaleContainers(suite.hasWriteCtx, deploy2))
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorRemovalByContainerIDAgain() {
	suite.setupDataStoreNoPruning()

	indicators := generateIndicators([]string{"d1", "d2"}, []string{"c1", "c2"})
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators...))
	suite.verifyIndicatorsAre(indicators...)

	deploy1 := &storage.Deployment{
		Id: "d1",
	}
	suite.NoError(suite.datastore.RemoveProcessIndicatorsOfStaleContainers(suite.hasWriteCtx, deploy1))
	suite.verifyIndicatorsAre(generateIndicators([]string{"d2"}, []string{"c1", "c2"})...)
}

func (suite *IndicatorDataStoreTestSuite) TestPruning() {
	const prunePeriod = 100 * time.Millisecond
	mockPrunerFactory := prunerMocks.NewMockFactory(suite.mockCtrl)
	mockPrunerFactory.EXPECT().Period().Return(prunePeriod)
	indicators, _ := getIndicators()

	prunedSignal := concurrency.NewSignal()
	mockPruner := prunerMocks.NewMockPruner(suite.mockCtrl)
	mockPruner.EXPECT().Finish().AnyTimes().Do(func() {
		prunedSignal.Signal()
	})

	pruneTurnstile := concurrency.NewTurnstile()
	mockPrunerFactory.EXPECT().StartPruning().AnyTimes().DoAndReturn(func() pruner.Pruner {
		pruneTurnstile.Wait()
		return mockPruner
	})

	matcher := func(expectedIndicators ...*storage.ProcessIndicator) gomock.Matcher {
		return testutils.PredMatcher(fmt.Sprintf("id with args matcher for %+v", expectedIndicators), func(passed []processindicator.IDAndArgs) bool {
			ourIDsAndArgs := make([]processindicator.IDAndArgs, 0, len(expectedIndicators))
			for _, indicator := range expectedIndicators {
				ourIDsAndArgs = append(ourIDsAndArgs, processindicator.IDAndArgs{ID: indicator.GetId(), Args: indicator.GetSignal().GetArgs()})
			}
			sort.Slice(ourIDsAndArgs, func(i, j int) bool {
				return ourIDsAndArgs[i].ID < ourIDsAndArgs[j].ID
			})
			sort.Slice(passed, func(i, j int) bool {
				return passed[i].ID < passed[j].ID
			})
			if len(ourIDsAndArgs) != len(passed) {
				return false
			}
			for i, idAndArg := range ourIDsAndArgs {
				if idAndArg.ID != passed[i].ID || idAndArg.Args != passed[i].Args {
					return false
				}
			}
			return true
		})
	}
	var err error
	suite.datastore, err = New(suite.storage, suite.indexer, suite.searcher, mockPrunerFactory)
	suite.Require().NoError(err)
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators...))
	suite.verifyIndicatorsAre(indicators...)

	mockPruner.EXPECT().Prune(matcher(indicators...)).Return(nil)
	pruneTurnstile.AllowOne()
	suite.True(concurrency.WaitWithTimeout(&prunedSignal, 3*prunePeriod))
	suite.verifyIndicatorsAre(indicators...)

	// Allow the next prune to go through. However, the prune function should not be
	// called because we should have a cache hit.
	prunedSignal.Reset()
	pruneTurnstile.AllowOne()
	suite.True(concurrency.WaitWithTimeout(&prunedSignal, 3*prunePeriod))
	suite.verifyIndicatorsAre(indicators...)

	// Now add an extra indicator; this should cause a cache miss and we should hit the pruning.
	extraIndicator := proto.Clone(indicators[0]).(*storage.ProcessIndicator)
	extraIndicator.Id = uuid.NewV4().String()
	extraIndicator.Signal.Args = uuid.NewV4().String()
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, extraIndicator))

	// Allow the next prune to go through; this time, prune something.
	expectedIndicators := []*storage.ProcessIndicator{extraIndicator}
	expectedIndicators = append(expectedIndicators, indicators...)
	mockPruner.EXPECT().Prune(matcher(expectedIndicators...)).Return([]string{indicators[0].GetId()})
	prunedSignal.Reset()
	pruneTurnstile.AllowOne()
	suite.True(concurrency.WaitWithTimeout(&prunedSignal, 3*prunePeriod))
	expectedIndicators = []*storage.ProcessIndicator{extraIndicator}
	expectedIndicators = append(expectedIndicators, indicators[1:]...)
	suite.verifyIndicatorsAre(expectedIndicators...)

	// Allow the next prune to go through; we should have a cache hit, so no need to mock a call.
	prunedSignal.Reset()
	pruneTurnstile.AllowOne()
	suite.True(concurrency.WaitWithTimeout(&prunedSignal, 3*prunePeriod))

	suite.Len(suite.datastore.(*datastoreImpl).prunedArgsLengthCache, 1)

	// Delete all the indicators.
	uniqueDeploymentIDs := set.NewStringSet()
	for _, indicator := range expectedIndicators {
		uniqueDeploymentIDs.Add(indicator.GetDeploymentId())
	}
	for _, depID := range uniqueDeploymentIDs.AsSlice() {
		suite.NoError(suite.datastore.RemoveProcessIndicatorsByDeployment(suite.hasWriteCtx, depID))
	}

	// Allow one more prune through.
	// All the indicators have been deleted, so no call to Prune is expected.
	prunedSignal.Reset()
	pruneTurnstile.AllowOne()
	suite.True(concurrency.WaitWithTimeout(&prunedSignal, 3*prunePeriod))

	// This is not great because we're testing an implementation detail, but whatever.
	// The goal is to make sure that there's no memory leak here.
	suite.Len(suite.datastore.(*datastoreImpl).prunedArgsLengthCache, 0)

	// Close the prune turnstile to allow all the prunes to go through, else the Stop will be blocked on the turnstile.
	pruneTurnstile.Close()

	suite.datastore.Stop()
	suite.True(suite.datastore.Wait(concurrency.Timeout(3 * prunePeriod)))
}

func (suite *IndicatorDataStoreTestSuite) TestEnforcesGet() {
	mockStore, _, _ := suite.setupDataStoreWithMocks()
	mockStore.EXPECT().GetProcessIndicator(gomock.Any()).Return(&storage.ProcessIndicator{}, true, nil)

	indicator, exists, err := suite.datastore.GetProcessIndicator(suite.hasNoneCtx, "hkjddjhk")
	suite.NoError(err, "expected no error, should return nil without access")
	suite.False(exists)
	suite.Nil(indicator, "expected return value to be nil")
}

func (suite *IndicatorDataStoreTestSuite) TestAllowsGet() {
	mockStore, _, _ := suite.setupDataStoreWithMocks()
	testIndicator := &storage.ProcessIndicator{}

	mockStore.EXPECT().GetProcessIndicator(gomock.Any()).Return(testIndicator, true, nil)
	indicator, exists, err := suite.datastore.GetProcessIndicator(suite.hasReadCtx, "An Id")
	suite.NoError(err, "expected no error trying to read with permissions")
	suite.True(exists)
	suite.Equal(testIndicator, indicator)

	mockStore.EXPECT().GetProcessIndicator(gomock.Any()).Return(testIndicator, true, nil)
	indicator, exists, err = suite.datastore.GetProcessIndicator(suite.hasWriteCtx, "beef")
	suite.NoError(err, "expected no error trying to read with permissions")
	suite.True(exists)
	suite.Equal(testIndicator, indicator)
}

func (suite *IndicatorDataStoreTestSuite) TestEnforcesAdd() {
	storeMock, indexMock, _ := suite.setupDataStoreWithMocks()
	storeMock.EXPECT().AddProcessIndicators(gomock.Any()).Times(0)
	indexMock.EXPECT().AddProcessIndicators(gomock.Any()).Times(0)

	err := suite.datastore.AddProcessIndicators(suite.hasNoneCtx, &storage.ProcessIndicator{})
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.AddProcessIndicators(suite.hasReadCtx, &storage.ProcessIndicator{})
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestEnforcesAddMany() {
	storeMock, indexMock, _ := suite.setupDataStoreWithMocks()
	storeMock.EXPECT().AddProcessIndicators(gomock.Any()).Times(0)
	indexMock.EXPECT().AddProcessIndicators(gomock.Any()).Times(0)

	err := suite.datastore.AddProcessIndicators(suite.hasNoneCtx, &storage.ProcessIndicator{})
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.AddProcessIndicators(suite.hasReadCtx, &storage.ProcessIndicator{})
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestAllowsAddMany() {
	storeMock, indexMock, _ := suite.setupDataStoreWithMocks()
	storeMock.EXPECT().AddProcessIndicators(gomock.Any()).Return(nil)
	indexMock.EXPECT().AddProcessIndicators(gomock.Any()).Return(nil)

	storeMock.EXPECT().AckKeysIndexed("id").Return(nil)

	err := suite.datastore.AddProcessIndicators(suite.hasWriteCtx, &storage.ProcessIndicator{Id: "id"})
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestEnforcesRemoveByDeployment() {
	_, indexMock, _ := suite.setupDataStoreWithMocks()
	indexMock.EXPECT().DeleteProcessIndicators(gomock.Any()).Times(0)

	err := suite.datastore.RemoveProcessIndicatorsByDeployment(suite.hasNoneCtx, "Joseph Rules")
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.RemoveProcessIndicatorsByDeployment(suite.hasReadCtx, "nfsiux")
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestAllowsRemoveByDeployment() {
	storeMock, indexMock, searchMock := suite.setupDataStoreWithMocks()
	searchMock.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{{ID: "jkldfjk"}}, nil)
	storeMock.EXPECT().RemoveProcessIndicators(gomock.Any()).Return(nil)
	indexMock.EXPECT().DeleteProcessIndicators(gomock.Any()).Return(nil)

	storeMock.EXPECT().AckKeysIndexed("jkldfjk").Return(nil)

	err := suite.datastore.RemoveProcessIndicatorsByDeployment(suite.hasWriteCtx, "eoiurvbf")
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestEnforcesRemoveByStaleContainers() {
	storeMock, indexMock, _ := suite.setupDataStoreWithMocks()
	storeMock.EXPECT().RemoveProcessIndicators(gomock.Any()).Times(0)
	indexMock.EXPECT().DeleteProcessIndicators(gomock.Any()).Times(0)

	deploy1 := &storage.Deployment{
		Id:   uuid.NewV4().String(),
		Name: "Joseph rules",
	}

	err := suite.datastore.RemoveProcessIndicatorsOfStaleContainers(suite.hasNoneCtx, deploy1)
	suite.Error(err, "expected an error trying to write without permissions")

	deploy2 := &storage.Deployment{
		Id:   uuid.NewV4().String(),
		Name: "nsfiux",
	}
	err = suite.datastore.RemoveProcessIndicatorsOfStaleContainers(suite.hasReadCtx, deploy2)
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestAllowsRemoveByStaleContainers() {
	storeMock, indexMock, searchMock := suite.setupDataStoreWithMocks()
	searchMock.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{{ID: "jkldfjk"}}, nil)
	storeMock.EXPECT().RemoveProcessIndicators(gomock.Any()).Return(nil)
	indexMock.EXPECT().DeleteProcessIndicators(gomock.Any()).Return(nil)

	storeMock.EXPECT().AckKeysIndexed("jkldfjk").Return(nil)

	deploy1 := &storage.Deployment{
		Id:   uuid.NewV4().String(),
		Name: "eoiurvbf",
	}
	err := suite.datastore.RemoveProcessIndicatorsOfStaleContainers(suite.hasWriteCtx, deploy1)
	suite.NoError(err, "expected no error trying to write with permissions")
}

func TestProcessIndicatorReindexSuite(t *testing.T) {
	suite.Run(t, new(ProcessIndicatorReindexSuite))
}

type ProcessIndicatorReindexSuite struct {
	suite.Suite

	storage  *storeMocks.MockStore
	indexer  *indexMocks.MockIndexer
	searcher *searchMocks.MockSearcher

	mockCtrl *gomock.Controller
}

func (suite *ProcessIndicatorReindexSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.storage = storeMocks.NewMockStore(suite.mockCtrl)
	suite.indexer = indexMocks.NewMockIndexer(suite.mockCtrl)
	suite.searcher = searchMocks.NewMockSearcher(suite.mockCtrl)
}

func (suite *ProcessIndicatorReindexSuite) TestReconciliationPartialReindex() {
	suite.storage.EXPECT().GetKeysToIndex().Return([]string{"A", "B", "C"}, nil)
	suite.indexer.EXPECT().NeedsInitialIndexing().Return(false, nil)

	pi1 := fixtures.GetProcessIndicator()
	pi1.Id = "A"
	pi2 := fixtures.GetProcessIndicator()
	pi2.Id = "B"
	pi3 := fixtures.GetProcessIndicator()
	pi3.Id = "C"

	processes := []*storage.ProcessIndicator{pi1, pi2, pi3}

	suite.storage.EXPECT().GetBatchProcessIndicators([]string{"A", "B", "C"}).Return(processes, nil, nil)
	suite.indexer.EXPECT().AddProcessIndicators(processes).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed([]string{"A", "B", "C"}).Return(nil)

	_, err := New(suite.storage, suite.indexer, suite.searcher, nil)
	suite.NoError(err)

	// Make listAlerts just A,B so C should be deleted
	processes = processes[:1]
	suite.storage.EXPECT().GetKeysToIndex().Return([]string{"A", "B", "C"}, nil)
	suite.indexer.EXPECT().NeedsInitialIndexing().Return(false, nil)

	suite.storage.EXPECT().GetBatchProcessIndicators([]string{"A", "B", "C"}).Return(processes, []int{2}, nil)
	suite.indexer.EXPECT().AddProcessIndicators(processes).Return(nil)
	suite.indexer.EXPECT().DeleteProcessIndicators([]string{"C"}).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed([]string{"A", "B", "C"}).Return(nil)

	_, err = New(suite.storage, suite.indexer, suite.searcher, nil)
	suite.NoError(err)
}
