package boltdb

import (
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestBoltMultipliers(t *testing.T) {
	suite.Run(t, new(BoltMultiplierTestSuite))
}

type BoltMultiplierTestSuite struct {
	suite.Suite
	*BoltDB
}

func (suite *BoltMultiplierTestSuite) SetupSuite() {
	db, err := boltFromTmpDir()
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.BoltDB = db
}

func (suite *BoltMultiplierTestSuite) TeardownSuite() {
	suite.Close()
	os.Remove(suite.Path())
}

func (suite *BoltMultiplierTestSuite) TestMultipliers() {
	multipliers := []*v1.Multiplier{
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
		id, err := suite.AddMultiplier(m)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	for _, m := range multipliers {
		got, exists, err := suite.GetMultiplier(m.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, m)
	}

	// Test Update
	for _, m := range multipliers {
		m.Value += 3
	}

	for _, r := range multipliers {
		suite.NoError(suite.UpdateMultiplier(r))
	}

	for _, m := range multipliers {
		got, exists, err := suite.GetMultiplier(m.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, m)
	}

	// Test Remove
	for _, m := range multipliers {
		suite.NoError(suite.RemoveMultiplier(m.GetId()))
	}

	for _, m := range multipliers {
		_, exists, err := suite.GetMultiplier(m.GetId())
		suite.NoError(err)
		suite.False(exists)
	}
}
