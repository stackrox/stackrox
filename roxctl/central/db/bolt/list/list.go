package list

import (
	"encoding/hex"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/utils"
	groupsUtils "github.com/stackrox/rox/roxctl/central/db/bolt/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/util"
	bolt "go.etcd.io/bbolt"
)

const (
	groupsBucket = "groups2"
)

type listCommand struct {
	path     string
	bucket   string
	detailed bool

	db  *bolt.DB
	env environment.Environment
}

// Command provides the cobra commands for listing bucket entries.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	listCmd := &listCommand{env: cliEnvironment}

	cmd := &cobra.Command{
		Use: "list",
		RunE: util.RunENoArgs(func(cmd *cobra.Command) error {
			if err := listCmd.Construct(cmd); err != nil {
				return err
			}

			defer utils.IgnoreError(listCmd.db.Close)
			if err := listCmd.List(); err != nil {
				return err
			}
			return nil
		}),
	}
	cmd.Flags().StringP("file", "f", "", "Path to the Bolt DB file")
	cmd.Flags().BoolVar(&listCmd.detailed, "details", false, "Include detailed output for each entry.")

	utils.Must(cmd.MarkFlagRequired("file"))
	return cmd
}

// Construct will initialize the list command struct with all required values by e.g. retrieving them from flags
// or initializing the DB connection.
func (l *listCommand) Construct(cmd *cobra.Command) error {
	path, err := cmd.Flags().GetString("file")
	if err != nil {
		return errors.Wrap(err, "retrieving value of file flag")
	}
	l.path = path

	db, err := bolthelper.New(l.path)
	if err != nil {
		return errors.Wrap(err, "connecting to DB")
	}
	l.db = db
	return nil
}

// List will list all key value pairs from a specific bucket in different formats (simple string value,
// proto messages, hex values).
func (l *listCommand) List() error {
	var (
		numValidEntries   int
		numInvalidEntries int
	)

	err := l.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(groupsBucket))
		if bucket == nil {
			return errox.NotFound.Newf("bucket %q does not exist", groupsBucket)
		}
		err := bucket.ForEach(func(k, v []byte) error {
			valid, errCode := groupsUtils.ValidGroupKeyValuePair(k, v)

			if valid {
				numValidEntries++
			} else {
				numInvalidEntries++
			}

			if l.detailed {
				printKeyValue(l.env.Logger(), k, v, errCode)
			}
			return nil
		})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "listing entries within bucket %q", groupsBucket)
	}

	l.env.Logger().PrintfLn("Found %d entries.\nNumber of valid entries: %d\nNumber of invalid entries:%d",
		numValidEntries+numInvalidEntries, numValidEntries, numInvalidEntries)
	return nil
}

func printKeyValue(log logger.Logger, k, v []byte, errCode groupsUtils.ValidationErrorCode) {
	log.PrintfLn("Key %s", k)
	if errCode != groupsUtils.UnsetErrorCode {
		log.ErrfLn("Invalid entry due to %s", errCode)
	}

	log.PrintfLn(">>>\tstring value:\n%s", v)
	var group storage.Group
	if err := proto.Unmarshal(v, &group); err != nil {
		log.ErrfLn("Could not unmarshal the value for key %s to a proto message: %v",
			k, err)
	}
	log.PrintfLn(">>>\tproto message:\n%v", proto.MarshalTextString(&group))

	log.PrintfLn(">>>\thex value:\n%s", hex.EncodeToString(v))
}
