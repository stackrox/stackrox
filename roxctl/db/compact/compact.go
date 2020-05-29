package compact

import (
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/roxctl/common/util"
	"github.com/stackrox/rox/roxctl/db/common"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/badgerhelper"
)

var (
	log = logging.LoggerForModule()
)

// Command defines the central command tree
func Command() *cobra.Command {
	var (
		path         string
		discardRatio float64
		iterations   int
	)
	c := &cobra.Command{
		Use:   "compact",
		Short: "Compact the database offline",
		Long:  "Compact the database offline",
		RunE: util.RunENoArgs(func(*cobra.Command) error {
			return compact(path, discardRatio, iterations)
		}),
	}
	c.Flags().StringVar(&path, "path", "/var/lib/stackrox/badgerdb", "Specify this path if you want to point explicitly at a specific BadgerDB")
	c.Flags().Float64Var(&discardRatio, "discard-ratio", 0.5, "Specify the required amount of data to be rewritten for GC to rewrite a value log. Lower is more aggressive")
	c.Flags().IntVar(&iterations, "iterations", 20, "Specify the number of iterations of GC to run. At some point, they stop becoming effective if there is no rewrite")
	return c
}

func compact(path string, discardRatio float64, iterations int) error {
	opts := badgerhelper.GetDefaultOptions(path)
	opts = common.AddTableLoadingModeToOptions(opts)

	badgerDB, err := badger.Open(opts)
	if err != nil {
		return errors.Wrap(err, "could not initialize badger")
	}

	var numRewrites int
	for i := 0; i < iterations; i++ {
		err := badgerDB.RunValueLogGC(discardRatio)
		if err != nil && err != badger.ErrNoRewrite {
			log.Errorf("error running GC: %v", err)
		} else {
			if err == nil {
				numRewrites++
			}
			log.Infof("Successfully completed %d/%d iterations\n", i+1, iterations)
		}
	}
	log.Infof("Successfully rewrote the value log %d times\n", numRewrites)
	return badgerDB.Close()
}
