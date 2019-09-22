package userpki

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/central/userpki/create"
	"github.com/stackrox/rox/roxctl/central/userpki/delete"
	"github.com/stackrox/rox/roxctl/central/userpki/list"
)

// Command adds the userpki command
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "userpki",
		Short: "Commands to administer user PKI certificates",
		Long:  "Commands to administer user PKI certificates",
	}
	c.AddCommand(list.Command())
	c.AddCommand(create.Command())
	c.AddCommand(delete.Command())
	return c
}
