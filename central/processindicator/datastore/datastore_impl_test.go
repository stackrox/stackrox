//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/stackrox/rox/central/processindicator"
	indexMocks "github.com/stackrox/rox/central/processindicator/index/mocks"
	"github.com/stackrox/rox/central/processindicator/pruner"
	prunerMocks "github.com/stackrox/rox/central/processindicator/pruner/mocks"
	processSearch "github.com/stackrox/rox/central/processindicator/search"
	searchMocks "github.com/stackrox/rox/central/processindicator/search/mocks"
	"github.com/stackrox/rox/central/processindicator/store"
	storeMocks "github.com/stackrox/rox/central/processindicator/store/mocks"
	postgresStore "github.com/stackrox/rox/central/processindicator/store/postgres"
	plopStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestIndicatorDatastore(t *testing.T) {
	suite.Run(t, new(IndicatorDataStoreTestSuite))
}

type IndicatorDataStoreTestSuite struct {
	suite.Suite
	datastore   DataStore
	storage     store.Store
	plopStorage plopStore.Store
	searcher    processSearch.Searcher

	postgres *pgtest.TestPostgres

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	mockCtrl *gomock.Controller

	podToIndicators map[string]map[string]string
}

func (suite *IndicatorDataStoreTestSuite) SetupTest() {
	suite.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	suite.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension)))
	suite.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension)))

	suite.postgres = pgtest.ForT(suite.T())
	suite.storage = postgresStore.New(suite.postgres.DB)
	suite.plopStorage = plopStore.New(suite.postgres.DB)
	suite.searcher = processSearch.New(suite.storage, postgresStore.NewIndexer(suite.postgres.DB))

	suite.mockCtrl = gomock.NewController(suite.T())

	suite.initPodToIndicatorsMap()
}

func (suite *IndicatorDataStoreTestSuite) TearDownTest() {
	suite.postgres.Teardown(suite.T())
	suite.mockCtrl.Finish()
}

func (suite *IndicatorDataStoreTestSuite) initPodToIndicatorsMap() {
	suite.podToIndicators = make(map[string]map[string]string)
	suite.podToIndicators[fixtureconsts.PodUID1] = make(map[string]string)
	suite.podToIndicators[fixtureconsts.PodUID1]["c1"] = uuid.NewV4().String()
	suite.podToIndicators[fixtureconsts.PodUID1]["c2"] = uuid.NewV4().String()
	suite.podToIndicators[fixtureconsts.PodUID1]["c3"] = uuid.NewV4().String()

	suite.podToIndicators[fixtureconsts.PodUID2] = make(map[string]string)
	suite.podToIndicators[fixtureconsts.PodUID2]["c1"] = uuid.NewV4().String()
	suite.podToIndicators[fixtureconsts.PodUID2]["c2"] = uuid.NewV4().String()
	suite.podToIndicators[fixtureconsts.PodUID2]["c3"] = uuid.NewV4().String()

	suite.podToIndicators[fixtureconsts.PodUID3] = make(map[string]string)
	suite.podToIndicators[fixtureconsts.PodUID3]["c1"] = uuid.NewV4().String()
	suite.podToIndicators[fixtureconsts.PodUID3]["c2"] = uuid.NewV4().String()
	suite.podToIndicators[fixtureconsts.PodUID3]["c3"] = uuid.NewV4().String()
}

func (suite *IndicatorDataStoreTestSuite) setupDataStoreNoPruning() {
	var err error
	suite.datastore, err = New(suite.storage, suite.plopStorage, suite.searcher, nil)
	suite.Require().NoError(err)
}

func (suite *IndicatorDataStoreTestSuite) setupDataStoreWithMocks() (*storeMocks.MockStore, *indexMocks.MockIndexer, *searchMocks.MockSearcher) {
	mockStorage := storeMocks.NewMockStore(suite.mockCtrl)
	mockIndexer := indexMocks.NewMockIndexer(suite.mockCtrl)
	mockSearcher := searchMocks.NewMockSearcher(suite.mockCtrl)
	var err error
	suite.datastore, err = New(mockStorage, nil, mockSearcher, nil)
	suite.Require().NoError(err)

	return mockStorage, mockIndexer, mockSearcher
}

