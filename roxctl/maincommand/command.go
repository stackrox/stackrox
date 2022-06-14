package maincommand

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/stackrox/pkg/buildinfo"
	"github.com/stackrox/stackrox/pkg/version"
	"github.com/stackrox/stackrox/roxctl/central"
	"github.com/stackrox/stackrox/roxctl/cluster"
	"github.com/stackrox/stackrox/roxctl/collector"
	"github.com/stackrox/stackrox/roxctl/common/environment"
	"github.com/stackrox/stackrox/roxctl/common/flags"
	"github.com/stackrox/stackrox/roxctl/completion"
	"github.com/stackrox/stackrox/roxctl/deployment"
	"github.com/stackrox/stackrox/roxctl/helm"
	"github.com/stackrox/stackrox/roxctl/image"
	"github.com/stackrox/stackrox/roxctl/logconvert"
	"github.com/stackrox/stackrox/roxctl/scanner"
	"github.com/stackrox/stackrox/roxctl/sensor"
)

func versionCommand(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:  "version",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			if useJSON, _ := c.Flags().GetBool("json"); useJSON {
				enc := json.NewEncoder(cliEnvironment.InputOutput().Out())
				enc.SetIndent("", "  ")
				versions := version.GetAllVersionsDevelopment()
				if buildinfo.ReleaseBuild {
					versions = version.GetAllVersionsUnified()
				}
				return errors.Wrap(enc.Encode(versions), "could not encode version")
			}
			cliEnvironment.Logger().PrintfLn(version.GetMainVersion())
			return nil
		},
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

	flags.AddNoColor(c)
	flags.AddPassword(c)
	flags.AddConnectionFlags(c)
	flags.AddAPITokenFile(c)

	cliEnvironment := environment.CLIEnvironment()
	c.SetErr(errorWriter{
		logger: cliEnvironment.Logger(),
	})

	c.AddCommand(
		central.Command(cliEnvironment),
		cluster.Command(cliEnvironment),
		collector.Command(cliEnvironment),
		deployment.Command(cliEnvironment),
		logconvert.Command(cliEnvironment),
		image.Command(cliEnvironment),
		scanner.Command(cliEnvironment),
		sensor.Command(cliEnvironment),
		helm.Command(cliEnvironment),
		versionCommand(cliEnvironment),
		completion.Command(cliEnvironment),
	)

	return c
}
