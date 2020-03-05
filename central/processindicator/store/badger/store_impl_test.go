package badger

import (
	"fmt"
	"os"
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/central/processindicator/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stretchr/testify/suite"
)

func TestIndicatorStore(t *testing.T) {
	suite.Run(t, new(IndicatorStoreTestSuite))
}

type IndicatorStoreTestSuite struct {
	suite.Suite

	db  *badger.DB
	dir string

	store store.Store
}

func (suite *IndicatorStoreTestSuite) SetupSuite() {
	db, dir, err := badgerhelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BadgerDB", err.Error())
	}

	suite.db = db
	suite.dir = dir
	suite.store = New(db)
}

func (suite *IndicatorStoreTestSuite) TearDownSuite() {
	_ = suite.db.Close()
	_ = os.RemoveAll(suite.dir)
}

func (suite *IndicatorStoreTestSuite) verifyIndicatorsAre(indicators ...*storage.ProcessIndicator) {
	for _, i := range indicators {
		retrievedIndicator, exists, err := suite.store.GetProcessIndicator(i.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.NotNil(retrievedIndicator)
		suite.Equal(i, retrievedIndicator)
	}

	// Get all indicators
	retrievedIndicators, err := suite.store.GetProcessIndicators()
	suite.NoError(err)
	suite.ElementsMatch(indicators, retrievedIndicators)

}

func (suite *IndicatorStoreTestSuite) TestIndicators() {
	repeatedSignal := &storage.ProcessSignal{
		Args:         "da_args",
		ContainerId:  "aa",
		Name:         "blah",
		ExecFilePath: "file",
	}

	indicators := []*storage.ProcessIndicator{
		{
			Id:           "id1",
			DeploymentId: "d1",

			Signal: repeatedSignal,
		},
		{
			Id:           "id2",
			DeploymentId: "d2",

			Signal: &storage.ProcessSignal{
				Args: "args2",
			},
		},
	}

	repeatIndicator := &storage.ProcessIndicator{
		Id:           "id1",
		DeploymentId: "d1",
		Signal:       repeatedSignal,
	}

	err := suite.store.AddProcessIndicators(indicators...)
	suite.NoError(err)
	suite.verifyIndicatorsAre(indicators...)

	for _, i := range indicators {
		keys, err := suite.store.GetKeysToIndex()
		suite.NoError(err)
		suite.Contains(keys, i.GetId())
	}

	var walkedIndicators []*storage.ProcessIndicator
	err = suite.store.WalkAll(func(pi *storage.ProcessIndicator) error {
		walkedIndicators = append(walkedIndicators, pi)
		return nil
	})
	suite.NoError(err)
	suite.verifyIndicatorsAre(walkedIndicators...)

	// Clear indexing
	keys, err := suite.store.GetKeysToIndex()
	suite.NoError(err)
	suite.NoError(suite.store.AckKeysIndexed(keys...))

	// Adding an indicator with the same secondary key should replace the original one.
	err = suite.store.AddProcessIndicators(repeatIndicator)
	suite.NoError(err)
	suite.verifyIndicatorsAre(indicators[1], repeatIndicator)

	// The removed key and the added key should both be waiting to be indexed
	keys, err = suite.store.GetKeysToIndex()
	suite.NoError(err)
	suite.ElementsMatch([]string{repeatIndicator.GetId()}, keys)

	// Clear indexing
	keys, err = suite.store.GetKeysToIndex()
	suite.NoError(err)
	suite.NoError(suite.store.AckKeysIndexed(keys...))

	for _, i := range []*storage.ProcessIndicator{indicators[1], repeatIndicator} {
		suite.NoError(suite.store.RemoveProcessIndicators([]string{i.GetId()}))
		keys, err = suite.store.GetKeysToIndex()
		suite.NoError(err)
		suite.Contains(keys, i.GetId())
	}
	suite.verifyIndicatorsAre()

	keys, err = suite.store.GetKeysToIndex()
	suite.NoError(err)
	suite.NoError(suite.store.AckKeysIndexed(keys...))

	err = suite.store.AddProcessIndicators(indicators...)
	suite.NoError(err)

	indicatorIDs := make([]string, 0, len(indicators))
	for _, i := range indicators {
		indicatorIDs = append(indicatorIDs, i.GetId())
	}
	keys, err = suite.store.GetKeysToIndex()
	suite.NoError(err)
	suite.ElementsMatch(keys, indicatorIDs)

	// Modify indicator ids so we can batch add and we should get the old values out
	for idx, i := range indicators {
		i.Id = fmt.Sprintf("id%d", idx+3)
	}

	// Clear indexing
	keys, err = suite.store.GetKeysToIndex()
	suite.NoError(err)
	suite.NoError(suite.store.AckKeysIndexed(keys...))

	err = suite.store.AddProcessIndicators(indicators...)
	suite.NoError(err)

	indicators, err = suite.store.GetProcessIndicators()
	suite.NoError(err)

	var ids []string
	for _, i := range indicators {
		ids = append(ids, i.GetId())
	}
	suite.NoError(suite.store.RemoveProcessIndicators(ids))

	v, err := suite.store.GetProcessInfoToArgs()
	suite.NoError(err)
	suite.Len(v, 0)
}
