package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/roxctl/central"
	"github.com/stackrox/rox/roxctl/cluster"
	"github.com/stackrox/rox/roxctl/collector"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
	"github.com/stackrox/rox/roxctl/db"
	"github.com/stackrox/rox/roxctl/deployment"
	"github.com/stackrox/rox/roxctl/gcp"
	"github.com/stackrox/rox/roxctl/image"
	"github.com/stackrox/rox/roxctl/scanner"
	"github.com/stackrox/rox/roxctl/sensor"

	// Make sure devbuild setting is registered.
	_ "github.com/stackrox/rox/pkg/devbuild"
)

func versionCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "version",
		Short: "Print the roxctl version number",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			if useJSON, _ := c.Flags().GetBool("json"); useJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(version.GetAllVersions())
			}
			fmt.Println(version.GetMainVersion())
			return nil
		}),
	}
	c.PersistentFlags().Bool("json", false, "print extended version information as JSON")
	return c
}

func main() {
	c := &cobra.Command{
		SilenceUsage: true,
		Use:          os.Args[0],
	}

	c.AddCommand(
		central.Command(),
		cluster.Command(),
		collector.Command(),
		deployment.Command(),
		gcp.Command(),
		image.Command(),
		scanner.Command(),
		sensor.Command(),
		db.Command(),
		versionCommand(),
	)

	flags.AddPassword(c)
	flags.AddConnectionFlags(c)

	if err := c.Execute(); err != nil {
		os.Exit(1)
	}
}
