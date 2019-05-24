package datastore

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

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
	storeMocks "github.com/stackrox/rox/central/processindicator/store/mocks"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
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

	db, err := bolthelper.NewTemp(testutils.DBFileName(suite))
	suite.NoError(err)
	suite.storage = store.New(db)

	tmpIndex, err := globalindex.TempInitializeIndices("")
	suite.NoError(err)
	suite.indexer = index.New(tmpIndex)

	suite.searcher, err = processSearch.New(suite.storage, suite.indexer)
	suite.NoError(err)

	suite.mockCtrl = gomock.NewController(suite.T())
}

func (suite *IndicatorDataStoreTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *IndicatorDataStoreTestSuite) setupDataStoreNoPruning() {
	suite.datastore = New(suite.storage, suite.indexer, suite.searcher, nil)
}

func (suite *IndicatorDataStoreTestSuite) setupDataStoreWithMocks() (*storeMocks.MockStore, *indexMocks.MockIndexer, *searchMocks.MockSearcher) {
	mockStorage := storeMocks.NewMockStore(suite.mockCtrl)
	mockIndexer := indexMocks.NewMockIndexer(suite.mockCtrl)
	mockSearcher := searchMocks.NewMockSearcher(suite.mockCtrl)
	suite.datastore = New(mockStorage, mockIndexer, mockSearcher, nil)
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
		Id:           "id3",
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
	suite.NoError(suite.datastore.AddProcessIndicator(suite.hasWriteCtx, indicators[0]))
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

	indicators := generateIndicators([]string{"d1", "d2"}, []string{"c1", "c2"})
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators...))
	suite.verifyIndicatorsAre(indicators...)

	suite.NoError(suite.datastore.RemoveProcessIndicatorsOfStaleContainers(suite.hasWriteCtx, "d1", []string{"d1_c2"}))
	suite.verifyIndicatorsAre(
		append(generateIndicators([]string{"d1"}, []string{"c2"}), generateIndicators([]string{"d2"}, []string{"c1", "c2"})...)...)

	suite.NoError(suite.datastore.RemoveProcessIndicatorsOfStaleContainers(suite.hasWriteCtx, "d2", []string{"d2_c2"}))
	suite.verifyIndicatorsAre(generateIndicators([]string{"d1", "d2"}, []string{"c2"})...)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorRemovalByContainerIDAgain() {
	suite.setupDataStoreNoPruning()

	indicators := generateIndicators([]string{"d1", "d2"}, []string{"c1", "c2"})
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators...))
	suite.verifyIndicatorsAre(indicators...)

	suite.NoError(suite.datastore.RemoveProcessIndicatorsOfStaleContainers(suite.hasWriteCtx, "d1", nil))
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

	m := testutils.PredMatcher("id with args matcher", func(passed []processindicator.IDAndArgs) bool {
		ourIDsAndArgs := make([]processindicator.IDAndArgs, 0, len(indicators))
		for _, indicator := range indicators {
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
	suite.datastore = New(suite.storage, suite.indexer, suite.searcher, mockPrunerFactory)
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators...))
	suite.verifyIndicatorsAre(indicators...)

	mockPruner.EXPECT().Prune(m).Return(nil)
	pruneTurnstile.AllowOne()
	suite.Assert().True(concurrency.WaitWithTimeout(&prunedSignal, 3*prunePeriod))
	suite.verifyIndicatorsAre(indicators...)

	mockPruner.EXPECT().Prune(m).Return([]string{indicators[0].GetId()})
	prunedSignal.Reset()
	pruneTurnstile.AllowOne()
	suite.Assert().True(concurrency.WaitWithTimeout(&prunedSignal, 3*prunePeriod))
	suite.verifyIndicatorsAre(indicators[1:]...)

	mockPruner.EXPECT().Prune(gomock.Any()).AnyTimes().Return(nil)
	pruneTurnstile.Close()

	suite.datastore.Stop()
	suite.True(suite.datastore.Wait(concurrency.Timeout(3 * prunePeriod)))
}

func (suite *IndicatorDataStoreTestSuite) TestEnforcesGet() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
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

func (suite *IndicatorDataStoreTestSuite) TestEnforcesGetAll() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
	storeMock, _, searchMock := suite.setupDataStoreWithMocks()
	storeMock.EXPECT().GetProcessIndicators().Times(0)
	searchMock.EXPECT().SearchRawProcessIndicators(gomock.Any(), gomock.Any()).Return(nil, nil)

	indicators, err := suite.datastore.GetProcessIndicators(suite.hasNoneCtx)
	suite.NoError(err, "expected no error, should return nil without access")
	suite.Nil(indicators, "expected return value to be nil")
}

