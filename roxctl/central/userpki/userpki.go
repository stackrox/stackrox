package userpki

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/central/userpki/create"
	"github.com/stackrox/rox/roxctl/central/userpki/delete"
	"github.com/stackrox/rox/roxctl/central/userpki/list"
	"github.com/stackrox/rox/roxctl/common"
)

// Command adds the userpki command
func Command(cliEnvironment common.Environment) *cobra.Command {
	c := &cobra.Command{
		Use: "userpki",
	}
	c.AddCommand(list.Command(cliEnvironment))
	c.AddCommand(create.Command(cliEnvironment))
	c.AddCommand(delete.Command(cliEnvironment))
	return c
}
