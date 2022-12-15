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
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/util"
	bolt "go.etcd.io/bbolt"
)

const (
	groupsBucket = "groups2"
)

var (
	bucketNameToProto = map[string]proto.Message{
		"groups2": &storage.Group{},
	}
)

type listCommand struct {
	path   string
	bucket string

	db *bolt.DB

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
	cmd.Flags().StringP("bucket", "b", groupsBucket, "Bucket name to list")
	cmd.Flags().StringP("file", "f", "", "Path to the Bolt DB file")

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

	bucket, err := cmd.Flags().GetString("bucket")
	if err != nil {
		return errors.Wrap(err, "retrieving value of bucket flag")
	}
	l.bucket = bucket

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
	err := l.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(l.bucket))
		if bucket == nil {
			return errox.NotFound.Newf("bucket %q does not exist", l.bucket)
		}
		err := bucket.ForEach(func(k, v []byte) error {
			l.env.Logger().PrintfLn("Key %s", k)

			printStringValue(v, l.env.Logger())
			printProtoMessage(k, v, l.bucket, l.env.Logger())
			printHexValue(v, l.env.Logger())
			return nil
		})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "listing entries within bucket %q", l.bucket)
	}
	return nil
}

func printStringValue(value []byte, log logger.Logger) {
	log.PrintfLn(">>>\tstring value:\n%s", value)
}

func printProtoMessage(key, value []byte, bucketName string, log logger.Logger) {
	obj, exist := bucketNameToProto[bucketName]
	if !exist {
		return
	}
	if err := proto.Unmarshal(value, obj); err != nil {
		log.ErrfLn("Could not unmarshal the value for key %s to a proto message: %v",
			key, err)
		return
	}
	log.PrintfLn(">>>\tproto message:\n%v", proto.MarshalTextString(obj))
}

func printHexValue(value []byte, log logger.Logger) {
	log.PrintfLn(">>>\thex value:\n%s", hex.EncodeToString(value))
}
