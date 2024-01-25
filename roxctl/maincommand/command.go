package maincommand

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/roxctl/central"
	"github.com/stackrox/rox/roxctl/cluster"
	"github.com/stackrox/rox/roxctl/collector"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/completion"
	connectivitymapDeprecated "github.com/stackrox/rox/roxctl/connectivity-map"
	"github.com/stackrox/rox/roxctl/declarativeconfig"
	"github.com/stackrox/rox/roxctl/deployment"
	"github.com/stackrox/rox/roxctl/doc"
	"github.com/stackrox/rox/roxctl/generate"
	"github.com/stackrox/rox/roxctl/helm"
	"github.com/stackrox/rox/roxctl/image"
	"github.com/stackrox/rox/roxctl/logconvert"
	"github.com/stackrox/rox/roxctl/netpol"
	"github.com/stackrox/rox/roxctl/scanner"
	"github.com/stackrox/rox/roxctl/sensor"
)

func versionCommand(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "version",
		Short: "Display the current roxctl version.",
		Args:  cobra.NoArgs,
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
	flags.HideInheritedFlags(c)
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
		connectivitymapDeprecated.Command(cliEnvironment),
		netpol.Command(cliEnvironment),
	)
	if features.RoxctlNetpolGenerate.Enabled() {
		c.AddCommand(generate.Command(cliEnvironment))
	}
	if env.DeclarativeConfiguration.BooleanSetting() {
		c.AddCommand(declarativeconfig.Command(cliEnvironment))
	}
	if !buildinfo.ReleaseBuild {
		c.AddCommand(doc.Command(cliEnvironment))
	}

	return c
}
