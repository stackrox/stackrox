package diff

import (
	"os"

	npguard "github.com/np-guard/netpol-analyzer/pkg/netpol/diff"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/npg"
)

type diffNetpolCommand struct {
	// Properties that are bound to cobra flags.
	stopOnFirstError      bool
	treatWarningsAsErrors bool
	inputFolderPath1      string
	inputFolderPath2      string
	outputFilePath        string
	removeOutputPath      bool
	outputToFile          bool
	outputFormat          string

	// injected or constructed values
	env environment.Environment
}

// Command defines the netpol connectivity diff command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	diffNetpolCmd := &diffNetpolCommand{env: cliEnvironment}
	c := &cobra.Command{
		Use:   "diff",
		Short: "Report connectivity-diff based on two directories containing network policies and YAML manifests with workload resources",
		Long:  `Based on two input folders containing Kubernetes workloads and network policy YAMLs, this command will report all differences in allowed connections between the resources.`,

		Args: cobra.ExactArgs(0),
		RunE: func(c *cobra.Command, args []string) error {
			analyzer, err := diffNetpolCmd.construct()
			if err != nil {
				return err
			}
			if err := diffNetpolCmd.validate(); err != nil {
				return err
			}
			warns, errs := diffNetpolCmd.analyzeConnectivityDiff(analyzer)
			err = npg.SummarizeErrors(warns, errs, diffNetpolCmd.treatWarningsAsErrors, diffNetpolCmd.env.Logger())
			if err != nil {
				return errors.Wrap(err, "analyzing connectivity diff")
			}
			return nil
		},
	}

	c.Flags().StringVarP(&diffNetpolCmd.inputFolderPath1, "dir1", "", "", "First dir path of input resources (required)")
	c.Flags().StringVarP(&diffNetpolCmd.inputFolderPath2, "dir2", "", "", "Second dir path of input resources to be compared with the first dir path (required)")
	c.Flags().BoolVar(&diffNetpolCmd.treatWarningsAsErrors, "strict", false, "Treat warnings as errors")
	c.Flags().BoolVar(&diffNetpolCmd.stopOnFirstError, "fail", false, "Fail on the first encountered error")
	c.Flags().BoolVar(&diffNetpolCmd.removeOutputPath, "remove", false, "Remove the output path if it already exists")
	c.Flags().BoolVar(&diffNetpolCmd.outputToFile, "save-to-file", false, "Whether to save connections diff output into default file")
	c.Flags().StringVarP(&diffNetpolCmd.outputFilePath, "output-file", "f", "", "Save connections diff output into specific file")
	c.Flags().StringVarP(&diffNetpolCmd.outputFormat, "output-format", "o", defaultOutputFormat, "Configure the connections diff in specific format, supported formats: txt|md|csv|dot")
	return c
}

func (cmd *diffNetpolCommand) construct() (diffAnalyzer, error) {
	var opts []npguard.DiffAnalyzerOption
	if cmd.env != nil && cmd.env.Logger() != nil {
		opts = append(opts, npguard.WithLogger(npg.NewLogger(cmd.env.Logger())))
	}
	if cmd.stopOnFirstError {
		opts = append(opts, npguard.WithStopOnError())
	}
	if cmd.outputFormat != "" {
		opts = append(opts, npguard.WithOutputFormat(cmd.outputFormat))
	}
	opts = append(opts, npguard.WithArgNames("dir1", "dir2"))
	if cmd.outputFilePath != "" {
		cmd.outputToFile = true
	}
	return npguard.NewDiffAnalyzer(opts...), nil
}

func (cmd *diffNetpolCommand) validate() error {
	if cmd.inputFolderPath1 == "" {
		return errox.InvalidArgs.Newf("--dir1 is required")
	}
	if cmd.inputFolderPath2 == "" {
		return errox.InvalidArgs.Newf("--dir2 is required")
	}
	if err := cmd.setupPath(cmd.outputFilePath); err != nil {
		return errors.Wrap(err, "failed to set up file path")
	}
	return nil
}

func (cmd *diffNetpolCommand) setupPath(path string) error {
	if _, err := os.Stat(path); err == nil && !cmd.removeOutputPath {
		return errox.AlreadyExists.Newf("path %q already exists. Use --remove to overwrite or select a different path.", path)
	} else if !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to check whether path %q exists", path)
	}
	return nil
}
