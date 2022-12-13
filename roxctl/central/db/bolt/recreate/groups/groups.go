package groups

import (
	"encoding/hex"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/groups"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/util"
	bolt "go.etcd.io/bbolt"
)

const (
	groupsBucketName = "groups2"
	// Value has been taken from:
	//	https://github.com/stackrox/stackrox/blob/6a702b26d66dcc2236a742907809071249187070/central/group/datastore/validate.go#L13
	groupIDPrefix = "io.stackrox.authz.group."
	// Value has been taken from:
	//	https://github.com/stackrox/stackrox/blob/1bd8c26d4918c3b530ad4fd713244d9cf71e786d/migrator/migrations/m_105_to_m_106_group_id/migration.go#L134
	groupMigratedIDPrefix = "io.stackrox.authz.group.migrated."
)

type bucketEntry struct {
	key   []byte
	value []byte
}

type recreateGroupsCommand struct {
	path string
	env  environment.Environment
	db   *bolt.DB
}

// Command provides the cobra command for the re-creation of the groups bucket.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	recreateCmd := &recreateGroupsCommand{env: cliEnvironment}

	cmd := &cobra.Command{
		Use: "groups",
		RunE: util.RunENoArgs(func(cmd *cobra.Command) error {
			if err := recreateCmd.Construct(cmd); err != nil {
				return err
			}

			defer utils.IgnoreError(recreateCmd.db.Close)
			if err := recreateCmd.Recreate(); err != nil {
				return err
			}
			return nil
		}),
	}

	cmd.Flags().StringP("file", "f", "", "Path to the Bolt DB file")
	utils.Must(cmd.MarkFlagRequired("file"))
	return cmd
}

// Construct will initialize the recreate command struct with all required values by e.g. retrieving them from flags
// or initializing the DB connection.
func (r *recreateGroupsCommand) Construct(cmd *cobra.Command) error {
	path, err := cmd.Flags().GetString("file")
	if err != nil {
		return errors.Wrap(err, "retrieving value of file flag")
	}
	r.path = path

	db, err := bolthelper.New(r.path)
	if err != nil {
		return errors.Wrap(err, "connecting to DB")
	}
	r.db = db
	return nil
}

// Recreate will recreate the groups bucket by filtering out invalid entries.
// An invalid entry either:
// - Has a key that does not conform the groups UUID format (io.stackrox.authz.groups.<UUID>).
// - Has a value that cannot be unmarshalled into a groups proto message.
// - Has invalid values within the groups proto message (i.e. fails validation).
func (r *recreateGroupsCommand) Recreate() error {
	// Fetch all group bucket entries.
	entries, err := fetchGroupBucketEntries(r.db)
	if err != nil {
		return errors.Wrapf(err, "fetching entries from bucket %q", groupsBucketName)
	}

	// Drop the groups bucket.
	if err := dropGroupsBucket(r.db); err != nil {
		return errors.Wrapf(err, "dropping bucket %q", groupsBucketName)
	}

	// Recreate bucket from previously existing entries, ensuring only to insert valid entries.
	if err := recreateBucket(r.db, entries, r.env.Logger()); err != nil {
		return errors.Wrapf(err, "recreating bucket %q", groupsBucketName)
	}

	return nil
}

func fetchGroupBucketEntries(db *bolt.DB) ([]bucketEntry, error) {
	var entries []bucketEntry
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(groupsBucketName))
		if bucket == nil {
			return errox.NotFound.Newf("bucket %s does not exist", groupsBucketName)
		}

		err := bucket.ForEach(func(k, v []byte) error {
			entries = append(entries, bucketEntry{
				key:   k,
				value: v,
			})
			return nil
		})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func dropGroupsBucket(db *bolt.DB) error {
	err := db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(groupsBucketName))
	})
	if err != nil {
		return err
	}
	return nil
}

func recreateBucket(db *bolt.DB, entries []bucketEntry, log logger.Logger) error {
	err := db.Update(func(tx *bolt.Tx) error {
		// Purposefully fail here, since we expect the groups bucket to _not_ exist prior, hence we are
		// not using CreateBucketIfNotExists here.
		bucket, err := tx.CreateBucket([]byte(groupsBucketName))
		if err != nil {
			return err
		}

		var upsertGroupErrs *multierror.Error
		for _, entry := range entries {
			if !validBucketEntry(entry) {
				// Since we are unsure _what_ this entry actually is, we are simply going to print the hex value of
				// both key and value, just to be sure.
				log.WarnfLn("An invalid entry within the groups bucket has been found. "+
					"The entry will NOT be included in the re-created bucket.\n"+
					"This is the entry represented encoded as Hex string:\nKey: %s\nValue: %s",
					hex.EncodeToString(entry.key), hex.EncodeToString(entry.value))
				continue
			}

			if err := bucket.Put(entry.key, entry.value); err != nil {
				upsertGroupErrs = multierror.Append(upsertGroupErrs, err)
			}
		}
		return upsertGroupErrs.ErrorOrNil()
	})
	if err != nil {
		return err
	}
	return nil
}

func validBucketEntry(entry bucketEntry) bool {
	key := string(entry.key)

	// Ensure the key has the correct prefix for a group.
	if !strings.HasPrefix(key, groupIDPrefix) && !strings.HasPrefix(key, groupMigratedIDPrefix) {
		return false
	}

	// Ensure the key contains a valid UUID after trimming the prefix.
	// Note that the order is important, as trimming group ID prefix with a migrated ID would leave a .migrated.
	key = strings.TrimPrefix(key, groupMigratedIDPrefix)
	key = strings.TrimPrefix(key, groupIDPrefix)
	_, err := uuid.FromString(key)
	if err != nil {
		return false
	}

	// Ensure that the value can be unmarshalled to a group proto message.
	var group storage.Group
	if err := group.Unmarshal(entry.value); err != nil {
		return false
	}

	// Ensure that the group is a valid group.
	if err := groups.ValidateGroup(&group, true); err != nil {
		return false
	}

	return true
}
