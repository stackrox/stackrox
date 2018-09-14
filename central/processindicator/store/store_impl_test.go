package store

import (
	"os"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
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

func (suite *IndicatorStoreTestSuite) TeardownSuite() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *IndicatorStoreTestSuite) TestIndicators() {
	var indicators = []*v1.ProcessIndicator{
		{
			Id:           "id1",
			DeploymentId: "d1",

			Signal: &v1.ProcessSignal{
				Args: "args",
			},
		},
		{
			Id:           "id2",
			DeploymentId: "d2",

			Signal: &v1.ProcessSignal{
				Args: "args2",
			},
		},
	}

	for _, i := range indicators {
		inserted, err := suite.store.AddProcessIndicator(i)
		suite.NoError(err)
		suite.True(inserted)
	}

	// Adding an indicator with the exact same commandline should have inserted be false
	for _, i := range indicators {
		inserted, _ := suite.store.AddProcessIndicator(i)
		suite.False(inserted)
	}

	// Get all indicators
	retrievedIndicators, err := suite.store.GetProcessIndicators()
	suite.Nil(err)
	suite.ElementsMatch(indicators, retrievedIndicators)

	for _, i := range indicators {
		retrievedIndicator, exists, err := suite.store.GetProcessIndicator(i.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(i, retrievedIndicator)
	}

	for _, i := range indicators {
		suite.NoError(suite.store.RemoveProcessIndicator(i.GetId()))
	}

	retrievedIndicators, err = suite.store.GetProcessIndicators()
	suite.NoError(err)
	suite.Empty(retrievedIndicators)
}
