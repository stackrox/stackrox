package m8to9

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
		if _, err := tx.CreateBucketIfNotExists(clusterBucketName); err != nil {
			return err
		}
		_, err := tx.CreateBucketIfNotExists(listAlertBucketName)
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

func (suite *migrationTestSuite) mustInsertCluster(cluster *storage.Cluster) {
	indicatorBucket := bolthelpers.TopLevelRef(suite.db, clusterBucketName)
	suite.NoError(insertThing(indicatorBucket, cluster.GetId(), cluster))
}

func (suite *migrationTestSuite) mustInsertListAlert(listAlert *storage.ListAlert) {
	deploymentBucket := bolthelpers.TopLevelRef(suite.db, listAlertBucketName)
	suite.NoError(insertThing(deploymentBucket, listAlert.GetId(), listAlert))
}

func (suite *migrationTestSuite) TestProcessIndicatorMigration() {
	oldListAlerts := []*storage.ListAlert{
		{Id: "1", Deployment: &storage.ListAlertDeployment{ClusterName: "c1Name"}},
		{Id: "2", Deployment: &storage.ListAlertDeployment{ClusterName: "c1Name"}},
		{Id: "3", Deployment: &storage.ListAlertDeployment{ClusterName: "c2Name"}},
		{Id: "4", Deployment: &storage.ListAlertDeployment{ClusterName: "c3Name"}},
	}
	clusters := []*storage.Cluster{
		{Id: "c1ID", Name: "c1Name"},
		{Id: "c2ID", Name: "c2Name"},
		{Id: "c3ID", Name: "c3Name"},
		{Id: "c4ID", Name: "c4Name"},
	}
	expectedListAlerts := []*storage.ListAlert{
		{Id: "1", Deployment: &storage.ListAlertDeployment{ClusterName: "c1Name", ClusterId: "c1ID"}},
		{Id: "2", Deployment: &storage.ListAlertDeployment{ClusterName: "c1Name", ClusterId: "c1ID"}},
		{Id: "3", Deployment: &storage.ListAlertDeployment{ClusterName: "c2Name", ClusterId: "c2ID"}},
		{Id: "4", Deployment: &storage.ListAlertDeployment{ClusterName: "c3Name", ClusterId: "c3ID"}},
	}

	for _, listAlert := range oldListAlerts {
		suite.mustInsertListAlert(listAlert)
	}

	for _, cluster := range clusters {
		suite.mustInsertCluster(cluster)
	}

	suite.NoError(migration.Run(suite.db, nil))

	newListAlerts := make([]*storage.ListAlert, 0, len(oldListAlerts))
	listAlertBucket := bolthelpers.TopLevelRef(suite.db, listAlertBucketName)
	suite.NoError(listAlertBucket.View(func(b *bolt.Bucket) error {
		return b.ForEach(func(_, v []byte) error {
			listAlert := new(storage.ListAlert)
			err := proto.Unmarshal(v, listAlert)
			if err != nil {
				return err
			}
			newListAlerts = append(newListAlerts, listAlert)
			return nil
		})
	}))
	suite.ElementsMatch(expectedListAlerts, newListAlerts)
}
