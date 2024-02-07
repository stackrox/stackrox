package netpol

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/netpol/connectivity"
	"github.com/stackrox/rox/roxctl/netpol/generate"
)

// Command defines the netpol command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "netpol",
		Short: "(Technology Preview) Commands related to network policies.",
		Long:  `Commands related to to network policies.` + common.TechPreviewLongText,
	}

	c.AddCommand(
		connectivity.Command(cliEnvironment),
		generate.Command(cliEnvironment),
	)
	return c
}
