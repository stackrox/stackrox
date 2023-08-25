package m102tom103

import (
	"strconv"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrateServiceIdentitySerial))
}

type migrateServiceIdentitySerial struct {
	suite.Suite

	db *bolt.DB
}

func (suite *migrateServiceIdentitySerial) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.db = db
}

func (suite *migrateServiceIdentitySerial) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func (suite *migrateServiceIdentitySerial) TestMigrate() {
	// Buckets don't exist should succeed still
	suite.NoError(migrateSerials(suite.db))

	cases := []struct {
		oldSerial *storage.ServiceIdentity
		newSerial *storage.ServiceIdentity
	}{
		{
			oldSerial: &storage.ServiceIdentity{
				Srl: &storage.ServiceIdentity_Serial{
					Serial: 12345,
				},
				Id: "case1",
			},
			newSerial: &storage.ServiceIdentity{
				SerialStr: "12345",
				Srl: &storage.ServiceIdentity_Serial{
					Serial: 12345,
				},
				Id: "case1",
			},
		},
		{
			oldSerial: &storage.ServiceIdentity{
				SerialStr: "ABC",
				Id:        "case2",
			},
			newSerial: &storage.ServiceIdentity{
				SerialStr: "ABC",
				Id:        "case2",
			},
		},
	}
	err := suite.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucketName)
		suite.NoError(err)
		for _, c := range cases {
			si, err := c.oldSerial.Marshal()
			suite.NoError(err)

			if c.oldSerial.GetSerialStr() != "" {
				suite.NoError(bucket.Put([]byte(c.oldSerial.GetSerialStr()), si))
			} else {
				suite.NoError(bucket.Put([]byte(strconv.FormatInt(c.oldSerial.GetSerial(), 10)), si))
			}
		}
		return nil
	})
	suite.NoError(err)

	suite.NoError(migrateSerials(suite.db))

	err = suite.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)

		idx := 0
		err := bucket.ForEach(func(k, v []byte) error {
			var si storage.ServiceIdentity
			suite.NoError(proto.Unmarshal(v, &si))
			suite.Equal(cases[idx].newSerial, &si)
			idx++
			return nil
		})
		suite.NoError(err)
		return nil
	})
	suite.NoError(err)
}
