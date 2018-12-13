package store

import (
	"fmt"
	"os"
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stretchr/testify/suite"
)

func TestIndicatorStore(t *testing.T) {
	suite.Run(t, new(IndicatorStoreTestSuite))
}

type IndicatorStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *IndicatorStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *IndicatorStoreTestSuite) TearDownSuite() {
	suite.db.Close()
	os.Remove(suite.db.Path())
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
		Id:           "id3",
		DeploymentId: "d1",
		Signal:       repeatedSignal,
	}

	for _, i := range indicators {
		_, err := suite.store.AddProcessIndicator(i)
		suite.NoError(err)
	}

	suite.verifyIndicatorsAre(indicators...)

	// Adding an indicator with the same secondary key should replace the original one.
	removed, err := suite.store.AddProcessIndicator(repeatIndicator)
	suite.NoError(err)
	suite.Equal("id1", removed)
	suite.verifyIndicatorsAre(indicators[1], repeatIndicator)

	for _, i := range []*storage.ProcessIndicator{indicators[1], repeatIndicator} {
		suite.NoError(suite.store.RemoveProcessIndicator(i.GetId()))
	}
	suite.verifyIndicatorsAre()

	oldIDs, err := suite.store.AddProcessIndicators(indicators...)
	suite.NoError(err)
	suite.Empty(oldIDs)

	// Modify indicator ids so we can batch add and we should get the old values out
	for idx, i := range indicators {
		i.Id = fmt.Sprintf("id%d", idx+3)
	}
	oldIDs, err = suite.store.AddProcessIndicators(indicators...)
	suite.NoError(err)
	suite.ElementsMatch([]string{"id1", "id2"}, oldIDs)
}
