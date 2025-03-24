package connectivitymap

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/npg"
)

// Cmd represents 'netpol connectivity map' command
type Cmd struct {
	// Properties that are bound to cobra flags.
	stopOnFirstError      bool
	treatWarningsAsErrors bool
	inputFolderPath       string
	outputFilePath        string
	removeOutputPath      bool
	outputToFile          bool
	focusWorkload         string
	outputFormat          string
	exposure              bool
	explain               bool

	// injected or constructed values
	env environment.Environment
}

// Command defines the map command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	cmd := NewCmd(cliEnvironment)
	c := &cobra.Command{
		Use:   "map <folder-path>",
		Short: "Analyze connectivity based on network policies and other resources.",
		Long:  `Based on a given folder containing deployment and network policy YAMLs, will analyze permitted cluster connectivity. Will write to stdout if no output flags are provided.`,

		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return cmd.RunE(c, args)
		},
	}
	return cmd.AddFlags(c)
}

func (cmd *Cmd) validate() error {
	if err := cmd.setupPath(cmd.outputFilePath); err != nil {
		return errors.Wrap(err, "failed to set up file path")
	}
	return nil
}

func (cmd *Cmd) setupPath(path string) error {
	if _, err := os.Stat(path); err == nil && !cmd.removeOutputPath {
		return errox.AlreadyExists.Newf("path %s already exists. Use --remove to overwrite or select a different path.", path)
	} else if !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to check if path %s exists", path)
	}
	return nil
}

// RunE executes the command and returns potential errors
func (cmd *Cmd) RunE(_ *cobra.Command, args []string) error {
	analyzer, err := cmd.construct(args)
	if err != nil {
		return err
	}
	if err := cmd.validate(); err != nil {
		return err
	}
	warns, errs := cmd.analyze(analyzer)
	err = npg.SummarizeErrors(warns, errs, cmd.treatWarningsAsErrors, cmd.env.Logger())
	if err != nil {
		return errors.Wrap(err, "building connectivity map")
	}
	return nil
}

// AddFlags is for parsing flags and storing their values
func (cmd *Cmd) AddFlags(c *cobra.Command) *cobra.Command {
	c.Flags().BoolVar(&cmd.treatWarningsAsErrors, "strict", false, "Treat warnings as errors")
	c.Flags().BoolVar(&cmd.stopOnFirstError, "fail", false, "Fail on the first encountered error")
	c.Flags().BoolVar(&cmd.removeOutputPath, "remove", false, "Remove the output path if it already exists")
	c.Flags().BoolVar(&cmd.outputToFile, "save-to-file", false, "Whether to save connections list output into default file")
	c.Flags().StringVarP(&cmd.outputFilePath, "output-file", "f", "", "Save connections list output into specific file")
	c.Flags().StringVarP(&cmd.focusWorkload, "focus-workload", "", "", "Focus on connections of specified workload name in the output")
	c.Flags().StringVarP(&cmd.outputFormat, "output-format", "o", defaultOutputFormat, "Configure the connections list in specific format, supported formats: txt|json|md|dot|csv")
	c.Flags().BoolVar(&cmd.exposure, "exposure", false, "Enhance the analysis of permitted connectivity with exposure analysis")
	c.Flags().BoolVar(&cmd.explain, "explain", false, "Enhance the analysis of permitted connectivity with explanations per denied/allowed connection; supported only for txt output format")
	return c
}
