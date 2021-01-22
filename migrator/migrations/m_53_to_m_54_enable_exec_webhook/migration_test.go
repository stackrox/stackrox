package m53tom54

import (
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestExecWebhookMigration(t *testing.T) {
	suite.Run(t, new(execWebhookTestSuite))
}

type execWebhookTestSuite struct {
	suite.Suite

	db *bolt.DB
}

func (suite *execWebhookTestSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.NoError(db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(clustersBucket); err != nil {
			return err
		}
		return nil
	}))
	suite.db = db
}

func (suite *execWebhookTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func (suite *execWebhookTestSuite) TestMigrateClustersWithExecWebhooks() {
	clusters := []*storage.Cluster{
		{
			Id:   "1",
			Type: storage.ClusterType_OPENSHIFT_CLUSTER,
		},
		{
			Id:                        "2",
			Type:                      storage.ClusterType_OPENSHIFT_CLUSTER,
			AdmissionControllerEvents: true,
		},
		{
			Id:   "3",
			Type: storage.ClusterType_KUBERNETES_CLUSTER,
		},
		{
			Id:                        "4",
			Type:                      storage.ClusterType_KUBERNETES_CLUSTER,
			AdmissionControllerEvents: true,
		},
	}

	err := suite.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(clustersBucket)
		for _, cluster := range clusters {
			bytes, err := proto.Marshal(cluster)
			if err != nil {
				return err
			}
			if err := bucket.Put([]byte(cluster.GetId()), bytes); err != nil {
				return err
			}
		}
		return nil
	})
	suite.NoError(err)

	// Migrate the data
	suite.NoError(migrateExecWebhook(suite.db))

	expected := []*storage.Cluster{
		{
			Id:                        "1",
			Type:                      storage.ClusterType_OPENSHIFT_CLUSTER,
			AdmissionControllerEvents: false,
		},
		{
			Id:                        "2",
			Type:                      storage.ClusterType_OPENSHIFT_CLUSTER,
			AdmissionControllerEvents: true,
		},
		{
			Id:                        "3",
			Type:                      storage.ClusterType_KUBERNETES_CLUSTER,
			AdmissionControllerEvents: true,
		},
		{
			Id:                        "4",
			Type:                      storage.ClusterType_KUBERNETES_CLUSTER,
			AdmissionControllerEvents: true,
		},
	}

	err = suite.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(clustersBucket)
		for _, cluster := range expected {
			value := bucket.Get([]byte(cluster.GetId()))
			if len(value) == 0 {
				return fmt.Errorf("no value for id: %q", cluster.GetId())
			}
			var c storage.Cluster
			if err := proto.Unmarshal(value, &c); err != nil {
				return err
			}

			suite.Equal(cluster, &c)
		}
		return nil
	})
	suite.NoError(err)
}
