package m7to8

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
	suite.Run(t, new(migrationTestSuite))
}

type migrationTestSuite struct {
	suite.Suite

	db *bolt.DB
}

func (suite *migrationTestSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.NoError(db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(processIndicatorBucketName); err != nil {
			return err
		}
		_, err := tx.CreateBucketIfNotExists(deploymentBucketName)
		return err
	}))
	suite.db = db
}

func (suite *migrationTestSuite) TearDownTest() {
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

func (suite *migrationTestSuite) mustInsertProcessIndicator(indicator *storage.ProcessIndicator) {
	indicatorBucket := bolthelpers.TopLevelRef(suite.db, processIndicatorBucketName)
	suite.NoError(insertThing(indicatorBucket, indicator.GetId(), indicator))
}

func (suite *migrationTestSuite) mustInsertDeployment(deployment *storage.Deployment) {
	deploymentBucket := bolthelpers.TopLevelRef(suite.db, deploymentBucketName)
	suite.NoError(insertThing(deploymentBucket, deployment.GetId(), deployment))
}

func (suite *migrationTestSuite) TestProcessIndicatorMigration() {
	oldIndicators := []*storage.ProcessIndicator{
		{Id: "1", DeploymentId: "A"},
		{Id: "2", DeploymentId: "A"},
		{Id: "3", DeploymentId: "B"},
		{Id: "4", DeploymentId: "C"},
		// This deployment won't exist to test stale process indicators
		{Id: "5", DeploymentId: "Nonexistent"},
	}
	deployments := []*storage.Deployment{
		{Id: "A", ClusterId: "1", Namespace: "1"},
		{Id: "B", ClusterId: "1", Namespace: "2"},
		{Id: "C", ClusterId: "2", Namespace: "3"},
	}
	expectedResults := []*storage.ProcessIndicator{
		{Id: "1", DeploymentId: "A", ClusterId: "1", Namespace: "1"},
		{Id: "2", DeploymentId: "A", ClusterId: "1", Namespace: "1"},
		{Id: "3", DeploymentId: "B", ClusterId: "1", Namespace: "2"},
		{Id: "4", DeploymentId: "C", ClusterId: "2", Namespace: "3"},
		{Id: "5", DeploymentId: "Nonexistent"},
	}

	for _, indicator := range oldIndicators {
		suite.mustInsertProcessIndicator(indicator)
	}

	for _, deployment := range deployments {
		suite.mustInsertDeployment(deployment)
	}

	suite.NoError(migration.Run(suite.db, nil))

	newIndicators := make([]*storage.ProcessIndicator, 0, len(oldIndicators))
	indicatorBucket := bolthelpers.TopLevelRef(suite.db, processIndicatorBucketName)
	suite.NoError(indicatorBucket.View(func(b *bolt.Bucket) error {
		return b.ForEach(func(_, v []byte) error {
			indicator := new(storage.ProcessIndicator)
			err := proto.Unmarshal(v, indicator)
			if err != nil {
				return err
			}
			newIndicators = append(newIndicators, indicator)
			return nil
		})
	}))
	suite.ElementsMatch(expectedResults, newIndicators)
}
