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
	"github.com/stackrox/rox/central/processindicator/pruner"
	"github.com/stackrox/rox/central/processindicator/pruner/mocks"
	processSearch "github.com/stackrox/rox/central/processindicator/search"
	"github.com/stackrox/rox/central/processindicator/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/concurrency"
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

	mockCtrl *gomock.Controller
}

func (suite *IndicatorDataStoreTestSuite) SetupTest() {
	db, err := bolthelper.NewTemp(testutils.DBFileName(suite.Suite))
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
	suite.NoError(suite.datastore.AddProcessIndicators(context.TODO(), append(indicators, repeatIndicator)...))
	suite.verifyIndicatorsAre(indicators[1], repeatIndicator)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorBatchAddWithOldIndicator() {
	suite.setupDataStoreNoPruning()

	indicators, repeatIndicator := getIndicators()
	suite.NoError(suite.datastore.AddProcessIndicator(context.TODO(), indicators[0]))
	suite.verifyIndicatorsAre(indicators[0])

	suite.NoError(suite.datastore.AddProcessIndicators(context.TODO(), indicators[1], repeatIndicator))
	suite.verifyIndicatorsAre(indicators[1], repeatIndicator)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorAddOneByOne() {
	suite.setupDataStoreNoPruning()

	indicators, repeatIndicator := getIndicators()
	suite.NoError(suite.datastore.AddProcessIndicators(context.TODO(), indicators[0]))
	suite.verifyIndicatorsAre(indicators[0])

	suite.NoError(suite.datastore.AddProcessIndicators(context.TODO(), indicators[1]))
	suite.verifyIndicatorsAre(indicators...)

	suite.NoError(suite.datastore.AddProcessIndicators(context.TODO(), repeatIndicator))
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
	suite.NoError(suite.datastore.AddProcessIndicators(context.TODO(), indicators...))
	suite.verifyIndicatorsAre(indicators...)

	suite.NoError(suite.datastore.RemoveProcessIndicatorsByDeployment(context.TODO(), "d1"))
	suite.verifyIndicatorsAre(generateIndicators([]string{"d2"}, []string{"c1", "c2"})...)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorRemovalByDeploymentIDAgain() {
	suite.setupDataStoreNoPruning()

	indicators := generateIndicators([]string{"d1", "d2", "d3"}, []string{"c1", "c2", "c3"})
	suite.NoError(suite.datastore.AddProcessIndicators(context.TODO(), indicators...))
	suite.verifyIndicatorsAre(indicators...)

	suite.NoError(suite.datastore.RemoveProcessIndicatorsByDeployment(context.TODO(), "dnonexistent"))
	suite.verifyIndicatorsAre(indicators...)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorRemovalByContainerID() {
	suite.setupDataStoreNoPruning()

	indicators := generateIndicators([]string{"d1", "d2"}, []string{"c1", "c2"})
	suite.NoError(suite.datastore.AddProcessIndicators(context.TODO(), indicators...))
	suite.verifyIndicatorsAre(indicators...)

	suite.NoError(suite.datastore.RemoveProcessIndicatorsOfStaleContainers(context.TODO(), "d1", []string{"d1_c2"}))
	suite.verifyIndicatorsAre(
		append(generateIndicators([]string{"d1"}, []string{"c2"}), generateIndicators([]string{"d2"}, []string{"c1", "c2"})...)...)

	suite.NoError(suite.datastore.RemoveProcessIndicatorsOfStaleContainers(context.TODO(), "d2", []string{"d2_c2"}))
	suite.verifyIndicatorsAre(generateIndicators([]string{"d1", "d2"}, []string{"c2"})...)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorRemovalByContainerIDAgain() {
	suite.setupDataStoreNoPruning()

	indicators := generateIndicators([]string{"d1", "d2"}, []string{"c1", "c2"})
	suite.NoError(suite.datastore.AddProcessIndicators(context.TODO(), indicators...))
	suite.verifyIndicatorsAre(indicators...)

	suite.NoError(suite.datastore.RemoveProcessIndicatorsOfStaleContainers(context.TODO(), "d1", nil))
	suite.verifyIndicatorsAre(generateIndicators([]string{"d2"}, []string{"c1", "c2"})...)
}

func (suite *IndicatorDataStoreTestSuite) TestPruning() {
	const prunePeriod = 100 * time.Millisecond
	mockPrunerFactory := mocks.NewMockFactory(suite.mockCtrl)
	mockPrunerFactory.EXPECT().Period().Return(prunePeriod)
	indicators, _ := getIndicators()

	prunedSignal := concurrency.NewSignal()
	mockPruner := mocks.NewMockPruner(suite.mockCtrl)
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
	suite.NoError(suite.datastore.AddProcessIndicators(context.TODO(), indicators...))
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
