package userpki

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/stackrox/roxctl/central/userpki/create"
	"github.com/stackrox/stackrox/roxctl/central/userpki/delete"
	"github.com/stackrox/stackrox/roxctl/central/userpki/list"
	"github.com/stackrox/stackrox/roxctl/common/environment"
)

// Command adds the userpki command
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use: "userpki",
	}
	c.AddCommand(list.Command(cliEnvironment))
	c.AddCommand(create.Command(cliEnvironment))
	c.AddCommand(delete.Command(cliEnvironment))
	return c
}
