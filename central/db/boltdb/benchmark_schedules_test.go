package boltdb

import (
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/uuid"
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
	db, err := boltFromTmpDir()
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
	cluster1 := uuid.NewV4().String()
	cluster2 := uuid.NewV4().String()
	schedules := []*v1.BenchmarkSchedule{
		{
			Name:       "bench1",
			ClusterIds: []string{cluster1},
		},
		{
			Name:       "bench2",
			ClusterIds: []string{cluster2},
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
	changedCluster := uuid.NewV4().String()
	for _, b := range schedules {
		b.ClusterIds = []string{changedCluster}
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
