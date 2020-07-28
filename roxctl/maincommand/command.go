package maincommand

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/roxctl/central"
	"github.com/stackrox/rox/roxctl/cluster"
	"github.com/stackrox/rox/roxctl/collector"
	"github.com/stackrox/rox/roxctl/common/util"
	"github.com/stackrox/rox/roxctl/db"
	"github.com/stackrox/rox/roxctl/deployment"
	"github.com/stackrox/rox/roxctl/gcp"
	"github.com/stackrox/rox/roxctl/image"
	"github.com/stackrox/rox/roxctl/logconvert"
	"github.com/stackrox/rox/roxctl/scanner"
	"github.com/stackrox/rox/roxctl/sensor"
)

func versionCommand() *cobra.Command {
	c := &cobra.Command{
		Use: "version",
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
	c.PersistentFlags().Bool("json", false, "display extended version information as JSON")
	return c
}

// Command constructs and returns the roxctl command tree
func Command() *cobra.Command {
	c := &cobra.Command{
		SilenceUsage: true,
		Use:          os.Args[0],
	}
	c.AddCommand(
		central.Command(),
		cluster.Command(),
		collector.Command(),
		deployment.Command(),
		logconvert.Command(),
		gcp.Command(),
		image.Command(),
		scanner.Command(),
		sensor.Command(),
		db.Command(),
		versionCommand(),
	)
	return c
}
