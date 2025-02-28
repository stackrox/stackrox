package main

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/maincommand"
	"github.com/stackrox/rox/roxctl/utils"

	// Make sure devbuild setting is registered.
	_ "github.com/stackrox/rox/pkg/devbuild"
)

func main() {
	c := maincommand.Command()

	c.SetHelpFunc(utils.FormatHelp)

	once := sync.Once{}
	// Peak only the deepest command path. The hooks are added to all commands.
	common.PatchPersistentPreRunHooks(c, func(cmd *cobra.Command, args []string) {
		once.Do(func() {
			common.RoxctlCommand = getCommandPath(cmd)
		})
	})

	clientconn.SetUserAgent(clientconn.Roxctl)

	if err := c.Execute(); err != nil {
		os.Exit(1)
	}
}

func getCommandPath(cmd *cobra.Command) string {
	binaryName := cmd.Root().CommandPath() + " "
	return strings.TrimPrefix(cmd.CommandPath(), binaryName)
}
