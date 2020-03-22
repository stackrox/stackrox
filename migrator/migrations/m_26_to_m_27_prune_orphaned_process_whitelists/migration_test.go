package m26to27

import (
	"fmt"
	"testing"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/badgerhelpers"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

const (
	depsN = 10
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(MigrationTestSuite))
}

type MigrationTestSuite struct {
	suite.Suite

	boltdb   *bolt.DB
	badgerdb *badger.DB
}

func (suite *MigrationTestSuite) SetupTest() {
	boltdb, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.NoError(boltdb.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(whitelistBucket); err != nil {
			return err
		}
		return err
	}))

	suite.boltdb = boltdb
	suite.badgerdb, err = badgerhelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BadgerDB", err.Error())
	}
}

func (suite *MigrationTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.boltdb)
	_ = suite.badgerdb.Close()
}

func insertThing(bucket bolthelpers.BucketRef, id string, pb proto.Message) error {
	return bucket.Update(func(b *bolt.Bucket) error {
		bytes, err := proto.Marshal(pb)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), bytes)
	})
}

func (suite *MigrationTestSuite) mustInsertProcessWhitelist(pw *storage.ProcessWhitelist) {
	pWBucket := bolthelpers.TopLevelRef(suite.boltdb, whitelistBucket)
	suite.NoError(insertThing(pWBucket, pw.GetId(), pw))
}

func (suite *MigrationTestSuite) addDeploymentIds() {
	var depIds []string
	for idx := 1; idx < depsN; idx++ {
		depIds = append(depIds, fmt.Sprintf("dep-%d", idx))
	}

	batch := suite.badgerdb.NewWriteBatch()
	suite.NoError(batch.Error())

	defer batch.Cancel()
	for _, depID := range depIds {
		key := make([]byte, 0, len(deploymentBucket)+len(depID)+1)
		key = append(key, deploymentBucket...)
		key = append(key, []byte(depID)...)

		err := batch.Set(key, []byte{})
		suite.NoError(err)
	}

	suite.NoError(batch.Flush())
}

func (suite *MigrationTestSuite) TestProcessWhitelistPruning() {
	suite.addDeploymentIds()

	oldPWLs := []*storage.ProcessWhitelist{
		{Id: "1",
			Key: &storage.ProcessWhitelistKey{
				DeploymentId:  "A",
				ContainerName: "A",
				ClusterId:     "1",
				Namespace:     "1",
			},
			Created: &types.Timestamp{
				Seconds: time.Now().Add(-orphanWindow).Unix(),
			},
		},
		{Id: "2",
			Key: &storage.ProcessWhitelistKey{
				DeploymentId:  "dep-1",
				ContainerName: "C",
				ClusterId:     "2",
				Namespace:     "3",
			},
		},
		{Id: "3",
			Key: &storage.ProcessWhitelistKey{
				DeploymentId:  "dep-5",
				ContainerName: "C",
				ClusterId:     "2",
				Namespace:     "3",
			},
		},
		{Id: "4",
			Key: &storage.ProcessWhitelistKey{
				DeploymentId:  "B",
				ContainerName: "B",
				ClusterId:     "1",
				Namespace:     "2",
			},
		},
		{Id: "5",
			Key: &storage.ProcessWhitelistKey{
				DeploymentId:  "dep-9",
				ContainerName: "C",
				ClusterId:     "2",
				Namespace:     "3",
			},
		},
	}

	expectedPWLs := []*storage.ProcessWhitelist{
		{Id: "2",
			Key: &storage.ProcessWhitelistKey{
				DeploymentId:  "dep-1",
				ContainerName: "C",
				ClusterId:     "2",
				Namespace:     "3",
			},
		},
		{Id: "3",
			Key: &storage.ProcessWhitelistKey{
				DeploymentId:  "dep-5",
				ContainerName: "C",
				ClusterId:     "2",
				Namespace:     "3",
			},
		},
		{Id: "4",
			Key: &storage.ProcessWhitelistKey{
				DeploymentId:  "B",
				ContainerName: "B",
				ClusterId:     "1",
				Namespace:     "2",
			},
		},
		{Id: "5",
			Key: &storage.ProcessWhitelistKey{
				DeploymentId:  "dep-9",
				ContainerName: "C",
				ClusterId:     "2",
				Namespace:     "3",
			},
		},
	}

	for _, pw := range oldPWLs {
		suite.mustInsertProcessWhitelist(pw)
	}

	suite.NoError(migration.Run(suite.boltdb, suite.badgerdb))

	newProcessWhitelist := make([]*storage.ProcessWhitelist, 0, len(oldPWLs))
	newPWBucket := bolthelpers.TopLevelRef(suite.boltdb, whitelistBucket)
	suite.NoError(newPWBucket.View(func(b *bolt.Bucket) error {
		return b.ForEach(func(_, v []byte) error {
			var processWhitelist storage.ProcessWhitelist
			err := proto.Unmarshal(v, &processWhitelist)
			if err != nil {
				return err
			}
			newProcessWhitelist = append(newProcessWhitelist, &processWhitelist)
			return nil
		})
	}))

	suite.ElementsMatch(expectedPWLs, newProcessWhitelist)
}
