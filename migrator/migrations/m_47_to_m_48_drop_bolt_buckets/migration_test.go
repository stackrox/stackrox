package m47tom48

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/migrator/bolthelpers"
	dbTypes "github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(dropBucketMigrationSuite))
}

type dropBucketMigrationSuite struct {
	suite.Suite

	databases *dbTypes.Databases
}

func (suite *dropBucketMigrationSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	suite.NoError(err)

	suite.databases = &dbTypes.Databases{BoltDB: db}
}

func (suite *dropBucketMigrationSuite) TearDownTest() {
	testutils.TearDownDB(suite.databases.BoltDB)
}

func (suite *dropBucketMigrationSuite) TestDeleteBuckets() {
	// Create and write into half the buckets. Tests that a nonexistent bucket doesn't error
	for i := 0; i < len(bucketsToBeDropped)/2; i++ {
		bucket := bucketsToBeDropped[i]
		err := suite.databases.BoltDB.Update(func(tx *bolt.Tx) error {
			bucket, err := tx.CreateBucket([]byte(bucket))
			if err != nil {
				return err
			}
			for i := 0; i < 100; i++ {
				data := []byte(fmt.Sprintf("%d", i))
				if err := bucket.Put(data, data); err != nil {
					return err
				}
			}
			return nil
		})
		suite.NoError(err)
	}

	suite.NoError(dropBoltBuckets(suite.databases))

	err := suite.databases.BoltDB.View(func(tx *bolt.Tx) error {
		for _, bucket := range bucketsToBeDropped {
			suite.Nil(tx.Bucket([]byte(bucket)))
		}
		return nil
	})
	suite.NoError(err)
}
