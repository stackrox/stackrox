package flatten

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/util"
	"github.com/stackrox/rox/roxctl/db/common"

	"github.com/dgraph-io/badger"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/badgerhelper"
)

// Command defines the central command tree
func Command() *cobra.Command {
	var path string
	var workers int
	c := &cobra.Command{
		Use:   "flatten",
		Short: "Flatten the LSM tree of the BadgerDB to a single level",
		Long:  "Flatten the LSM tree of the BadgerDB to a single level",
		RunE: util.RunENoArgs(func(*cobra.Command) error {
			return flatten(path, workers)
		}),
	}
	c.Flags().StringVar(&path, "path", "/var/lib/stackrox/badgerdb", "Specify this path if you want to point explicitly at a specific BadgerDB")
	c.Flags().IntVar(&workers, "workers", 2, "Specify the number of workers to use")
	return c
}

func flatten(path string, workers int) error {
	opts := badgerhelper.GetDefaultOptions(path).WithNumCompactors(0)
	opts = common.AddTableLoadingModeToOptions(opts)

	badgerDB, err := badger.Open(opts)
	if err != nil {
		return errors.Wrap(err, "could not initialize badger")
	}
	dbClose := common.CloseOnce(badgerDB.Close)
	defer utils.IgnoreError(dbClose)

	if err := badgerDB.Flatten(workers); err != nil {
		return errors.Wrap(err, "error flattening LSM tree")
	}
	return dbClose()
}
