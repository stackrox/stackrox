package db

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/stackrox/roxctl/central/db/backup"
	"github.com/stackrox/stackrox/roxctl/central/db/restore"
	"github.com/stackrox/stackrox/roxctl/common/environment"
	"github.com/stackrox/stackrox/roxctl/common/flags"
)

// Command controls all of the functions being applied to a sensor
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use: "db",
	}
	c.AddCommand(backup.Command(cliEnvironment))
	c.AddCommand(restore.V2Command(cliEnvironment))
	flags.AddTimeoutWithDefault(c, 1*time.Hour)
	return c
}
