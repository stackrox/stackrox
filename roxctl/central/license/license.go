package license

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/stackrox/roxctl/central/license/add"
	"github.com/stackrox/stackrox/roxctl/central/license/info"
	"github.com/stackrox/stackrox/roxctl/common/environment"
	"github.com/stackrox/stackrox/roxctl/common/flags"
)

// Command controls all of the functions in this subpackage. See usage string below for details.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:    "license",
		Hidden: true,
	}
	c.AddCommand(add.Command(cliEnvironment))
	c.AddCommand(info.Command(cliEnvironment))
	flags.AddTimeout(c)
	return c
}
