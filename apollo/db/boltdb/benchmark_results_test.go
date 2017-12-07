package boltdb

import (
	"io/ioutil"
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/suite"
)

func TestBoltBenchmarkResults(t *testing.T) {
	suite.Run(t, new(BoltBenchmarkResultTestSuite))
}

type BoltBenchmarkResultTestSuite struct {
	suite.Suite
	*BoltDB
}

func (suite *BoltBenchmarkResultTestSuite) SetupSuite() {
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

func (suite *BoltBenchmarkResultTestSuite) TeardownSuite() {
	suite.Close()
	os.Remove(suite.Path())
}

func (suite *BoltBenchmarkResultTestSuite) TestBenchmarks() {
	benchmarks := []*v1.BenchmarkResult{
		{
			Id:        "bench1",
			StartTime: ptypes.TimestampNow(),
			EndTime:   ptypes.TimestampNow(),
			Host:      "host1",
		},
		{
			Id:        "bench2",
			StartTime: ptypes.TimestampNow(),
			EndTime:   ptypes.TimestampNow(),
			Host:      "host2",
		},
	}

	// Test Add
	for _, b := range benchmarks {
		suite.NoError(suite.AddBenchmarkResult(b))
	}

	for _, b := range benchmarks {
		got, exists, err := suite.GetBenchmarkResult(b.Id)
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, b)
	}
}
