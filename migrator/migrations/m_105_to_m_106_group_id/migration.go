package m105tom106

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/uuid"
	bolt "go.etcd.io/bbolt"
)

// groupStoredByCompositeKey is a helper struct which contains the group as well as the composite key.
type groupStoredByCompositeKey struct {
	grp          *storage.Group
	compositeKey []byte
}

var (
	bucketName = []byte("groups2")

	migration = types.Migration{
		StartingSeqNum: 105,
		VersionAfter:   &storage.Version{SeqNum: 106},
		Run: func(databases *types.Databases) error {
			return migrateGroupsStoredByCompositeKey(databases.BoltDB)
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func migrateGroupsStoredByCompositeKey(db *bolt.DB) error {
	groupsWithoutID, err := fetchGroupsToMigrate(db)
	if err != nil {
		return errors.Wrap(err, "error fetching groups to migrate")
	}

	if err := addIDsToGroups(db, groupsWithoutID); err != nil {
		return errors.Wrap(err, "error adding IDs to group and storing them")
	}

	if err := removeGroupsStoredByCompositeKey(db, groupsWithoutID); err != nil {
		return errors.Wrap(err, "error removing groups without ID")
	}

	return nil
}

func fetchGroupsToMigrate(db *bolt.DB) (groupsStoredByCompositeKey []groupStoredByCompositeKey, err error) {
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		// Pre-req: Migrating a non-existent bucket should not fail.
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			// 1. Try to unmarshal the stored value to the group proto. If it can be successfully unmarshalled, then
			// 	  it is stored using the ID as key instead of the serialized key.
			if err = proto.Unmarshal(v, &storage.Group{}); err == nil {
				return nil
			}

			// 2. We found a group that is stored using the composite key as index. Deserialize it to a storage.Group.
			grp, err := deserialize(k, v)
			if err != nil {
				return err
			}

			groupsStoredByCompositeKey = append(groupsStoredByCompositeKey, groupStoredByCompositeKey{grp: grp,
				compositeKey: sliceutils.ShallowClone(k)})

			return nil
		})
	})
	return groupsStoredByCompositeKey, err
}

func addIDsToGroups(db *bolt.DB, groupsStoredByCompositeKey []groupStoredByCompositeKey) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		// Pre-req: Migrating a non-existent bucket should not fail.
		if bucket == nil {
			return nil
		}

		for i := range groupsStoredByCompositeKey {
			grp := groupsStoredByCompositeKey[i].grp

			// 1. Generate the group ID if the group does not already have an ID associated with it.
			if grp.GetProps().GetId() == "" {
				grp.GetProps().Id = generateGroupID()
			}

			// 2. Marshal the group proto.
			groupData, err := proto.Marshal(grp)
			if err != nil {
				return err
			}

			// 3. Save the group using the generated / pre-existing ID as key.
			if err := bucket.Put([]byte(grp.GetProps().GetId()), groupData); err != nil {
				return err
			}
		}

		return nil
	})
}

func removeGroupsStoredByCompositeKey(db *bolt.DB, groupStoredByCompositeKeys []groupStoredByCompositeKey) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		// Pre-req: Migrating a non-existent bucket should not fail.
		if bucket == nil {
			return nil
		}

		for i := range groupStoredByCompositeKeys {
			compositeKey := groupStoredByCompositeKeys[i].compositeKey

			// 1. Remove the value stored behind the composite key, since the migrated group is now successfully stored.
			if err := bucket.Delete(compositeKey); err != nil {
				return err
			}
		}

		return nil
	})
}

func generateGroupID() string {
	return "io.stackrox.authz.group.migrated." + uuid.NewV4().String()
}
