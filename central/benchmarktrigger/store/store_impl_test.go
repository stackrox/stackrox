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
	suite.Require().NoError(err, "failed to create BoltDB")

	suite.db = db
	suite.store = New(db)
}

func (suite *BenchmarkTriggerStoreTestSuite) TearDownTest() {
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

	suite.ElementsMatch(triggers, actualTriggers)
}
