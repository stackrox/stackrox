package keys

import (
	"fmt"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/util"
	"github.com/stackrox/rox/roxctl/db/common"
)

// Command defines the central command tree
func Command() *cobra.Command {
	var path string
	var prefix string
	c := &cobra.Command{
		Use:   "keys",
		Short: "Keys opens the BadgerDB and allows the user to get key counts",
		Long:  "Keys opens the BadgerDB and allows the user to get key counts",
		RunE: util.RunENoArgs(func(*cobra.Command) error {
			return keys(path, prefix)
		}),
	}
	c.Flags().StringVar(&path, "path", "/var/lib/stackrox/badgerdb", "Specify this path if you want to point explicitly at a specific BadgerDB")
	c.Flags().StringVar(&prefix, "prefix", "", "Specify the prefix of the keys to count")
	return c
}

func keys(path, prefix string) error {
	opts := badgerhelper.GetDefaultOptions(path).WithNumCompactors(0)
	opts = common.AddTableLoadingModeToOptions(opts)

	badgerDB, err := badger.Open(opts)
	if err != nil {
		return errors.Wrap(err, "could not initialize badger")
	}
	dbClose := common.CloseOnce(badgerDB.Close)
	defer utils.IgnoreError(dbClose)

	err = badgerDB.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		prefixBytes := []byte(prefix)
		if prefix != "" {
			opts.Prefix = prefixBytes
		}
		it := tx.NewIterator(opts)
		defer it.Close()
		count := 0
		if prefix != "" {
			for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
				count++
			}
			fmt.Printf("Found %d keys for prefix %q\n", count, prefix)
			return nil
		}
		for it.Rewind(); it.Valid(); it.Next() {
			count++
		}
		fmt.Printf("Found %d keys\n", count)
		return nil
	})
	if err != nil {
		return err
	}

	return dbClose()
}