func (suite *IndicatorDataStoreTestSuite) TestAllowsGetAll() {
	storeMock, _, _ := suite.setupDataStoreWithMocks()
	storeMock.EXPECT().GetProcessIndicators().Return(nil, nil)

	_, err := suite.datastore.GetProcessIndicators(suite.hasReadCtx)
	suite.NoError(err, "expected no error trying to read with permissions")

	storeMock.EXPECT().GetProcessIndicators().Return(nil, nil)

	_, err = suite.datastore.GetProcessIndicators(suite.hasWriteCtx)
	suite.NoError(err, "expected no error trying to read with permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestEnforcesAdd() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
	storeMock, indexMock, _ := suite.setupDataStoreWithMocks()
	storeMock.EXPECT().AddProcessIndicator(gomock.Any()).Times(0)
	indexMock.EXPECT().AddProcessIndicator(gomock.Any()).Times(0)

	err := suite.datastore.AddProcessIndicator(suite.hasNoneCtx, &storage.ProcessIndicator{})
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.AddProcessIndicator(suite.hasReadCtx, &storage.ProcessIndicator{})
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestAllowsAdd() {
	storeMock, indexMock, _ := suite.setupDataStoreWithMocks()
	storeMock.EXPECT().AddProcessIndicator(gomock.Any()).Return("", nil)
	indexMock.EXPECT().AddProcessIndicator(gomock.Any()).Return(nil)

	err := suite.datastore.AddProcessIndicator(suite.hasWriteCtx, &storage.ProcessIndicator{})
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestEnforcesAddMany() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
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
	storeMock.EXPECT().AddProcessIndicators(gomock.Any()).Return(nil, nil)
	indexMock.EXPECT().AddProcessIndicators(gomock.Any()).Return(nil)

	err := suite.datastore.AddProcessIndicators(suite.hasWriteCtx, &storage.ProcessIndicator{})
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestEnforcesRemoveByDeployment() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
	storeMock, indexMock, _ := suite.setupDataStoreWithMocks()
	storeMock.EXPECT().RemoveProcessIndicator(gomock.Any()).Times(0)
	indexMock.EXPECT().DeleteProcessIndicators(gomock.Any()).Times(0)

	err := suite.datastore.RemoveProcessIndicatorsByDeployment(suite.hasNoneCtx, "Joseph Rules")
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.RemoveProcessIndicatorsByDeployment(suite.hasReadCtx, "nfsiux")
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestAllowsRemoveByDeployment() {
	storeMock, indexMock, searchMock := suite.setupDataStoreWithMocks()
	searchMock.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{{ID: "jkldfjk"}}, nil)
	storeMock.EXPECT().RemoveProcessIndicator(gomock.Any()).Return(nil)
	indexMock.EXPECT().DeleteProcessIndicators(gomock.Any()).Return(nil)

	err := suite.datastore.RemoveProcessIndicatorsByDeployment(suite.hasWriteCtx, "eoiurvbf")
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestEnforcesRemoveByStaleContainers() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
	storeMock, indexMock, _ := suite.setupDataStoreWithMocks()
	storeMock.EXPECT().RemoveProcessIndicator(gomock.Any()).Times(0)
	indexMock.EXPECT().DeleteProcessIndicators(gomock.Any()).Times(0)

	err := suite.datastore.RemoveProcessIndicatorsOfStaleContainers(suite.hasNoneCtx, "Joseph Rules", []string{})
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.RemoveProcessIndicatorsOfStaleContainers(suite.hasReadCtx, "nfsiux", []string{})
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestAllowsRemoveByStaleContainers() {
	storeMock, indexMock, searchMock := suite.setupDataStoreWithMocks()
	searchMock.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{{ID: "jkldfjk"}}, nil)
	storeMock.EXPECT().RemoveProcessIndicator(gomock.Any()).Return(nil)
	indexMock.EXPECT().DeleteProcessIndicators(gomock.Any()).Return(nil)

	err := suite.datastore.RemoveProcessIndicatorsOfStaleContainers(suite.hasWriteCtx, "eoiurvbf", []string{})
	suite.NoError(err, "expected no error trying to write with permissions")
}
