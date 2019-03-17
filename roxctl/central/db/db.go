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
		Use:   "db",
		Short: "DB is the list of commands that control DB operations",
		Long:  "DB is the list of commands that control DB operations",
	}
	c.AddCommand(backup.Command())
	c.AddCommand(restore.Command())
	flags.AddTimeoutWithDefault(c, 2*time.Minute)
	return c
}
