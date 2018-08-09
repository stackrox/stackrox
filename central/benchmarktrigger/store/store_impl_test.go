package store

import (
	"os"
	"testing"

	"github.com/boltdb/bolt"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stretchr/testify/suite"
)

func TestBenchmarkTriggerStore(t *testing.T) {
	suite.Run(t, new(BenchmarkTriggerStoreTestSuite))
}

type BenchmarkTriggerStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *BenchmarkTriggerStoreTestSuite) SetupTest() {
	db, err := bolthelper.NewTemp("BenchmarkTriggerStoreTestSuite.db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *BenchmarkTriggerStoreTestSuite) TeardownTest() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *BenchmarkTriggerStoreTestSuite) TestTriggers() {
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
		suite.NoError(suite.store.AddBenchmarkTrigger(trigger))
	}

	actualTriggers, err := suite.store.GetBenchmarkTriggers(&v1.GetBenchmarkTriggersRequest{})
	suite.NoError(err)

	suite.Equal(triggers, actualTriggers)
}
