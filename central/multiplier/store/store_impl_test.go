package store

import (
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestMultiplierStore(t *testing.T) {
	suite.Run(t, new(MultiplierStoreTestSuite))
}

type MultiplierStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *MultiplierStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *MultiplierStoreTestSuite) TearDownSuite() {
	testutils.TearDownDB(suite.db)
}

func (suite *MultiplierStoreTestSuite) TestMultipliers() {
	multipliers := []*storage.Multiplier{
		{
			Name:  "multiplier1",
			Value: 1,
		},
		{
			Name:  "multiplier2",
			Value: 2,
		},
	}

	// Test Add
	for _, m := range multipliers {
		id, err := suite.store.AddMultiplier(m)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	for _, m := range multipliers {
		got, exists, err := suite.store.GetMultiplier(m.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, m)
	}

	// Test Update
	for _, m := range multipliers {
		m.Value += 3
	}

	for _, r := range multipliers {
		suite.NoError(suite.store.UpdateMultiplier(r))
	}

	for _, m := range multipliers {
		got, exists, err := suite.store.GetMultiplier(m.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, m)
	}

	// Test Remove
	for _, m := range multipliers {
		suite.NoError(suite.store.RemoveMultiplier(m.GetId()))
	}

	for _, m := range multipliers {
		_, exists, err := suite.store.GetMultiplier(m.GetId())
		suite.NoError(err)
		suite.False(exists)
	}
}
