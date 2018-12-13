package db

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/central/db/backup"
	"github.com/stackrox/rox/roxctl/central/db/restore"
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
	return c
}
