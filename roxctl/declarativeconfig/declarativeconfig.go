package declarativeconfig

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/declarativeconfig/create"
)

// Command defines the declarative config command tree.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "declarative-config",
		Short: "Commands related to declarative configuration",
	}

	c.AddCommand(
		create.Command(cliEnvironment),
	)
	return c
}
