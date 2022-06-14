package m102tom103

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/binenc"
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
	suite.NoError(addIDToGroups(suite.db))

	cases := []struct {
		oldGroup *storage.Group
		newGroup *storage.Group
		oldValue bool
	}{
		{
			oldGroup: &storage.Group{
				Props: &storage.GroupProperties{
					AuthProviderId: "something",
				},
				RoleName: "r1",
			},
			newGroup: &storage.Group{
				Props: &storage.GroupProperties{
					AuthProviderId: "something",
					Id:             "io.stackrox.group.",
				},
				RoleName: "r1",
			},
			oldValue: true,
		},
		{
			oldGroup: &storage.Group{
				Props: &storage.GroupProperties{
					AuthProviderId: "something",
					Id:             "io.stackrox.group.",
				},
				RoleName: "r1",
			},
			newGroup: &storage.Group{
				Props: &storage.GroupProperties{
					AuthProviderId: "something",
					Id:             "io.stackrox.group.",
				},
				RoleName: "r1",
			},
		},
	}
	err := suite.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucketName)
		suite.NoError(err)
		for _, c := range cases {
			var key, value []byte
			if c.oldValue {
				key, value = serialize(c.oldGroup)
			} else {
				key = []byte(c.oldGroup.GetProps().GetId())
				value, err = c.oldGroup.Marshal()
				suite.NoError(err)
			}

			suite.NoError(bucket.Put(key, value))
		}
		return nil
	})
	suite.NoError(err)

	suite.NoError(addIDToGroups(suite.db))

	err = suite.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)

		idx := 0
		err := bucket.ForEach(func(k, v []byte) error {
			var group storage.Group
			suite.NoError(proto.Unmarshal(v, &group))
			suite.Equal(cases[idx].newGroup.GetRoleName(), group.GetRoleName())
			suite.Equal(cases[idx].newGroup.GetProps().GetAuthProviderId(), group.GetProps().GetAuthProviderId())
			suite.Contains(group.GetProps().GetId(), cases[idx].newGroup.GetProps().GetId())
			idx++
			return nil
		})
		suite.NoError(err)
		return nil
	})
	suite.NoError(err)

	err = suite.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		for _, c := range cases {
			if !c.oldValue {
				continue
			}

			key, _ := serialize(c.oldGroup)

			val := bucket.Get(key)
			suite.Nil(val)
		}

		return nil
	})
	suite.NoError(err)
}

func serialize(grp *storage.Group) ([]byte, []byte) {
	key := binenc.EncodeBytesList([]byte(grp.GetProps().GetAuthProviderId()), []byte(grp.GetProps().GetKey()),
		[]byte(grp.GetProps().GetValue()))

	value := []byte(grp.GetRoleName())

	return key, value
}
