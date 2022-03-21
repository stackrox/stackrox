package derivelocalvalues

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/helm/internal/common"
)

const (
	defaultNamespace = "stackrox"
	standardOutput   = ""
)

// Command for deriving local values from existing StackRox Kubernetes resources.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	cmdCfg := &helmDeriveLocalValuesCommand{env: cliEnvironment}

	c := &cobra.Command{
		Use: fmt.Sprintf("derive-local-values --output <path> <%s>", common.MakePrettyChartNameList(supportedCharts...)),
		RunE: func(cmd *cobra.Command, args []string) error {

			if err := cmdCfg.Construct(args, cmd); err != nil {
				return err
			}
			if err := cmdCfg.Validate(); err != nil {
				return err
			}

			return deriveLocalValuesForChart(defaultNamespace, cmdCfg.chartName, cmdCfg.input, cmdCfg.outputPath, cmdCfg.useDirectory)

		},
	}
	c.PersistentFlags().StringVar(&cmdCfg.output, "output", "", "path to output file")
	c.PersistentFlags().StringVar(&cmdCfg.outputDir, "output-dir", "", "path to output directory")
	c.PersistentFlags().StringVar(&cmdCfg.input, "input", "", "path to file or directory containing YAML input")

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
	env          environment.Environment
	logger       environment.Logger
}

// Construct will enhance the struct with other values coming either from os.Args, other, global flags or environment variables
func (cfg *helmDeriveLocalValuesCommand) Construct(args []string, cmd *cobra.Command) error {
	if len(args) != 1 {
		return errors.New("incorrect number of arguments, see --help for usage information")
	}
	cfg.chartName = args[0]

	cfg.logger = cfg.env.Logger()

	return nil
}

// Validate will validate the injected values and check whether it's possible to execute the operation with the
// provided values
func (cfg *helmDeriveLocalValuesCommand) Validate() error {
	if cfg.output == "" && cfg.outputDir == "" {
		cfg.logger.ErrfLn(`No output file specified using either "--output" or "--output-dir".`)
		cfg.logger.ErrfLn(`If the derived Helm configuration should really be written to stdout, please use "--output=-".`)
		return errox.NewErrInvalidArgs("no output file specified")
	}

	if cfg.output != "" && cfg.outputDir != "" {
		cfg.logger.ErrfLn(`Specify either "--output" or "--output-dir" but not both.`)
		return errox.NewErrInvalidArgs(`invalid arguments "--output" and "--output-dir"`)
	}

	if cfg.output == "-" {
		cfg.output = standardOutput
	}

	cfg.outputPath = cfg.output
	if cfg.outputDir != "" {
		cfg.outputPath = cfg.outputDir
		cfg.useDirectory = true
	}

	return nil
}
