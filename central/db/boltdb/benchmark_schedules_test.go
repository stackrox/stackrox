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
			Id:          "id1",
			BenchmarkId: "bench1",
			ClusterIds:  []string{cluster1},
		},
		{
			Id:          "id2",
			BenchmarkId: "bench2",
			ClusterIds:  []string{cluster2},
		},
	}

	// Test Add
	for _, b := range schedules {
		id, err := suite.AddBenchmarkSchedule(b)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	for _, b := range schedules {
		got, exists, err := suite.GetBenchmarkSchedule(b.GetId())
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
		got, exists, err := suite.GetBenchmarkSchedule(b.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, b)
	}

	// Test Remove
	for _, b := range schedules {
		suite.NoError(suite.RemoveBenchmarkSchedule(b.GetId()))
	}

	for _, b := range schedules {
		_, exists, err := suite.GetBenchmarkSchedule(b.GetId())
		suite.NoError(err)
		suite.False(exists)
	}
}
