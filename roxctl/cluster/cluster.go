package cluster

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/cluster/delete"
)

// Command controls all of the functions being applied to a sensor
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "cluster",
		Short: "Cluster is the list of commands that pertain to operations on cluster objects",
		Long:  "Cluster is the list of commands that pertain to operations on cluster objects",
	}
	c.AddCommand(delete.Command())
	return c
}
