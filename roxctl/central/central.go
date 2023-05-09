package central

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/roxctl/central/backup"
	"github.com/stackrox/rox/roxctl/central/cert"
	"github.com/stackrox/rox/roxctl/central/db"
	"github.com/stackrox/rox/roxctl/central/debug"
	"github.com/stackrox/rox/roxctl/central/generate"
	"github.com/stackrox/rox/roxctl/central/initbundles"
	"github.com/stackrox/rox/roxctl/central/login"
	"github.com/stackrox/rox/roxctl/central/userpki"
	"github.com/stackrox/rox/roxctl/central/whoami"
	"github.com/stackrox/rox/roxctl/common/environment"
)

// Command defines the central command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "central",
		Short: "Commands related to the Central service.",
	}
	c.AddCommand(
		cert.Command(cliEnvironment),
		generate.Command(cliEnvironment),
		db.Command(cliEnvironment),
		backup.Command(cliEnvironment, pointers.Bool(true)),
		debug.Command(cliEnvironment),
		userpki.Command(cliEnvironment),
		whoami.Command(cliEnvironment),
		initbundles.Command(cliEnvironment),
		login.Command(cliEnvironment),
	)
	return c
}
