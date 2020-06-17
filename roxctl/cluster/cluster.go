package cluster

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/cluster/delete"
	"github.com/stackrox/rox/roxctl/common/flags"
)

// Command controls all of the functions being applied to a sensor
func Command() *cobra.Command {
	c := &cobra.Command{
		Use: "cluster",
	}
	c.AddCommand(delete.Command())
	flags.AddTimeout(c)
	return c
}
