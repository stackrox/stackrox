package m111tom112

import (
	"bytes"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

// groupStoredByCompositeKey is a helper struct which contains the group as well as the composite key.
type groupStoredByCompositeKey struct {
	grp          *storage.Group
	compositeKey []byte
}

var (
	bucketName = []byte("groups2")

	emptyPropertiesCompositeKey = serializePropsKey(nil)

	migration = types.Migration{
		StartingSeqNum: 111,
		VersionAfter:   &storage.Version{SeqNum: 112},
		Run: func(databases *types.Databases) error {
			return removeGroupsWithInvalidValues(databases.BoltDB)
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func removeGroupsWithInvalidValues(db *bolt.DB) error {
	groupsWithInvalidValues, err := fetchGroupsToRemove(db)
	if err != nil {
		return errors.Wrap(err, "error fetching groups to remove")
	}

	if err := removeGroupsStoredByCompositeKey(db, groupsWithInvalidValues); err != nil {
		return errors.Wrap(err, "error removing groups with invalid values")
	}

	return nil
}

func fetchGroupsToRemove(db *bolt.DB) (groupsStoredByCompositeKey []groupStoredByCompositeKey, err error) {
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		// Pre-req: Migrating a non-existent bucket should not fail.
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			// In a prior migration, the groups bucket was migrated from groups being stored by a composite key to groups
			// being stored by a UUID.
			// Within that migration, groups that had an empty role name were skipped during migration.
			// This lead to bucket entries where the properties and role name was both empty, thus making it impossible
			// to delete the group now that we require an ID being set.
			// We should check if there are still groups left stored by the composite key due to having an empty role
			// name or empty properties and remove those.
			if len(v) == 0 || bytes.Equal(k, emptyPropertiesCompositeKey) {
				grp, err := deserialize(k, v)
				if err != nil {
					return err
				}

				groupsStoredByCompositeKey = append(groupsStoredByCompositeKey,
					groupStoredByCompositeKey{grp: grp, compositeKey: k})
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return groupsStoredByCompositeKey, nil
}

func removeGroupsStoredByCompositeKey(db *bolt.DB, groupStoredByCompositeKeys []groupStoredByCompositeKey) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		// Pre-req: Migrating a non-existent bucket should not fail.
		if bucket == nil {
			return nil
		}

		var deleteGroupErrs *multierror.Error
		for _, group := range groupStoredByCompositeKeys {
			// Remove the value stored behind the composite key, since the migrated group is now successfully stored.
			if err := bucket.Delete(group.compositeKey); err != nil {
				deleteGroupErrs = multierror.Append(deleteGroupErrs, err)
			}
		}

		return deleteGroupErrs.ErrorOrNil()
	})
}
