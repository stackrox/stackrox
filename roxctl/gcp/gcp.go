package gcp

import (
	"github.com/spf13/cobra"
)

// Command defines the central command tree
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:    "gcp",
		Hidden: true,
	}

	c.AddCommand(Generate())
	return c
}
