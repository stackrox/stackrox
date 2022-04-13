package maincommand

import (
	"encoding/json"
	"fmt"
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
	"github.com/stackrox/stackrox/roxctl/common/printer"
	"github.com/stackrox/stackrox/roxctl/completion"
	"github.com/stackrox/stackrox/roxctl/deployment"
	"github.com/stackrox/stackrox/roxctl/helm"
	"github.com/stackrox/stackrox/roxctl/image"
	"github.com/stackrox/stackrox/roxctl/logconvert"
	"github.com/stackrox/stackrox/roxctl/scanner"
	"github.com/stackrox/stackrox/roxctl/sensor"
)

func versionCommand() *cobra.Command {
	c := &cobra.Command{
		Use:  "version",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			if useJSON, _ := c.Flags().GetBool("json"); useJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				versions := version.GetAllVersionsDevelopment()
				if buildinfo.ReleaseBuild {
					versions = version.GetAllVersionsUnified()
				}
				return errors.Wrap(enc.Encode(versions), "could not encode version")
			}
			fmt.Println(version.GetMainVersion())
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

	// We have chicken and egg problem here. We need to parse flags to know if --no-color was set
	// but at the same time we need to set printer to handle possible flags parsing errors.
	// Instead of using native cobra flags mechanism we can just check if os.Args contains --no-color.
	var colorPrinter printer.ColorfulPrinter
	if flags.HasNoColor(os.Args) {
		colorPrinter = printer.NoColorPrinter()
	} else {
		colorPrinter = printer.DefaultColorPrinter()
	}
	cliEnvironment := environment.NewCLIEnvironment(environment.DefaultIO(), colorPrinter)
	c.SetErr(errorWriter{
		logger: cliEnvironment.Logger(),
	})

	c.AddCommand(
		central.Command(cliEnvironment),
		cluster.Command(cliEnvironment),
		collector.Command(cliEnvironment),
		deployment.Command(cliEnvironment),
		logconvert.Command(),
		image.Command(cliEnvironment),
		scanner.Command(cliEnvironment),
		sensor.Command(cliEnvironment),
		helm.Command(cliEnvironment),
		versionCommand(),
		completion.Command(),
	)

	return c
}
