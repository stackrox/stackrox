package stats

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/central/debug/db/stats/reset"
	"github.com/stackrox/rox/roxctl/common/environment"
)

// Command controls all of the functions being applied to resetting things within the DB
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "stats",
		Short: "Commands that control the stats of the database",
	}
	c.AddCommand(reset.Command(cliEnvironment))
	return c
}
