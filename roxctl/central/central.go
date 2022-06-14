package central

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/stackrox/pkg/pointers"
	"github.com/stackrox/stackrox/roxctl/central/backup"
	"github.com/stackrox/stackrox/roxctl/central/cert"
	"github.com/stackrox/stackrox/roxctl/central/db"
	"github.com/stackrox/stackrox/roxctl/central/debug"
	"github.com/stackrox/stackrox/roxctl/central/generate"
	"github.com/stackrox/stackrox/roxctl/central/initbundles"
	"github.com/stackrox/stackrox/roxctl/central/license"
	"github.com/stackrox/stackrox/roxctl/central/userpki"
	"github.com/stackrox/stackrox/roxctl/central/whoami"
	"github.com/stackrox/stackrox/roxctl/common/environment"
)

// Command defines the central command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use: "central",
	}
	c.AddCommand(
		cert.Command(cliEnvironment),
		generate.Command(cliEnvironment),
		db.Command(cliEnvironment),
		backup.Command(cliEnvironment, pointers.Bool(true)),
		debug.Command(cliEnvironment),
		license.Command(),
		userpki.Command(cliEnvironment),
		whoami.Command(cliEnvironment),
		initbundles.Command(cliEnvironment),
	)
	return c
}
