package db

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/central/db/backup"
	"github.com/stackrox/rox/roxctl/central/db/generate"
	"github.com/stackrox/rox/roxctl/central/db/restore"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
)

// Command controls all of the functions being applied to a central-db
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "db",
		Short: "Commands that control the database operations",
	}
	c.AddCommand(backup.Command(cliEnvironment))
	c.AddCommand(restore.V2Command(cliEnvironment))
	c.AddCommand(generate.Command(cliEnvironment))
	flags.AddTimeoutWithDefault(c, 1*time.Hour)
	flags.AddRetryTimeoutWithDefault(c, time.Duration(0))
	return c
}
