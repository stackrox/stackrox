package derivelocalvalues

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/helm/internal/common"
)

// Command for deriving local values from existing StackRox Kubernetes resources.
func Command() *cobra.Command {
	var outputDir string
	var output string
	var input string

	c := &cobra.Command{
		Use: fmt.Sprintf("derive-local-values --output <path> <%s>", common.MakePrettyChartNameList(supportedCharts...)),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("incorrect number of arguments, see --help for usage information")
			}
			chartName := args[0]
			if output == "" && outputDir == "" {
				fmt.Fprintln(os.Stderr, `No output file specified using either "--output" or "--output-dir".`)
				fmt.Fprintln(os.Stderr, `If the derived Helm configuration should really be written to stdout, please use "--output=-".`)
				return errors.New("no output file specified")
			}

			if output != "" && outputDir != "" {
				fmt.Fprintln(os.Stderr, `Specify either "--output" or "--output-dir" but not both.`)
				return errors.New("invalid arguments")
			}

			if output == "-" {
				// Internally we represent stdout as empty string.
				output = ""
			}

			outputPath := output
			useDirectory := false
			if outputDir != "" {
				outputPath = outputDir
				useDirectory = true
			}
			return deriveLocalValuesForChart("stackrox", chartName, input, outputPath, useDirectory)

		},
	}
	c.PersistentFlags().StringVar(&output, "output", "", "path to output file")
	c.PersistentFlags().StringVar(&outputDir, "output-dir", "", "path to output directory")
	c.PersistentFlags().StringVar(&input, "input", "", "path to file or directory containing YAML input")

	return c
}
