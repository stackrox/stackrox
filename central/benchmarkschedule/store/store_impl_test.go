package store

import (
	"os"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestBenchmarkScheduleStore(t *testing.T) {
	suite.Run(t, new(BenchmarkScheduleStoreTestSuite))
}

type BenchmarkScheduleStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *BenchmarkScheduleStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *BenchmarkScheduleStoreTestSuite) TeardownSuite() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *BenchmarkScheduleStoreTestSuite) TestSchedules() {
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
		id, err := suite.store.AddBenchmarkSchedule(b)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	for _, b := range schedules {
		got, exists, err := suite.store.GetBenchmarkSchedule(b.GetId())
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
		suite.NoError(suite.store.UpdateBenchmarkSchedule(b))
	}

	for _, b := range schedules {
		got, exists, err := suite.store.GetBenchmarkSchedule(b.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, b)
	}

	// Test Remove
	for _, b := range schedules {
		suite.NoError(suite.store.RemoveBenchmarkSchedule(b.GetId()))
	}

	for _, b := range schedules {
		_, exists, err := suite.store.GetBenchmarkSchedule(b.GetId())
		suite.NoError(err)
		suite.False(exists)
	}
}
