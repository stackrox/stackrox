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
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

var fakeID = fixtures.GetProcessIndicator().GetId()

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
