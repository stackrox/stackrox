package bolt

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/central/db/bolt/list"
	"github.com/stackrox/rox/roxctl/central/db/bolt/recreate"
	"github.com/stackrox/rox/roxctl/common/environment"
)

// Command holds all bolt related commands.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	cmd := &cobra.Command{
		Use: "bolt",
	}
	cmd.AddCommand(list.Command(cliEnvironment), recreate.Command(cliEnvironment))
	return cmd
}
