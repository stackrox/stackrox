package connectivitymap

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
)

// Command defines the map command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	cmd := NewCmd(cliEnvironment)
	c := &cobra.Command{
		Use:   "map <folder-path>",
		Short: "(Technology Preview) Analyze connectivity based on network policies and other resources.",
		Long:  `Based on a given folder containing deployment and network policy YAMLs, will analyze permitted cluster connectivity. Will write to stdout if no output flags are provided.` + common.TechPreviewLongText,

		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return errors.Wrap(cmd.RunE(c, args), "building connectivity map")
		},
	}
	return cmd.AddFlags(c)
}
