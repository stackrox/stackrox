package rewrite

import (
	"os"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/db/common"
)

var (
	log = logging.LoggerForModule()
)

// Command defines the central command tree
func Command() *cobra.Command {
	var path string
	var outputDB string
	c := &cobra.Command{
		Use:   "rewrite",
		Short: "Rewrite opens the BadgerDB and does a logical rewrite to a new DB",
		Long:  "Rewrite opens the BadgerDB and does a logical rewrite to a new DB",
		RunE: func(c *cobra.Command, _ []string) error {
			return rewrite(path, outputDB)
		},
	}
	c.Flags().StringVar(&path, "path", "/var/lib/stackrox/badgerdb", "Specify this path if you want to point explicitly at a specific BadgerDB")
	c.Flags().StringVar(&outputDB, "output", "/var/lib/stackrox/badgerdb-rewrite", "Specify the path to write the new, rewritten BadgerDB out to")
	return c
}

func options(path string) badger.Options {
	opts := badgerhelper.GetDefaultOptions(path).WithNumCompactors(0)
	return common.AddTableLoadingModeToOptions(opts)
}

func rewrite(path, outputDB string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return errors.Wrapf(err, "could not create directory at %q", outputDB)
	}

	oldDB, err := badger.Open(options(path))
	if err != nil {
		return errors.Wrap(err, "could not initialize old badgerDB")
	}
	oldDBClose := common.CloseOnce(oldDB.Close)
	defer utils.IgnoreError(oldDBClose)

	newDB, err := badger.Open(badgerhelper.GetDefaultOptions(outputDB))
	if err != nil {
		return errors.Wrap(err, "could not initialize new badgerDB")
	}
	newDBClose := common.CloseOnce(newDB.Close)
	defer utils.IgnoreError(newDBClose)

	newWriteBatch := newDB.NewWriteBatch()
	defer newWriteBatch.Cancel()

	var keysWritten int
	err = oldDB.View(func(tx *badger.Txn) error {
		it := tx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			key := it.Item().KeyCopy(nil)
			value, err := it.Item().ValueCopy(nil)
			if err != nil {
				return errors.Wrapf(err, "error copying value for key %q", string(key))
			}
			if len(value) == 0 {
				continue
			}
			if err := newWriteBatch.Set(key, value); err != nil {
				return errors.Wrap(err, "error batch writing")
			}
			keysWritten++
			if keysWritten%100000 == 0 {
				log.Infof("Transferred %d keys to new DB", keysWritten)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	if err := newWriteBatch.Flush(); err != nil {
		return errors.Wrap(err, "error flushing batch")
	}

	if err := oldDBClose(); err != nil {
		return errors.Wrap(err, "error closing old DB")
	}

	return errors.Wrap(newDBClose(), "error closing new DB")
}
