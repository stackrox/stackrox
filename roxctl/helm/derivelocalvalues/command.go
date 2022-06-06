package derivelocalvalues

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	env "github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/helm/internal/common"
)

const (
	defaultNamespace        = "stackrox"
	standardOutput          = ""
	noOutputFileExplanation = `no output file specified using either "--output" or "--output-dir".
If the derived Helm configuration should really be written to stdout, please use "--output=-"`
)

// Command for deriving local values from existing StackRox Kubernetes resources.
func Command(cliEnvironment env.Environment) *cobra.Command {
	helmDeriveLocalValuesCmd := &helmDeriveLocalValuesCommand{env: cliEnvironment}

	c := &cobra.Command{
		Use:  fmt.Sprintf("derive-local-values --output <path> <%s>", common.MakePrettyChartNameList(supportedCharts...)),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			helmDeriveLocalValuesCmd.Construct(args[0])

			if err := helmDeriveLocalValuesCmd.Validate(); err != nil {
				return err
			}

			return deriveLocalValuesForChart(defaultNamespace, helmDeriveLocalValuesCmd.chartName, helmDeriveLocalValuesCmd.input, helmDeriveLocalValuesCmd.outputPath, helmDeriveLocalValuesCmd.useDirectory)

		},
	}
	c.PersistentFlags().StringVar(&helmDeriveLocalValuesCmd.output, "output", "", "path to output file")
	c.PersistentFlags().StringVar(&helmDeriveLocalValuesCmd.outputDir, "output-dir", "", "path to output directory")
	c.PersistentFlags().StringVar(&helmDeriveLocalValuesCmd.input, "input", "", "path to file or directory containing YAML input")

	return c
}

// helmDeriveLocalValuesCommand holds all configurations and metadata to execute a `helm derive-local-values` command
type helmDeriveLocalValuesCommand struct {
	// properties bound to cobra flags
	outputDir string
	output    string
	input     string

	// values injected from either Construct, parent command or for abstracting external dependencies
	chartName    string
	outputPath   string
	useDirectory bool
	env          env.Environment
}

// Construct will enhance the struct with other values coming either from os.Args, other, global flags or environment variables
func (cfg *helmDeriveLocalValuesCommand) Construct(chartName string) {
	cfg.chartName = chartName
}

// Validate will validate the injected values and check whether it's possible to execute the operation with the
// provided values
func (cfg *helmDeriveLocalValuesCommand) Validate() error {
	if cfg.output == "" && cfg.outputDir == "" {
		return errox.InvalidArgs.New(noOutputFileExplanation)
	}

	if cfg.output != "" && cfg.outputDir != "" {
		return errox.InvalidArgs.New(`specify either "--output" or "--output-dir" but not both`)
	}

	if cfg.output == "-" {
		// Internally we represent stdout as empty string.
		cfg.output = standardOutput
	}

	cfg.outputPath = cfg.output
	if cfg.outputDir != "" {
		cfg.outputPath = cfg.outputDir
		cfg.useDirectory = true
	}

	return nil
}
