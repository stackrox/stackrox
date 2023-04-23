package netpol

import (
	"os"

	npguard "github.com/np-guard/netpol-analyzer/pkg/netpol/connlist"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/npg"
)

type analyzeNetpolCommand struct {
	// Properties that are bound to cobra flags.
	stopOnFirstError      bool
	treatWarningsAsErrors bool
	inputFolderPath       string
	outputFilePath        string
	removeOutputPath      bool
	outputToFile          bool
	focusWorkload         string

	// injected or constructed values
	env environment.Environment
}

// Command defines the netpol command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	analyzeNetpolCmd := &analyzeNetpolCommand{env: cliEnvironment}
	c := &cobra.Command{
		Use:   "netpol <folder-path>",
		Short: "(Technology Preview) Analyze Network Policies based resources information.",
		Long: `Based on a given folder containing deployment and netpol YAMLs, will analyze permitted cluster connectivity. Will write to stdout if no output flags are provided.

** This is a Technology Preview feature **
Technology Preview features are not supported with Red Hat production service level agreements (SLAs) and might not be functionally complete.
Red Hat does not recommend using them in production.
These features provide early access to upcoming product features, enabling customers to test functionality and provide feedback during the development process.
For more information about the support scope of Red Hat Technology Preview features, see https://access.redhat.com/support/offerings/techpreview/`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			analyzeNetpolCmd.env.Logger().WarnfLn("This is a Technology Preview feature. Red Hat does not recommend using Technology Preview features in production.")
			analyzer, err := analyzeNetpolCmd.construct(args)
			if err != nil {
				return err
			}
			if err := analyzeNetpolCmd.validate(); err != nil {
				return err
			}
			return analyzeNetpolCmd.analyzeNetpols(analyzer)
		},
	}

	c.Flags().BoolVar(&analyzeNetpolCmd.treatWarningsAsErrors, "strict", false, "treat warnings as errors")
	c.Flags().BoolVar(&analyzeNetpolCmd.stopOnFirstError, "fail", false, "fail on the first encountered error")
	c.Flags().BoolVar(&analyzeNetpolCmd.removeOutputPath, "remove", false, "remove the output path if it already exists")
	c.Flags().BoolVar(&analyzeNetpolCmd.outputToFile, "save-to-file", false, "whether to save connlist output into default file")
	c.Flags().StringVarP(&analyzeNetpolCmd.outputFilePath, "output-file", "f", "", "save connlist output into specific txt file")
	c.Flags().StringVarP(&analyzeNetpolCmd.focusWorkload, "focus-workload", "", "", "focus connections of specified workload name in the output")
	return c
}
func (cmd *analyzeNetpolCommand) construct(args []string) (netpolAnalyzer, error) {
	cmd.inputFolderPath = args[0]
	var opts []npguard.ConnlistAnalyzerOption
	if cmd.env != nil && cmd.env.Logger() != nil {
		opts = append(opts, npguard.WithLogger(npg.NewLogger(cmd.env.Logger())))
	}
	if cmd.stopOnFirstError {
		opts = append(opts, npguard.WithStopOnError())
	}
	if cmd.focusWorkload != "" {
		opts = append(opts, npguard.WithFocusWorkload(cmd.focusWorkload))
	}
	if cmd.outputFilePath != "" {
		cmd.outputToFile = true
	}
	return npguard.NewConnlistAnalyzer(opts...), nil
}

func (cmd *analyzeNetpolCommand) validate() error {
	if err := cmd.setupPath(cmd.outputFilePath); err != nil {
		return errors.Wrap(err, "failed to set up file path")
	}
	return nil
}

func (cmd *analyzeNetpolCommand) setupPath(path string) error {
	if _, err := os.Stat(path); err == nil && !cmd.removeOutputPath {
		return errox.AlreadyExists.Newf("path %s already exists. Use --remove to overwrite or select a different path.", path)
	} else if !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to check if path %s exists", path)
	}
	return nil
}
