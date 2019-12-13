package common

import (
	"fmt"
	"os"

	"github.com/dgraph-io/badger"
	"github.com/dgraph-io/badger/options"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	tableLoadingMode string
)

// AddTableLoadingModeToCommand adds the table loading mode flag to the root command and references a local variable
func AddTableLoadingModeToCommand(c *cobra.Command) {
	c.PersistentFlags().StringVar(&tableLoadingMode, "table-loading-mode", "mmap", "set this mode to determine how to load the tables (options are mmap or fileio)")
}

// AddTableLoadingModeToOptions adds the current mode to the badger options
func AddTableLoadingModeToOptions(opts badger.Options) badger.Options {
	var mode options.FileLoadingMode
	switch tableLoadingMode {
	case "mmap":
		mode = options.MemoryMap
	case "fileio":
		mode = options.FileIO
	default:
		fmt.Fprintf(os.Stderr, "Unrecognized file loading mode: %q. Options are mmap or fileio\n", tableLoadingMode)
		os.Exit(1)
	}
	return opts.WithTableLoadingMode(mode)
}

// CloseOnce wraps the close function with a once to ensure
// that it can be deferred, but also can have its error handled
func CloseOnce(close func() error) func() error {
	var err error
	var once sync.Once
	return func() error {
		once.Do(func() {
			err = close()
		})
		return err
	}
}
