package db

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/db/common"
	"github.com/stackrox/rox/roxctl/db/compact"
	"github.com/stackrox/rox/roxctl/db/flatten"
	"github.com/stackrox/rox/roxctl/db/keys"
	"github.com/stackrox/rox/roxctl/db/rewrite"
)

func writeOutProfiles(path string) {
	t := time.NewTicker(30 * time.Second)
	count := 1
	for range t.C {
		filename := fmt.Sprintf("heap%d.pprof", count)
		fullpath := filepath.Join(path, filename)
		file, err := os.Create(fullpath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error creating path for heap profile: %q: %v", fullpath, err)
			continue
		}
		if err := pprof.WriteHeapProfile(file); err != nil {
			fmt.Fprintf(os.Stderr, "error writing heap profile: %v", err)
		}
		utils.IgnoreError(file.Close)
		count++
	}
}

// Command defines the central command tree
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "db",
		Short: "DB commands relate to DB management directly on a DB",
		Long:  "DB commands relate to DB management directly on a DB",
		Run: func(c *cobra.Command, _ []string) {
			_ = c.Help()
		},
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			profilingPath := os.Getenv("PROFILING_PATH")
			if profilingPath == "" {
				return
			}
			go writeOutProfiles(profilingPath)
		},
		Hidden: true, // Explicitly hide this as this is used for support
	}

	c.AddCommand(
		compact.Command(),
		flatten.Command(),
		keys.Command(),
		rewrite.Command(),
	)

	common.AddTableLoadingModeToCommand(c)

	return c
}
