package backup

import (
	"github.com/spf13/cobra"
	centralBackup "github.com/stackrox/rox/roxctl/central/backup"
)

const (
	warningDeprecatedDbBackup = `WARNING: The backup command has been deprecated. Please use "roxctl central backup"
to create central backup with database, keys and certificates.`
)

// Command defines the db backup command. This command is deprecated and can be removed on or after 3.0.57.
func Command() *cobra.Command {
	var full bool
	c := centralBackup.Command(&full)
	c.Deprecated = warningDeprecatedDbBackup
	c.Flags().BoolVarP(&full, "full", "", false, "Create backup with certificates. User admin required.")
	return c
}
