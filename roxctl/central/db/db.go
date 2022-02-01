package db

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/central/db/backup"
	"github.com/stackrox/rox/roxctl/central/db/restore"
	"github.com/stackrox/rox/roxctl/common/flags"
)

// Command controls all of the functions being applied to a sensor
func Command() *cobra.Command {
	c := &cobra.Command{
		Use: "db",
	}
	c.AddCommand(backup.Command())
	c.AddCommand(restore.V2Command())
	flags.AddTimeoutWithDefault(c, 1*time.Hour)
	return c
}
