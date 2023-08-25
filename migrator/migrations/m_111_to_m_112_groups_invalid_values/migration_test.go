package m111tom112

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
	suite.Run(t, new(removeGroupsMigrationSuite))
}

type removeGroupsMigrationSuite struct {
	suite.Suite

	db *bolt.DB
}

func (suite *removeGroupsMigrationSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	suite.Require().NoError(err, "failed to make BoltDB")
	suite.db = db
}

func (suite *removeGroupsMigrationSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func (suite *removeGroupsMigrationSuite) TestMigrate() {
	// Expected groups after migration. Note that:
	// * Group "r1" should not be removed, as it is by ID,
	// * Group "r2" should be removed, as it has an empty role name,
	// * Group "r3" should be removed, as it has nil properties,
	// * Group "r4" should be removed, as it has nil properties and an empty role name.
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
			RoleName: "",
		},
		"r3": {
			Props:    nil,
			RoleName: "r3",
		},
		"r4": {
			Props:    nil,
			RoleName: "",
		},
	}

	cases := []struct {
		storedGroup *storage.Group
		newGroup    *storage.Group
		oldValue    bool
	}{
		{
			storedGroup: expectedGroups["r2"],
			newGroup:    expectedGroups["r2"],
			oldValue:    true,
		},
		{
			storedGroup: expectedGroups["r1"],
			newGroup:    expectedGroups["r1"],
		},
		{
			storedGroup: expectedGroups["r3"],
			newGroup:    expectedGroups["r3"],
			oldValue:    true,
		},
		{
			storedGroup: expectedGroups["r4"],
			newGroup:    expectedGroups["r4"],
			oldValue:    true,
		},
	}

	// 1. Migration should succeed if the bucket does not exist.
	suite.NoError(removeGroupsWithInvalidValues(suite.db))

	// 2. Add the old groups to the groups bucket and create it if it does not exist yet.
	err := suite.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucketName)
		suite.NoError(err)
		for _, c := range cases {
			var key, value []byte
			if c.oldValue {
				key, value = serialize(c.storedGroup)
			} else {
				key = []byte(c.storedGroup.GetProps().GetId())
				value, err = c.storedGroup.Marshal()
				suite.NoError(err)
			}

			suite.NoError(bucket.Put(key, value))
		}
		return nil
	})
	suite.NoError(err)

	// 3. Migrate the groups without ID.
	suite.NoError(removeGroupsWithInvalidValues(suite.db))

	// 4. Verify the expected group can be found
	err = suite.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		err := bucket.ForEach(func(k, v []byte) error {
			var group storage.Group
			suite.NoError(proto.Unmarshal(v, &group))
			expectedGroup := expectedGroups["r1"]
			suite.Equal(expectedGroup.GetRoleName(), group.GetRoleName())
			suite.Equal(expectedGroup.GetProps().GetAuthProviderId(), group.GetProps().GetAuthProviderId())
			suite.Contains(group.GetProps().GetId(), expectedGroup.GetProps().GetId())
			return nil
		})
		suite.NoError(err)
		return nil
	})
	suite.NoError(err)

	// 5. Verify all other groups are removed.
	err = suite.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		for _, c := range cases {
			if !c.oldValue {
				continue
			}

			key, _ := serialize(c.storedGroup)

			val := bucket.Get(key)
			suite.Nil(val)
		}

		return nil
	})
	suite.NoError(err)
}
