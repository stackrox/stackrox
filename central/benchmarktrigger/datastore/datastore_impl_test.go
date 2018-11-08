package datastore

import (
	"os"
	"testing"

	"github.com/boltdb/bolt"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/benchmarktrigger/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestBenchmarkTriggerDataStore(t *testing.T) {
	suite.Run(t, new(BenchmarkTriggerDataStoreTestSuite))
}

type BenchmarkTriggerDataStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store     store.Store
	datastore DataStore
}

func (suite *BenchmarkTriggerDataStoreTestSuite) SetupTest() {
	db, err := bolthelper.NewTemp("BenchmarkTriggerDataStoreTestSuite.db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = store.New(db)
	suite.datastore = New(suite.store)
}

func (suite *BenchmarkTriggerDataStoreTestSuite) TeardownTest() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *BenchmarkTriggerDataStoreTestSuite) TestBenchmarkTriggers() {
	triggerTime1 := ptypes.TimestampNow()
	triggerTime2 := ptypes.TimestampNow()
	triggerTime2.Seconds += 1000

	triggers := []*v1.BenchmarkTrigger{
		{
			Id:   "trigger1",
			Time: triggerTime1,
		},
		{
			Id:   "trigger2",
			Time: triggerTime2,
		},
	}

	// Test Add
	for _, trigger := range triggers {
		suite.NoError(suite.datastore.AddBenchmarkTrigger(trigger))
	}
}

func (suite *BenchmarkTriggerDataStoreTestSuite) TestBenchmarkTriggersFiltering() {
	triggerTime1 := ptypes.TimestampNow()
	triggerTime2 := ptypes.TimestampNow()
	triggerTime3 := ptypes.TimestampNow()
	triggerTime2.Seconds += 1000
	triggerTime3.Seconds += 2000

	cluster1 := uuid.NewV4().String()
	cluster2a := uuid.NewV4().String()
	cluster2b := uuid.NewV4().String()

	trigger1 := &v1.BenchmarkTrigger{
		Id:         "trigger1",
		Time:       triggerTime1,
		ClusterIds: []string{cluster1},
	}
	trigger2 := &v1.BenchmarkTrigger{
		Id:         "trigger2",
		Time:       triggerTime2,
		ClusterIds: []string{cluster2a, cluster2b},
	}
	// trigger with no cluster
	trigger3 := &v1.BenchmarkTrigger{
		Id:   "trigger3",
		Time: triggerTime3,
	}
	triggers := []*v1.BenchmarkTrigger{
		trigger1,
		trigger2,
		trigger3,
	}

	// Test Add
	for _, trigger := range triggers {
		suite.NoError(suite.datastore.AddBenchmarkTrigger(trigger))
	}

	actualTriggers, err := suite.datastore.GetBenchmarkTriggers(&v1.GetBenchmarkTriggersRequest{})
	suite.NoError(err)
	suite.Equal(triggers, actualTriggers)

	actualTriggers, err = suite.datastore.GetBenchmarkTriggers(&v1.GetBenchmarkTriggersRequest{
		Ids: []string{"trigger1"},
	})
	suite.NoError(err)
	suite.Equal([]*v1.BenchmarkTrigger{trigger1}, actualTriggers)

	actualTriggers, err = suite.datastore.GetBenchmarkTriggers(&v1.GetBenchmarkTriggersRequest{
		ClusterIds: []string{cluster1},
	})
	suite.NoError(err)
	suite.Equal([]*v1.BenchmarkTrigger{trigger1, trigger3}, actualTriggers)
}
