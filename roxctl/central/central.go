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
	"github.com/stackrox/rox/roxctl/central/license"
	"github.com/stackrox/rox/roxctl/central/userpki"
	"github.com/stackrox/rox/roxctl/central/whoami"
	"github.com/stackrox/rox/roxctl/common"
)

// Command defines the central command tree
func Command(cliEnvironment common.Environment) *cobra.Command {
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
