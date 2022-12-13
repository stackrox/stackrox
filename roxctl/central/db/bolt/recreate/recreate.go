package recreate

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/central/db/bolt/recreate/groups"
	"github.com/stackrox/rox/roxctl/common/environment"
)

// Command provides the cobra command for the re-creation of buckets.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	cmd := &cobra.Command{
		Use: "re-create",
	}

	cmd.AddCommand(groups.Command(cliEnvironment))

	return cmd
}
