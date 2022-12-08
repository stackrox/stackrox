package m112tom113

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(recreateGroupsBucketMigrationTestSuite))
}

type recreateGroupsBucketMigrationTestSuite struct {
	suite.Suite

	db *bolt.DB
}

func (suite *recreateGroupsBucketMigrationTestSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	suite.Require().NoError(err, "failed to make BoltDB")
	suite.db = db
}

func (suite *recreateGroupsBucketMigrationTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func (suite *recreateGroupsBucketMigrationTestSuite) TestMigrate() {
	// existingGroup should not be dropped during re-creation.
	existingGroup := &storage.Group{
		Props: &storage.GroupProperties{
			Id:             "io.stackrox.authz.group." + uuid.NewV4().String(),
			AuthProviderId: "some-value",
		},
		RoleName: "some-value",
	}
	rawExistingGroup, err := proto.Marshal(existingGroup)
	suite.NoError(err)

	// migratedGroup should not be dropped during re-creation.
	migratedGroup := &storage.Group{
		Props: &storage.GroupProperties{
			Id:             "io.stackrox.authz.group.migrated." + uuid.NewV4().String(),
			AuthProviderId: "some-value",
		},
		RoleName: "some-value",
	}
	rawMigratedGroup, err := proto.Marshal(migratedGroup)
	suite.NoError(err)

	// invalidGroup should be dropped during re-creation.
	invalidGroup := &storage.Group{
		Props: &storage.GroupProperties{
			Key: "some-value",
		},
		RoleName: "",
	}
	rawInvalidGroup, err := proto.Marshal(invalidGroup)
	suite.NoError(err)

	// The following cases represent the entries within the groups bucket _before_ migration.
	// After migration, note that:
	// - existing-group should not have been dropped, due to having an ID.
	// - migrated-group should not have been dropped, due to having an ID.
	// - invalid-group should have been dropped, due to no ID.
	// - invalid-bytes should have been dropped, due to some weird data and no group proto message.
	cases := map[string]struct {
		entry                groupEntry
		existsAfterMigration bool
	}{
		"existing-group": {
			entry: groupEntry{
				key:   []byte(existingGroup.GetProps().GetId()),
				value: rawExistingGroup,
			},
			existsAfterMigration: true,
		},
		"migrated-group": {
			entry: groupEntry{
				key:   []byte(migratedGroup.GetProps().GetId()),
				value: rawMigratedGroup,
			},
			existsAfterMigration: true,
		},
		"invalid-group": {
			entry: groupEntry{
				key:   []byte("some-random-key"),
				value: rawInvalidGroup,
			},
		},
		"invalid-group-stored-by-id": {
			entry: groupEntry{
				key:   []byte(existingGroup.GetProps().GetId() + "make-it-unique"),
				value: rawInvalidGroup,
			},
		},
		"invalid-bytes": {
			entry: groupEntry{
				key:   []byte("some-random-bytes-no-one-knows"),
				value: []byte("some-other-random-bytes"),
			},
		},
	}

	var expectedEntriesAfterMigration int
	for _, c := range cases {
		if c.existsAfterMigration {
			expectedEntriesAfterMigration++
		}
	}

	// 1. Migration should succeed if the bucket does not exist.
	suite.NoError(recreateGroupsBucket(suite.db))

	// 2. Add the groups to the groups bucket before running the migration.
	err = suite.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucketName)
		suite.NoError(err)
		for _, c := range cases {
			suite.NoError(bucket.Put(c.entry.key, c.entry.value))
		}
		return nil
	})
	suite.NoError(err)

	// 3. Run the migration to re-create the groups bucket and remove invalid entries.
	suite.NoError(recreateGroupsBucket(suite.db))

	// 4. Verify that all entries are as expected.
	err = suite.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)

		for _, c := range cases {
			// In case the entry should not exist, it shouldn't be possible to retrieve any values from the given key.
			if !c.existsAfterMigration {
				suite.Empty(bucket.Get(c.entry.key))
			} else {
				// In case the entry should exist, it should match the expected value.
				value := bucket.Get(c.entry.key)
				suite.NotEmpty(value)
				suite.Equal(c.entry.value, value)
			}
		}
		return nil
	})
	suite.NoError(err)

	// 5. Verify that the entries count matches.
	var actualEntriesCount int
	err = suite.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)

		return bucket.ForEach(func(k, v []byte) error {
			actualEntriesCount++
			return nil
		})
	})
	suite.NoError(err)
	suite.Equal(expectedEntriesAfterMigration, actualEntriesCount)
}
