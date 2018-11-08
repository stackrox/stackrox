package datastore

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/processindicator/index"
	processSearch "github.com/stackrox/rox/central/processindicator/search"
	"github.com/stackrox/rox/central/processindicator/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/search"
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
}

func (suite *IndicatorDataStoreTestSuite) SetupTest() {
	db, err := bolthelper.NewTemp(fmt.Sprintf("Bolt%d.db", rand.Int()))
	suite.NoError(err)
	suite.storage = store.New(db)

	tmpIndex, err := globalindex.TempInitializeIndices("")
	suite.NoError(err)
	suite.indexer = index.New(tmpIndex)

	searcher, err := processSearch.New(suite.storage, suite.indexer)
	suite.NoError(err)

	suite.datastore = New(suite.storage, suite.indexer, searcher)
}

func (suite *IndicatorDataStoreTestSuite) verifyIndicatorsAre(indicators ...*v1.ProcessIndicator) {
	indexResults, err := suite.indexer.SearchProcessIndicators(search.EmptyQuery())
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

func getIndicators() (indicators []*v1.ProcessIndicator, repeatIndicator *v1.ProcessIndicator) {
	repeatedSignal := &v1.ProcessSignal{
		Args:         "da_args",
		ContainerId:  "aa",
		Name:         "blah",
		ExecFilePath: "file",
	}

	indicators = []*v1.ProcessIndicator{
		{
			Id:           "id1",
			DeploymentId: "d1",

			Signal: repeatedSignal,
		},
		{
			Id:           "id2",
			DeploymentId: "d2",

			Signal: &v1.ProcessSignal{
				Args: "args2",
			},
		},
	}

	repeatIndicator = &v1.ProcessIndicator{
		Id:           "id3",
		DeploymentId: "d1",
		Signal:       repeatedSignal,
	}
	return
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorBatchAdd() {
	indicators, repeatIndicator := getIndicators()
	suite.NoError(suite.datastore.AddProcessIndicators(append(indicators, repeatIndicator)...))
	suite.verifyIndicatorsAre(indicators[1], repeatIndicator)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorBatchAddWithOldIndicator() {
	indicators, repeatIndicator := getIndicators()
	suite.NoError(suite.datastore.AddProcessIndicator(indicators[0]))
	suite.verifyIndicatorsAre(indicators[0])

	suite.NoError(suite.datastore.AddProcessIndicators(indicators[1], repeatIndicator))
	suite.verifyIndicatorsAre(indicators[1], repeatIndicator)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorAddOneByOne() {
	indicators, repeatIndicator := getIndicators()
	suite.NoError(suite.datastore.AddProcessIndicators(indicators[0]))
	suite.verifyIndicatorsAre(indicators[0])

	suite.NoError(suite.datastore.AddProcessIndicators(indicators[1]))
	suite.verifyIndicatorsAre(indicators...)

	suite.NoError(suite.datastore.AddProcessIndicators(repeatIndicator))
	suite.verifyIndicatorsAre(indicators[1], repeatIndicator)
}

func generateIndicators(deploymentIDs []string, containerIDs []string) []*v1.ProcessIndicator {
	var indicators []*v1.ProcessIndicator
	for _, d := range deploymentIDs {
		for _, c := range containerIDs {
			indicators = append(indicators, &v1.ProcessIndicator{
				Id:           fmt.Sprintf("indicator_id_%s_%s", d, c),
				DeploymentId: d,
				Signal: &v1.ProcessSignal{
					ContainerId:  fmt.Sprintf("%s_%s", d, c),
					ExecFilePath: fmt.Sprintf("EXECFILE_%s_%s", d, c),
				},
			})
		}
	}
	return indicators
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorRemovalByDeploymentID() {
	indicators := generateIndicators([]string{"d1", "d2"}, []string{"c1", "c2"})
	suite.NoError(suite.datastore.AddProcessIndicators(indicators...))
	suite.verifyIndicatorsAre(indicators...)

	suite.NoError(suite.datastore.RemoveProcessIndicatorsByDeployment("d1"))
	suite.verifyIndicatorsAre(generateIndicators([]string{"d2"}, []string{"c1", "c2"})...)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorRemovalByDeploymentIDAgain() {
	indicators := generateIndicators([]string{"d1", "d2", "d3"}, []string{"c1", "c2", "c3"})
	suite.NoError(suite.datastore.AddProcessIndicators(indicators...))
	suite.verifyIndicatorsAre(indicators...)

	suite.NoError(suite.datastore.RemoveProcessIndicatorsByDeployment("dnonexistent"))
	suite.verifyIndicatorsAre(indicators...)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorRemovalByContainerID() {
	indicators := generateIndicators([]string{"d1", "d2"}, []string{"c1", "c2"})
	suite.NoError(suite.datastore.AddProcessIndicators(indicators...))
	suite.verifyIndicatorsAre(indicators...)

	suite.NoError(suite.datastore.RemoveProcessIndicatorsOfStaleContainers("d1", []string{"d1_c2"}))
	suite.verifyIndicatorsAre(
		append(generateIndicators([]string{"d1"}, []string{"c2"}), generateIndicators([]string{"d2"}, []string{"c1", "c2"})...)...)

	suite.NoError(suite.datastore.RemoveProcessIndicatorsOfStaleContainers("d2", []string{"d2_c2"}))
	suite.verifyIndicatorsAre(generateIndicators([]string{"d1", "d2"}, []string{"c2"})...)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorRemovalByContainerIDAgain() {
	indicators := generateIndicators([]string{"d1", "d2"}, []string{"c1", "c2"})
	suite.NoError(suite.datastore.AddProcessIndicators(indicators...))
	suite.verifyIndicatorsAre(indicators...)

	suite.NoError(suite.datastore.RemoveProcessIndicatorsOfStaleContainers("d1", nil))
	suite.verifyIndicatorsAre(generateIndicators([]string{"d2"}, []string{"c1", "c2"})...)
}
