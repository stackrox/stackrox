package groups

import (
	"os"
	"path"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/roxctl/common/mocks"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

const (
	groupUUID              = "3a4d0fe6-bd1f-4d46-9466-876cd2c335e0"
	groupIDForInvalidGroup = "io.stackrox.authz.group.90b89822-f3ee-430e-b6a3-ed5cbb2a765f"
	groupIDForInvalidBytes = "io.stackrox.authz.group.migrated.90b89822-f3ee-430e-b6a3-ed5cbb2a765f"
)

func TestGroupsRecreation(t *testing.T) {
	suite.Run(t, new(recreateGroupsTestSuite))
}

type recreateGroupsTestSuite struct {
	suite.Suite

	db *bolt.DB
}

func (s *recreateGroupsTestSuite) SetupTest() {
	db, err := bolthelper.NewTemp(testutils.DBFileName(s))
	s.Require().NoError(err, "failed to make BoltDB")
	s.db = db
}

func (s *recreateGroupsTestSuite) TearDownTest() {
	testutils.TearDownDB(s.db)
}

func (s *recreateGroupsTestSuite) TestRecreate() {
	// 1. Prepare the group entries that should be within the database.

	// existingGroup should not be dropped during re-creation.
	existingGroup := &storage.Group{
		Props: &storage.GroupProperties{
			Id:             "io.stackrox.authz.group." + groupUUID,
			AuthProviderId: "some-value",
		},
		RoleName: "some-value",
	}
	rawExistingGroup, err := proto.Marshal(existingGroup)
	s.NoError(err)

	// migratedGroup should not be dropped during re-creation.
	migratedGroup := &storage.Group{
		Props: &storage.GroupProperties{
			Id:             "io.stackrox.authz.group.migrated." + groupUUID,
			AuthProviderId: "some-value",
		},
		RoleName: "some-value",
	}
	rawMigratedGroup, err := proto.Marshal(migratedGroup)
	s.NoError(err)

	// invalidGroup should be dropped during re-creation.
	invalidGroup := &storage.Group{
		Props: &storage.GroupProperties{
			Key: "some-value",
		},
		RoleName: "",
	}
	rawInvalidGroup, err := proto.Marshal(invalidGroup)
	s.NoError(err)

	// The following cases represent the entries within the groups bucket _before_ migration.
	// After migration, note that:
	// - existing-group should not have been dropped, due to having an ID.
	// - migrated-group should not have been dropped, due to having an ID.
	// - invalid-group should have been dropped, due to no ID.
	// - invalid-bytes should have been dropped, due to some weird data and no group proto message.
	cases := map[string]struct {
		entry                bucketEntry
		existsAfterMigration bool
	}{
		"existing-group": {
			entry: bucketEntry{
				key:   []byte(existingGroup.GetProps().GetId()),
				value: rawExistingGroup,
			},
			existsAfterMigration: true,
		},
		"migrated-group": {
			entry: bucketEntry{
				key:   []byte(migratedGroup.GetProps().GetId()),
				value: rawMigratedGroup,
			},
			existsAfterMigration: true,
		},
		"invalid-group": {
			entry: bucketEntry{
				key:   []byte("some-random-key"),
				value: rawInvalidGroup,
			},
		},
		"invalid-group-stored-by-id": {
			entry: bucketEntry{
				key:   []byte(groupIDForInvalidGroup),
				value: rawInvalidGroup,
			},
		},
		"invalid-bytes": {
			entry: bucketEntry{
				key:   []byte(groupIDForInvalidBytes),
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

	// 2. Create command with mock values.
	env, stdOut, stdErr := mocks.NewEnvWithConn(nil, s.T())

	cmd := &recreateGroupsCommand{
		env: env,
		db:  s.db,
	}

	// 3. Add the groups to the groups bucket before running the migration.
	err = s.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(groupsBucketName))
		s.NoError(err)
		for _, c := range cases {
			s.NoError(bucket.Put(c.entry.key, c.entry.value))
		}
		return nil
	})
	s.NoError(err)

	// 4. Run the re-creation of the groups bucket.
	s.NoError(cmd.Recreate())

	// 5. Verify that all entries are as expected.
	err = s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(groupsBucketName))

		for _, c := range cases {
			// In case the entry should not exist, it shouldn't be possible to retrieve any values from the given key.
			if !c.existsAfterMigration {
				s.Empty(bucket.Get(c.entry.key))
			} else {
				// In case the entry should exist, it should match the expected value.
				value := bucket.Get(c.entry.key)
				s.NotEmpty(value)
				s.Equal(c.entry.value, value)
			}
		}
		return nil
	})
	s.NoError(err)

	// 6. Verify that the entries count matches.
	var actualEntriesCount int
	err = s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(groupsBucketName))

		return bucket.ForEach(func(k, v []byte) error {
			actualEntriesCount++
			return nil
		})
	})
	s.NoError(err)
	s.Equal(expectedEntriesAfterMigration, actualEntriesCount)

	// 7. Verify that we have printed the invalid entries within stdErr. StdOut should be empty.
	s.Empty(stdOut.String())
	expectedOutput, err := os.ReadFile(path.Join("testdata", "recreate-no-dry-run-stderr.txt"))
	s.Require().NoError(err)
	s.Equal(string(expectedOutput), stdErr.String())
}

