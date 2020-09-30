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
	var output string
	var input string

	c := &cobra.Command{
		Use: fmt.Sprintf("derive-local-values --output <path> <%s>", common.PrettyChartNameList),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("incorrect number of arguments, see --help for usage information")
			}
			chartName := args[0]
			if output == "" {
				fmt.Fprintln(os.Stderr, `No output file specified using "--output".`)
				fmt.Fprintln(os.Stderr, `If the derived Helm configuration should really be written to stdout, please use "--output=-".`)
				return errors.New("no output file specified")
			}
			if output == "-" {
				output = ""
			}
			return deriveLocalValuesForChart("stackrox", chartName, input, output)

		},
	}
	c.PersistentFlags().StringVar(&output, "output", "", "path to output file")
	c.PersistentFlags().StringVar(&input, "input", "", "path to file or directory containing YAML input")

	return c
}
