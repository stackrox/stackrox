package netpol

import (
	"os"

	npguard "github.com/np-guard/cluster-topology-analyzer/pkg/controller"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/npg"
	"github.com/stackrox/rox/roxctl/common/printer"
)

type generateNetpolCommand struct {
	// Properties that are bound to cobra flags.
	offline               bool
	stopOnFirstError      bool
	treatWarningsAsErrors bool
	inputFolderPath       string
	outputFolderPath      string
	outputFilePath        string
	removeOutputPath      bool
	mergeMode             bool
	splitMode             bool

	// injected or constructed values
	env     environment.Environment
	printer printer.ObjectPrinter
}

// Command defines the netpol command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	generateNetpolCmd := &generateNetpolCommand{env: cliEnvironment}
	c := &cobra.Command{
		Use:   "netpol <folder-path>",
		Short: "(Technology Preview) Recommend Network Policies based on deployment information.",
		Long: `Based on a given folder containing deployment YAMLs, will generate a list of recommended Network Policies. Will write to stdout if no output flags are provided.

** This is a Technology Preview feature **
Technology Preview features are not supported with Red Hat production service level agreements (SLAs) and might not be functionally complete.
Red Hat does not recommend using them in production.
These features provide early access to upcoming product features, enabling customers to test functionality and provide feedback during the development process.
For more information about the support scope of Red Hat Technology Preview features, see https://access.redhat.com/support/offerings/techpreview/`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			generateNetpolCmd.env.Logger().WarnfLn("This is a Technology Preview feature. Red Hat does not recommend using Technology Preview features in production.")
			synth, err := generateNetpolCmd.construct(args, c)
			if err != nil {
				return err
			}
			if err := generateNetpolCmd.validate(); err != nil {
				return err
			}
			return generateNetpolCmd.generateNetpol(synth)
		},
	}
	c.Flags().BoolVar(&generateNetpolCmd.treatWarningsAsErrors, "strict", false, "treat warnings as errors")
	c.Flags().BoolVar(&generateNetpolCmd.stopOnFirstError, "fail", false, "fail on the first encountered error")
	c.Flags().BoolVar(&generateNetpolCmd.removeOutputPath, "remove", false, "remove the output path if it already exists")
	c.Flags().StringVarP(&generateNetpolCmd.outputFolderPath, "output-dir", "d", "", "save generated policies into target folder - one file per policy")
	c.Flags().StringVarP(&generateNetpolCmd.outputFilePath, "output-file", "f", "", "save and merge generated policies into a single yaml file")
	return c
}

func (cmd *generateNetpolCommand) construct(args []string, c *cobra.Command) (netpolGenerator, error) {
	cmd.inputFolderPath = args[0]
	cmd.splitMode = c.Flags().Changed("output-dir")
	cmd.mergeMode = c.Flags().Changed("output-file")

	var opts []npguard.PoliciesSynthesizerOption
	if cmd.env != nil && cmd.env.Logger() != nil {
		opts = append(opts, npguard.WithLogger(npg.NewLogger(cmd.env.Logger())))
	}
	if cmd.stopOnFirstError {
		opts = append(opts, npguard.WithStopOnError())
	}
	return npguard.NewPoliciesSynthesizer(opts...), nil
}

func (cmd *generateNetpolCommand) validate() error {
	if cmd.outputFolderPath != "" && cmd.outputFilePath != "" {
		return errors.New("Flags [-d|--output-dir, -f|--output-file] cannot be used together")
	}
	if cmd.splitMode {
		if err := cmd.setupPath(cmd.outputFolderPath); err != nil {
			return errors.Wrap(err, "failed to set up folder path")
		}
	} else if cmd.mergeMode {
		if err := cmd.setupPath(cmd.outputFilePath); err != nil {
			return errors.Wrap(err, "failed to set up file path")
		}
	}

	return nil
}

func (cmd *generateNetpolCommand) setupPath(path string) error {
	if _, err := os.Stat(path); err == nil && !cmd.removeOutputPath {
		return errox.AlreadyExists.Newf("path %s already exists. Use --remove to overwrite or select a different path.", path)
	} else if !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to check if path %s exists", path)
	}
	return nil
}
