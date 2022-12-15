package groups

import (
	"encoding/hex"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/utils"
	groupsUtils "github.com/stackrox/rox/roxctl/central/db/bolt/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
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
	path   string
	dryRun bool
	env    environment.Environment
	db     *bolt.DB
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
	cmd.Flags().BoolVar(&recreateCmd.dryRun, "dry-run", false, "If dry-run is set, "+
		"the bucket will not be re-created, but only invalid entries will be printed.")

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
//   - Has a key that does not conform the group UUID format
//     io.stackrox.authz.groups.<UUID>/io.stackrox.authz.groups.migrated.<UUID>).
//   - Has a value that cannot be unmarshalled into a groups proto message.
//   - Has invalid values within the groups proto message (i.e. fails validation). Validation is based on the version
//     of central (i.e. the ID has become required with the 3.72.0 release).
func (r *recreateGroupsCommand) Recreate() error {
	// Fetch all group bucket entries.
	entries, err := fetchGroupBucketEntries(r.db)
	if err != nil {
		return errors.Wrapf(err, "fetching entries from bucket %q", groupsBucketName)
	}

	// Drop the groups bucket.
	if err := r.dropGroupsBucket(); err != nil {
		return errors.Wrapf(err, "dropping bucket %q", groupsBucketName)
	}

	// Recreate bucket from previously existing entries, ensuring only to insert valid entries.
	if err := r.recreateBucket(entries); err != nil {
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

func (r *recreateGroupsCommand) dropGroupsBucket() error {
	if r.dryRun {
		return nil
	}
	err := r.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(groupsBucketName))
	})
	if err != nil {
		return err
	}
	return nil
}

func (r *recreateGroupsCommand) recreateBucket(entries []bucketEntry) error {
	err := r.db.Update(func(tx *bolt.Tx) error {

		var bucket *bolt.Bucket
		if r.dryRun {
			bucket = tx.Bucket([]byte(groupsBucketName))
		} else {
			var err error
			// Purposefully fail here, since we expect the groups bucket to _not_ exist prior, hence we are
			// not using CreateBucketIfNotExists here.
			bucket, err = tx.CreateBucket([]byte(groupsBucketName))
			if err != nil {
				return err
			}
		}
		var upsertGroupErrs *multierror.Error
		for _, entry := range entries {
			valid, errCode := groupsUtils.ValidGroupKeyValuePair(entry.key, entry.value)
			if !valid {
				// Since we are unsure _what_ this entry actually is, we are simply going to print the hex value of
				// both key and value, just to be sure.
				// The message will include a reason why the entry was invalid, which will also help us in figuring out
				// what's wrong with the specific entry.
				r.env.Logger().WarnfLn("An invalid entry within the groups bucket has been found (reason: %s). "+
					"The entry will NOT be included in the re-created bucket.\n"+
					"This is the entry represented encoded as Hex string:\nKey: %s\nValue: %s",
					errCode, hex.EncodeToString(entry.key), hex.EncodeToString(entry.value))
				continue
			}

			if r.dryRun {
				r.env.Logger().InfofLn("The following entry would be added to the bucket:\n"+
					"(hex-format)\nKey: %s\nValue: %s", hex.EncodeToString(entry.key), hex.EncodeToString(entry.value))
			} else {
				if err := bucket.Put(entry.key, entry.value); err != nil {
					upsertGroupErrs = multierror.Append(upsertGroupErrs, err)
				}
			}
		}
		return upsertGroupErrs.ErrorOrNil()
	})
	if err != nil {
		return err
	}
	return nil
}
