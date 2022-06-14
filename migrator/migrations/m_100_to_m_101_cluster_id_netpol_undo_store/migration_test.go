package m100tom101

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(undostoreClusterIDTestSuite))
}

type undostoreClusterIDTestSuite struct {
	suite.Suite

	db *bolt.DB
}

func (suite *undostoreClusterIDTestSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.db = db
}

func (suite *undostoreClusterIDTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func (suite *undostoreClusterIDTestSuite) upsert(clusterID string, record *storage.NetworkPolicyApplicationUndoRecord) {
	serialized, err := proto.Marshal(record)
	suite.NoError(err)

	clusterKey := []byte(clusterID)
	err = suite.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucket)
		suite.NoError(err)

		return bucket.Put(clusterKey, serialized)
	})
	suite.NoError(err)
}

func (suite *undostoreClusterIDTestSuite) TestMigrate() {
	records := []struct {
		record    *storage.NetworkPolicyApplicationUndoRecord
		clusterID string
	}{
		{
			record: &storage.NetworkPolicyApplicationUndoRecord{
				User: "abc",
			},
			clusterID: "1",
		},
		{
			record: &storage.NetworkPolicyApplicationUndoRecord{
				User: "abcd",
			},
			clusterID: "2",
		},
	}
	for _, r := range records {
		suite.upsert(r.clusterID, r.record)

		// Set the cluster id for verification after the migration
		r.record.ClusterId = r.clusterID
	}
	err := addClusterIDToNetworkPolicyApplicationUndoRecord(suite.db)
	suite.NoError(err)

	var idx int
	err = suite.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucket)
		return bucket.ForEach(func(_, v []byte) error {
			var record storage.NetworkPolicyApplicationUndoRecord
			suite.NoError(proto.Unmarshal(v, &record))
			suite.Equal(records[idx].record, &record)
			idx++
			return nil
		})
	})
	suite.NoError(err)
}
