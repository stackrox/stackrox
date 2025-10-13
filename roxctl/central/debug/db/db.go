package db

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/central/debug/db/stats"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
)

// Command controls all of the functions being applied to resetting things within the DB
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "db",
		Short: "Commands that control the debugging of the database",
	}
	c.AddCommand(stats.Command(cliEnvironment))
	flags.AddTimeout(c)
	return c
}