func (suite *IndicatorDataStoreTestSuite) verifyIndicatorsAre(indicators ...*storage.ProcessIndicator) {
	indexResults, err := suite.searcher.Search(suite.hasWriteCtx, search.EmptyQuery())
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

	var foundIndicators []*storage.ProcessIndicator
	err = suite.storage.Walk(suite.hasReadCtx, func(pi *storage.ProcessIndicator) error {
		foundIndicators = append(foundIndicators, pi)
		return nil
	})
	suite.NoError(err)
	suite.ElementsMatch(foundIndicators, indicators)
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
			Id:            fixtureconsts.ProcessIndicatorID1,
			DeploymentId:  fixtureconsts.Deployment1,
			PodUid:        fixtureconsts.PodUID1,
			ContainerName: "container1",

			Signal: repeatedSignal,
		},
		{
			Id:            fixtureconsts.ProcessIndicatorID2,
			DeploymentId:  fixtureconsts.Deployment2,
			PodUid:        fixtureconsts.PodUID2,
			ContainerName: "container1",

			Signal: &storage.ProcessSignal{
				Name:         "blah",
				Args:         "args2",
				ExecFilePath: "blahpath",
			},
		},
	}

	repeatIndicator = &storage.ProcessIndicator{
		Id:            fixtureconsts.ProcessIndicatorID1,
		DeploymentId:  fixtureconsts.Deployment1,
		PodUid:        fixtureconsts.PodUID1,
		ContainerName: "container1",
		Signal:        repeatedSignal,
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

func (suite *IndicatorDataStoreTestSuite) generateIndicatorsWithPods(podIDs []string, containerIDs []string) []*storage.ProcessIndicator {
	var indicators []*storage.ProcessIndicator
	for _, p := range podIDs {
		for _, c := range containerIDs {
			indicators = append(indicators, &storage.ProcessIndicator{
				Id:     suite.podToIndicators[p][c],
				PodUid: p,
				Signal: &storage.ProcessSignal{
					ContainerId:  fmt.Sprintf("%s_%s", p, c),
					ExecFilePath: fmt.Sprintf("EXECFILE_%s_%s", p, c),
				},
			})
		}
	}
	return indicators
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorRemovalByPodID() {
	suite.setupDataStoreNoPruning()

	indicators := suite.generateIndicatorsWithPods([]string{fixtureconsts.PodUID1, fixtureconsts.PodUID2}, []string{"c1", "c2"})
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators...))
	suite.verifyIndicatorsAre(indicators...)

	suite.NoError(suite.datastore.RemoveProcessIndicatorsByPod(suite.hasWriteCtx, fixtureconsts.PodUID1))
	suite.verifyIndicatorsAre(suite.generateIndicatorsWithPods([]string{fixtureconsts.PodUID2}, []string{"c1", "c2"})...)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorRemovalByPodIDAgain() {
	suite.setupDataStoreNoPruning()

	indicators := suite.generateIndicatorsWithPods([]string{fixtureconsts.PodUID1, fixtureconsts.PodUID2, fixtureconsts.PodUID3}, []string{"c1", "c2", "c3"})
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators...))
	suite.verifyIndicatorsAre(indicators...)

	// Try to remove where pod id does not exist in indicators
	suite.NoError(suite.datastore.RemoveProcessIndicatorsByPod(suite.hasWriteCtx, uuid.NewV4().String()))
	suite.verifyIndicatorsAre(indicators...)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorRemovalBatch() {
	numIndicators := 70000
	suite.setupDataStoreNoPruning()

	indicators := suite.generateIndicatorsWithPods([]string{fixtureconsts.PodUID1, fixtureconsts.PodUID2, fixtureconsts.PodUID3}, []string{"c1", "c2", "c3"})
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators...))
	suite.verifyIndicatorsAre(indicators...)

	ids := make([]string, 0, numIndicators)
	for i, indicator := range indicators {
		// Skip the first one so we don't just delete them all
		if i == 0 {
			continue
		}
		ids = append(ids, indicator.Id)
	}

	for i := len(ids); i < numIndicators; i++ {
		ids = append(ids, uuid.NewV4().String())
	}

	// Try to remove where pod id does not exist in indicators
	suite.NoError(suite.datastore.RemoveProcessIndicators(suite.hasWriteCtx, ids))
	suite.verifyIndicatorsAre(indicators[0])
}

