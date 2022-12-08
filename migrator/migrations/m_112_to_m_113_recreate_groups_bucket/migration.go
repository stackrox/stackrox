package m112tom113

import (
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/uuid"
	bolt "go.etcd.io/bbolt"
)

type groupEntry struct {
	key   []byte
	value []byte
}

var (
	bucketName = []byte("groups2")

	migration = types.Migration{
		StartingSeqNum: 112,
		VersionAfter:   &storage.Version{SeqNum: 113},
		Run: func(databases *types.Databases) error {
			return recreateGroupsBucket(databases.BoltDB)
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func recreateGroupsBucket(db *bolt.DB) error {
	// Short-circuit if the bucket does not exist.
	exists, err := checkGroupBucketExists(db)
	if err != nil {
		return errors.Wrap(err, "error checking if groups bucket exists")
	}
	if !exists {
		log.WriteToStderr("groups bucket did not exist, hence no re-creation of the groups bucket was done.")
		return nil
	}

	groupEntries, err := fetchGroupsBucket(db)
	if err != nil {
		return errors.Wrap(err, "error fetching groups to recreate")
	}

	// Drop the bucket.
	if err := dropGroupsBucket(db); err != nil {
		return errors.Wrap(err, "error dropping groups bucket")
	}

	// Create groups bucket and filter out invalid entries.
	if err := createGroupsBucket(db, groupEntries); err != nil {
		return errors.Wrap(err, "error recreating groups bucket")
	}

	return nil
}

func fetchGroupsBucket(db *bolt.DB) (groupEntries []groupEntry, err error) {
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		// We previously checked that the bucket should be available, but still add this check here.
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			groupEntries = append(groupEntries, groupEntry{key: k, value: v})
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return groupEntries, nil
}

func dropGroupsBucket(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket(bucketName)
	})
}

func createGroupsBucket(db *bolt.DB, groupEntries []groupEntry) (err error) {
	err = db.Update(func(tx *bolt.Tx) error {
		// Explicitly use the CreateBucket here instead of CreateBucketIfNotExists, as we require the bucket to be
		// previously dropped.
		bucket, err := tx.CreateBucket(bucketName)
		if err != nil {
			return err
		}

		var putGroupErrs errorhelpers.ErrorList
		for _, entry := range groupEntries {
			// After migration 105_to_106, we can assume that the key will be a UUID and the value will be the group
			// proto message.
			// Here, we will check that the key will be a string and can be parsed as a UUID.
			// If that's the case, the entry is valid, and we will add it to the re-created bucket.
			// If not, we will log the invalid entry that will be dropped.
			if !verifyKeyValuePair(entry.key, entry.value) {
				log.WriteToStderrf("Invalid group entry found in groups bucket (key=%s, value=%s). This entry"+
					" will be dropped.",
					entry.key, entry.value)
				continue
			}

			if err := bucket.Put(entry.key, entry.value); err != nil {
				putGroupErrs.AddError(err)
			}
		}

		return putGroupErrs.ToError()
	})
	return err
}

func checkGroupBucketExists(db *bolt.DB) (exists bool, err error) {
	exists = true
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		if bucket == nil {
			exists = false
		}
		return nil
	})
	return exists, err
}

const (
	// Value has been taken from:
	//	https://github.com/stackrox/stackrox/blob/6a702b26d66dcc2236a742907809071249187070/central/group/datastore/validate.go#L13
	groupIDPrefix = "io.stackrox.authz.group."
	// Value has been taken from:
	//	https://github.com/stackrox/stackrox/blob/1bd8c26d4918c3b530ad4fd713244d9cf71e786d/migrator/migrations/m_105_to_m_106_group_id/migration.go#L134
	groupMigratedIDPrefix = "io.stackrox.authz.group.migrated."
)

func verifyKeyValuePair(key, value []byte) bool {
	stringKey := string(key)

	// The key should be a string ID, with a constant prefix and a UUID.
	if !strings.HasPrefix(stringKey, groupIDPrefix) && !strings.HasPrefix(stringKey, groupMigratedIDPrefix) {
		return false
	}
	stringKey = strings.TrimPrefix(stringKey, groupMigratedIDPrefix)
	stringKey = strings.TrimPrefix(stringKey, groupIDPrefix)
	_, err := uuid.FromString(stringKey)
	if err != nil {
		return false
	}

	// The value should be a storage.Group with ID set.
	var group storage.Group
	if err := proto.Unmarshal(value, &group); err != nil {
		return false
	}
	return group.GetProps().GetId() != ""
}
