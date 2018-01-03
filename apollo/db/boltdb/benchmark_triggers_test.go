package boltdb

import (
	"io/ioutil"
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/ptypes"
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
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		suite.FailNow("Failed to get temporary directory", err.Error())
	}
	db, err := New(tmpDir)
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
			Name: "trigger1",
			Time: ptypes.TimestampNow(),
		},
		{
			Name: "trigger2",
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
