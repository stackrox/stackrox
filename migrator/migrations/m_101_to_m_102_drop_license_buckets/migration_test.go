package m101tom102

import (
	"testing"

	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(dropLicenseBuckets))
}

type dropLicenseBuckets struct {
	suite.Suite

	db *bolt.DB
}

func (suite *dropLicenseBuckets) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.db = db
}

func (suite *dropLicenseBuckets) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func (suite *dropLicenseBuckets) bucketsExist() bool {
	exists := true
	err := suite.db.View(func(tx *bolt.Tx) error {
		for _, bucket := range buckets {
			if tx.Bucket([]byte(bucket)) == nil {
				exists = false
				return nil
			}
		}
		return nil
	})
	suite.NoError(err)
	return exists
}

func (suite *dropLicenseBuckets) TestMigrate() {
	// Buckets don't exist should succeed still
	suite.NoError(dropBuckets(suite.db))

	err := suite.db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range buckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
				return err
			}
		}
		return nil
	})
	suite.NoError(err)

	suite.True(suite.bucketsExist())
	suite.NoError(dropBuckets(suite.db))
	suite.False(suite.bucketsExist())
}
