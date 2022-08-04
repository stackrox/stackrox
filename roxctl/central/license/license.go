package license

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/central/license/add"
	"github.com/stackrox/rox/roxctl/central/license/info"
	"github.com/stackrox/rox/roxctl/common/flags"
)

// Command controls all of the functions in this subpackage. See usage string below for details.
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:    "license",
		Hidden: true,
	}
	c.AddCommand(add.Command())
	c.AddCommand(info.Command())
	flags.AddTimeout(c)
	return c
}
