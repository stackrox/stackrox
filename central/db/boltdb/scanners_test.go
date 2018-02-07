package boltdb

import (
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestBoltScanners(t *testing.T) {
	suite.Run(t, new(BoltScannerTestSuite))
}

type BoltScannerTestSuite struct {
	suite.Suite
	*BoltDB
}

func (suite *BoltScannerTestSuite) SetupSuite() {
	db, err := boltFromTmpDir()
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.BoltDB = db
}

func (suite *BoltScannerTestSuite) TeardownSuite() {
	suite.Close()
	os.Remove(suite.Path())
}

func (suite *BoltScannerTestSuite) TestScanners() {
	scanners := []*v1.Scanner{
		{
			Name:     "scanner1",
			Endpoint: "https://endpoint1",
		},
		{
			Name:     "scanner2",
			Endpoint: "https://endpoint2",
		},
	}

	// Test Add
	for _, r := range scanners {
		id, err := suite.AddScanner(r)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	for _, r := range scanners {
		got, exists, err := suite.GetScanner(r.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, r)
	}

	// Test Update
	for _, r := range scanners {
		r.Endpoint += "/api"
	}

	for _, r := range scanners {
		suite.NoError(suite.UpdateScanner(r))
	}

	for _, r := range scanners {
		got, exists, err := suite.GetScanner(r.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, r)
	}

	// Test Remove
	for _, r := range scanners {
		suite.NoError(suite.RemoveScanner(r.GetId()))
	}

	for _, r := range scanners {
		_, exists, err := suite.GetScanner(r.GetId())
		suite.NoError(err)
		suite.False(exists)
	}
}