func (suite *IndicatorDataStoreTestSuite) TestPruning() {
	suite.T().Setenv(env.ProcessPruningEnabled.EnvVar(), "true")

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
	suite.datastore, err = New(suite.storage, suite.plopStorage, suite.searcher, mockPrunerFactory)
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
	extraIndicator := indicators[0].Clone()
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
	uniquePodIDs := set.NewStringSet()
	for _, indicator := range expectedIndicators {
		uniquePodIDs.Add(indicator.GetPodUid())
	}
	for podID := range uniquePodIDs {
		suite.NoError(suite.datastore.RemoveProcessIndicatorsByPod(suite.hasWriteCtx, podID))
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
	mockStore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.ProcessIndicator{}, true, nil)

	indicator, exists, err := suite.datastore.GetProcessIndicator(suite.hasNoneCtx, uuid.Nil.String())
	suite.NoError(err, "expected no error, should return nil without access")
	suite.False(exists)
	suite.Nil(indicator, "expected return value to be nil")
}

func (suite *IndicatorDataStoreTestSuite) TestAllowsGet() {
	mockStore, _, _ := suite.setupDataStoreWithMocks()
	testIndicator := &storage.ProcessIndicator{}

	mockStore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(testIndicator, true, nil)
	indicator, exists, err := suite.datastore.GetProcessIndicator(suite.hasReadCtx, uuid.Nil.String())
	suite.NoError(err, "expected no error trying to read with permissions")
	suite.True(exists)
	suite.Equal(testIndicator, indicator)

	mockStore.EXPECT().Get(suite.hasWriteCtx, gomock.Any()).Return(testIndicator, true, nil)
	indicator, exists, err = suite.datastore.GetProcessIndicator(suite.hasWriteCtx, uuid.NewDummy().String())
	suite.NoError(err, "expected no error trying to read with permissions")
	suite.True(exists)
	suite.Equal(testIndicator, indicator)
}

func (suite *IndicatorDataStoreTestSuite) TestEnforcesAdd() {
	storeMock, _, _ := suite.setupDataStoreWithMocks()
	storeMock.EXPECT().UpsertMany(suite.hasWriteCtx, gomock.Any()).Times(0)

	err := suite.datastore.AddProcessIndicators(suite.hasNoneCtx, &storage.ProcessIndicator{})
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.AddProcessIndicators(suite.hasReadCtx, &storage.ProcessIndicator{})
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestEnforcesAddMany() {
	storeMock, _, _ := suite.setupDataStoreWithMocks()
	storeMock.EXPECT().UpsertMany(suite.hasWriteCtx, gomock.Any()).Times(0)

	err := suite.datastore.AddProcessIndicators(suite.hasNoneCtx, &storage.ProcessIndicator{})
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.AddProcessIndicators(suite.hasReadCtx, &storage.ProcessIndicator{})
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestAllowsAddMany() {
	storeMock, _, _ := suite.setupDataStoreWithMocks()
	storeMock.EXPECT().UpsertMany(suite.hasWriteCtx, gomock.Any()).Return(nil)
	err := suite.datastore.AddProcessIndicators(suite.hasWriteCtx, &storage.ProcessIndicator{Id: fixtureconsts.ProcessIndicatorID1})
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestEnforcesRemoveByPod() {
	err := suite.datastore.RemoveProcessIndicatorsByPod(suite.hasNoneCtx, uuid.NewDummy().String())
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.RemoveProcessIndicatorsByPod(suite.hasReadCtx, uuid.Nil.String())
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestAllowsRemoveByPod() {
	storeMock, _, _ := suite.setupDataStoreWithMocks()
	storeMock.EXPECT().DeleteByQuery(gomock.Any(), gomock.Any()).Return(nil)

	err := suite.datastore.RemoveProcessIndicatorsByPod(suite.hasWriteCtx, uuid.NewDummy().String())
	suite.NoError(err, "expected no error trying to write with permissions")
}

func TestProcessIndicatorReindexSuite(t *testing.T) {
	suite.Run(t, new(ProcessIndicatorReindexSuite))
}

type ProcessIndicatorReindexSuite struct {
	suite.Suite

	storage  *storeMocks.MockStore
	searcher *searchMocks.MockSearcher

	mockCtrl *gomock.Controller
}

func (suite *ProcessIndicatorReindexSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.storage = storeMocks.NewMockStore(suite.mockCtrl)
	suite.searcher = searchMocks.NewMockSearcher(suite.mockCtrl)
}
