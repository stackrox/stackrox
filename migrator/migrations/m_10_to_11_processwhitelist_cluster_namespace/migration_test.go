package m10to11

import (
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(MigrationTestSuite))
}

type MigrationTestSuite struct {
	suite.Suite

	db *bolt.DB
}

func (suite *MigrationTestSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.NoError(db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(pWBucketName); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(pWResultsBucketName); err != nil {
			return err
		}
		_, err := tx.CreateBucketIfNotExists(listDeploymentBucketName)
		return err
	}))
	suite.db = db
}

func (suite *MigrationTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
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
	pWBucket := bolthelpers.TopLevelRef(suite.db, pWBucketName)
	suite.NoError(insertThing(pWBucket, pw.GetId(), pw))
}

func (suite *MigrationTestSuite) mustInsertProcessWhitelistResults(pwr *storage.ProcessWhitelistResults) {
	pWResultsBucket := bolthelpers.TopLevelRef(suite.db, pWResultsBucketName)
	suite.NoError(insertThing(pWResultsBucket, pwr.GetDeploymentId(), pwr))
}

func (suite *MigrationTestSuite) mustInsertDeployment(deployment *storage.ListDeployment) {
	deploymentBucket := bolthelpers.TopLevelRef(suite.db, listDeploymentBucketName)
	suite.NoError(insertThing(deploymentBucket, deployment.GetId(), deployment))
}

func (suite *MigrationTestSuite) TestProcessWhitelistMigration() {
	oldProcessWhitelist := []*storage.ProcessWhitelist{
		{Id: "1",
			Key: &storage.ProcessWhitelistKey{
				DeploymentId:  "A",
				ContainerName: "A",
			},
		},
		{Id: "2",
			Key: &storage.ProcessWhitelistKey{
				DeploymentId:  "B",
				ContainerName: "B",
			},
		},
		{Id: "3",
			Key: &storage.ProcessWhitelistKey{
				DeploymentId:  "C",
				ContainerName: "C",
			},
		},
		{Id: "4",
			Key: &storage.ProcessWhitelistKey{
				DeploymentId:  "Nonexistent",
				ContainerName: "Nonexistent",
			},
		},
	}

	oldProcessWhitelistResults := []*storage.ProcessWhitelistResults{
		{DeploymentId: "A"},
		{DeploymentId: "B"},
		{DeploymentId: "C"},
		{DeploymentId: "Nonexistent"},
	}

	listDeployments := []*storage.ListDeployment{
		{Id: "A", ClusterId: "1", Namespace: "1"},
		{Id: "B", ClusterId: "1", Namespace: "2"},
		{Id: "C", ClusterId: "2", Namespace: "3"},
	}

	expectedPW := []*storage.ProcessWhitelist{
		{Id: "1",
			Key: &storage.ProcessWhitelistKey{
				DeploymentId:  "A",
				ContainerName: "A",
				ClusterId:     "1",
				Namespace:     "1",
			},
		},
		{Id: "2",
			Key: &storage.ProcessWhitelistKey{
				DeploymentId:  "B",
				ContainerName: "B",
				ClusterId:     "1",
				Namespace:     "2",
			},
		},
		{Id: "3",
			Key: &storage.ProcessWhitelistKey{
				DeploymentId:  "C",
				ContainerName: "C",
				ClusterId:     "2",
				Namespace:     "3",
			},
		},
		{Id: "4",
			Key: &storage.ProcessWhitelistKey{
				DeploymentId:  "Nonexistent",
				ContainerName: "Nonexistent",
			},
		},
	}
	expectedPWR := []*storage.ProcessWhitelistResults{
		{
			DeploymentId: "A",
			ClusterId:    "1",
			Namespace:    "1",
		},
		{
			DeploymentId: "B",
			ClusterId:    "1",
			Namespace:    "2",
		},
		{
			DeploymentId: "C",
			ClusterId:    "2",
			Namespace:    "3",
		},
		{DeploymentId: "Nonexistent"},
	}
	for _, pw := range oldProcessWhitelist {
		suite.mustInsertProcessWhitelist(pw)
	}

	for _, pwr := range oldProcessWhitelistResults {
		suite.mustInsertProcessWhitelistResults(pwr)
	}

	for _, listDeployment := range listDeployments {
		suite.mustInsertDeployment(listDeployment)
	}

	suite.NoError(migration.Run(suite.db, nil))

	newProcessWhitelist := make([]*storage.ProcessWhitelist, 0, len(oldProcessWhitelist))
	newPWBucket := bolthelpers.TopLevelRef(suite.db, newPWBucketName)
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
	suite.ElementsMatch(expectedPW, newProcessWhitelist)

	newProcessWhitelistResults := make([]*storage.ProcessWhitelistResults, 0, len(oldProcessWhitelistResults))
	pWResultsBucket := bolthelpers.TopLevelRef(suite.db, pWResultsBucketName)
	suite.NoError(pWResultsBucket.View(func(b *bolt.Bucket) error {
		return b.ForEach(func(_, v []byte) error {
			var processWhitelistResults storage.ProcessWhitelistResults
			err := proto.Unmarshal(v, &processWhitelistResults)
			if err != nil {
				return err
			}
			newProcessWhitelistResults = append(newProcessWhitelistResults, &processWhitelistResults)
			return nil
		})
	}))
	suite.ElementsMatch(expectedPWR, newProcessWhitelistResults)
}
