package boltdb

import (
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/suite"
)

func TestBoltBenchmarkTriggers(t *testing.T) {
	suite.Run(t, new(BoltBenchmarkTriggersTestSuite))
}

type BoltBenchmarkTriggersTestSuite struct {
	suite.Suite
	*BoltDB
}

func (suite *BoltBenchmarkTriggersTestSuite) SetupSuite() {
	db, err := boltFromTmpDir()
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.BoltDB = db
}

func (suite *BoltBenchmarkTriggersTestSuite) TeardownSuite() {
	suite.Close()
	os.Remove(suite.Path())
}

func (suite *BoltBenchmarkTriggersTestSuite) TestTriggers() {
	triggers := []*v1.BenchmarkTrigger{
		{
			Id:   "trigger1",
			Time: ptypes.TimestampNow(),
		},
		{
			Id:   "trigger2",
			Time: ptypes.TimestampNow(),
		},
	}

	// Test Add
	for _, trigger := range triggers {
		suite.NoError(suite.AddBenchmarkTrigger(trigger))
	}

	actualTriggers, err := suite.GetBenchmarkTriggers(&v1.GetBenchmarkTriggersRequest{})
	suite.NoError(err)

	suite.Equal(triggers, actualTriggers)
}