func (s *recreateGroupsTestSuite) TestRecreateDryRun() {
	// 1. Prepare the group entries that should be within the database.

	// existingGroup should not be dropped during re-creation.
	existingGroup := &storage.Group{
		Props: &storage.GroupProperties{
			Id:             "io.stackrox.authz.group." + groupUUID,
			AuthProviderId: "some-value",
		},
		RoleName: "some-value",
	}
	rawExistingGroup, err := proto.Marshal(existingGroup)
	s.NoError(err)

	// migratedGroup should not be dropped during re-creation.
	migratedGroup := &storage.Group{
		Props: &storage.GroupProperties{
			Id:             "io.stackrox.authz.group.migrated." + groupUUID,
			AuthProviderId: "some-value",
		},
		RoleName: "some-value",
	}
	rawMigratedGroup, err := proto.Marshal(migratedGroup)
	s.NoError(err)

	// invalidGroup should be dropped during re-creation.
	invalidGroup := &storage.Group{
		Props: &storage.GroupProperties{
			Key: "some-value",
		},
		RoleName: "",
	}
	rawInvalidGroup, err := proto.Marshal(invalidGroup)
	s.NoError(err)

	// The following cases represent the entries within the groups bucket _before_ migration.
	// After migration, note that:
	// - existing-group should not have been dropped, due to having an ID.
	// - migrated-group should not have been dropped, due to having an ID.
	// - invalid-group should have been dropped, due to no ID.
	// - invalid-bytes should have been dropped, due to some weird data and no group proto message.
	cases := map[string]struct {
		entry                bucketEntry
		existsAfterMigration bool
	}{
		"existing-group": {
			entry: bucketEntry{
				key:   []byte(existingGroup.GetProps().GetId()),
				value: rawExistingGroup,
			},
			existsAfterMigration: true,
		},
		"migrated-group": {
			entry: bucketEntry{
				key:   []byte(migratedGroup.GetProps().GetId()),
				value: rawMigratedGroup,
			},
			existsAfterMigration: true,
		},
		"invalid-group": {
			entry: bucketEntry{
				key:   []byte("some-random-key"),
				value: rawInvalidGroup,
			},
			existsAfterMigration: true,
		},
		"invalid-group-stored-by-id": {
			entry: bucketEntry{
				key:   []byte(groupIDForInvalidGroup),
				value: rawInvalidGroup,
			},
			existsAfterMigration: true,
		},
		"invalid-bytes": {
			entry: bucketEntry{
				key:   []byte(groupIDForInvalidBytes),
				value: []byte("some-other-random-bytes"),
			},
			existsAfterMigration: true,
		},
	}
	var expectedEntriesAfterMigration int
	for _, c := range cases {
		if c.existsAfterMigration {
			expectedEntriesAfterMigration++
		}
	}

	// 2. Create command with mock values.
	env, stdOut, stdErr := mocks.NewEnvWithConn(nil, s.T())

	cmd := &recreateGroupsCommand{
		env:    env,
		db:     s.db,
		dryRun: true,
	}

	// 3. Add the groups to the groups bucket before running the migration.
	err = s.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(groupsBucketName))
		s.NoError(err)
		for _, c := range cases {
			s.NoError(bucket.Put(c.entry.key, c.entry.value))
		}
		return nil
	})
	s.NoError(err)

	// 4. Run the re-creation of the groups bucket.
	s.NoError(cmd.Recreate())

	// 5. Verify that all entries are as expected.
	err = s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(groupsBucketName))

		for _, c := range cases {
			// In case the entry should not exist, it shouldn't be possible to retrieve any values from the given key.
			if !c.existsAfterMigration {
				s.Empty(bucket.Get(c.entry.key))
			} else {
				// In case the entry should exist, it should match the expected value.
				value := bucket.Get(c.entry.key)
				s.NotEmpty(value)
				s.Equal(c.entry.value, value)
			}
		}
		return nil
	})
	s.NoError(err)

	// 6. Verify that the entries count matches.
	var actualEntriesCount int
	err = s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(groupsBucketName))

		return bucket.ForEach(func(k, v []byte) error {
			actualEntriesCount++
			return nil
		})
	})
	s.NoError(err)
	s.Equal(expectedEntriesAfterMigration, actualEntriesCount)

	// 7. Verify that we have printed the invalid entries within stdErr. StdOut should be empty.
	s.Empty(stdOut.String())
	expectedOutput, err := os.ReadFile(path.Join("testdata", "recreate-dry-run-stderr.txt"))
	s.Require().NoError(err)
	s.Equal(string(expectedOutput), stdErr.String())
}
