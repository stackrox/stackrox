package license

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/central/license/add"
	"github.com/stackrox/rox/roxctl/common/flags"
)

const (
	description = "License controls operations related to licenses for the StackRox security platform"
)

// Command controls all of the functions in this subpackage. See usage string below for details.
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "license",
		Short: description,
		Long:  description,
	}
	c.AddCommand(add.Command())
	flags.AddTimeout(c)
	return c
}
