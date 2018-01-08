package boltdb

import (
	"io/ioutil"
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/suite"
)

func TestBoltBenchmarkSchedules(t *testing.T) {
	suite.Run(t, new(BoltBenchmarkSchedulesTestSuite))
}

type BoltBenchmarkSchedulesTestSuite struct {
	suite.Suite
	*BoltDB
}

func (suite *BoltBenchmarkSchedulesTestSuite) SetupSuite() {
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

func (suite *BoltBenchmarkSchedulesTestSuite) TeardownSuite() {
	suite.Close()
	os.Remove(suite.Path())
}

func (suite *BoltBenchmarkSchedulesTestSuite) TestSchedules() {
	schedules := []*v1.BenchmarkSchedule{
		{
			Name:         "bench1",
			StartTime:    ptypes.TimestampNow(),
			IntervalDays: 1,
			Clusters:     []string{"dev"},
		},
		{
			Name:         "bench2",
			StartTime:    ptypes.TimestampNow(),
			IntervalDays: 2,
			Clusters:     []string{"prod"},
		},
	}

	// Test Add
	for _, b := range schedules {
		suite.NoError(suite.AddBenchmarkSchedule(b))
	}

	for _, b := range schedules {
		got, exists, err := suite.GetBenchmarkSchedule(b.Name)
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, b)
	}

	// Test Update
	for _, b := range schedules {
		b.Clusters = []string{"integration"}
	}

	for _, b := range schedules {
		suite.NoError(suite.UpdateBenchmarkSchedule(b))
	}

	for _, b := range schedules {
		got, exists, err := suite.GetBenchmarkSchedule(b.GetName())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, b)
	}

	// Test Remove
	for _, b := range schedules {
		suite.NoError(suite.RemoveBenchmarkSchedule(b.GetName()))
	}

	for _, b := range schedules {
		_, exists, err := suite.GetBenchmarkSchedule(b.GetName())
		suite.NoError(err)
		suite.False(exists)
	}
}

/*
	benchmarkS

	// Test Add
	for _, b := range benchmarkSchedules {
		assert.NoError(t, insertStorage.AddBenchmarkSchedule(b))
	}
	// Verify insertion multiple times does not deadlock and causes an error
	for _, b := range benchmarkSchedules {
		assert.Error(t, insertStorage.AddBenchmarkSchedule(b))
	}

	for _, b := range benchmarkSchedules {
		got, exists, err := retrievalStorage.GetBenchmarkSchedule(b.Name)
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, b)
	}
*/
