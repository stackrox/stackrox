package m105tom106

import (
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
	suite.Require().NoError(err, "failed to make BoltDB")
	suite.db = db
}

func (suite *migrateServiceIdentitySerial) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func (suite *migrateServiceIdentitySerial) TestMigrate() {
	// Expected groups after migration. Note that:
	// * Group "r1" should not be updated, as it is by ID,
	// * Group "r2" should get an ID after migration,
	// * Group "r3" should be migrated despite it has an ID because it is stored by the composite key.
	expectedGroups := map[string]*storage.Group{
		"r1": {
			Props: &storage.GroupProperties{
				AuthProviderId: "something",
				Id:             "io.stackrox.authz.group.",
			},
			RoleName: "r1",
		},
		"r2": {
			Props: &storage.GroupProperties{
				AuthProviderId: "something",
				Id:             "io.stackrox.authz.group.migrated.",
			},
			RoleName: "r2",
		},
		"r3": {
			Props: &storage.GroupProperties{
				AuthProviderId: "something",
				Id:             "io.stackrox.authz.group.",
			},
			RoleName: "r3",
		},
	}

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
				RoleName: "r2",
			},
			newGroup: expectedGroups["r2"],
			oldValue: true,
		},
		{
			oldGroup: expectedGroups["r1"],
			newGroup: expectedGroups["r1"],
		},
		{
			oldGroup: expectedGroups["r3"],
			newGroup: expectedGroups["r3"],
			oldValue: true,
		},
	}

	// 1. Buckets don't exist should succeed still
	suite.NoError(migrateGroupsStoredByCompositeKey(suite.db))

	// 2. Add the old groups to the groups bucket and create it if it does not exist yet.
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

	// 3. Migrate the groups without ID.
	suite.NoError(migrateGroupsStoredByCompositeKey(suite.db))

	// 4. Verify all groups match the expected groups.
	err = suite.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		err := bucket.ForEach(func(k, v []byte) error {
			var group storage.Group
			suite.NoError(proto.Unmarshal(v, &group))
			expectedGroup, exists := expectedGroups[group.GetRoleName()]
			suite.True(exists)
			suite.Equal(expectedGroup.GetRoleName(), group.GetRoleName())
			suite.Equal(expectedGroup.GetProps().GetAuthProviderId(), group.GetProps().GetAuthProviderId())
			suite.Contains(group.GetProps().GetId(), expectedGroup.GetProps().GetId())
			return nil
		})
		suite.NoError(err)
		return nil
	})
	suite.NoError(err)

	// 5. Verify all old keys do not exist anymore.
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
