package generate

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

// NetpolGenerateOptions represents input parameters to NetpolGenerateCmd
type NetpolGenerateOptions struct {
	StopOnFirstError      bool
	TreatWarningsAsErrors bool
	OutputFolderPath      string
	OutputFilePath        string
	RemoveOutputPath      bool
}

// NetpolGenerateCmd represents NP-Guard functionality for generating network policies
type NetpolGenerateCmd struct {
	Options NetpolGenerateOptions
	// Properties that are bound to cobra flags.
	offline         bool
	inputFolderPath string
	mergeMode       bool
	splitMode       bool

	// injected or constructed values
	env     environment.Environment
	printer printer.ObjectPrinter
}

// NewNetpolGenerateCmd returns new NewNetpolGenerateCmd object
func NewNetpolGenerateCmd(env environment.Environment) *NetpolGenerateCmd {
	return &NetpolGenerateCmd{
		Options: NetpolGenerateOptions{
			StopOnFirstError:      false,
			TreatWarningsAsErrors: false,
			OutputFolderPath:      "",
			OutputFilePath:        "",
			RemoveOutputPath:      false,
		},
		offline:         false,
		inputFolderPath: "",
		mergeMode:       false,
		splitMode:       false,
		env:             env,
		printer:         nil,
	}

}

// AddFlags binds command flags to parameters
func (cmd *NetpolGenerateCmd) AddFlags(c *cobra.Command) *cobra.Command {
	c.Flags().BoolVar(&cmd.Options.TreatWarningsAsErrors, "strict", false, "treat warnings as errors")
	c.Flags().BoolVar(&cmd.Options.StopOnFirstError, "fail", false, "fail on the first encountered error")
	c.Flags().BoolVar(&cmd.Options.RemoveOutputPath, "remove", false, "remove the output path if it already exists")
	c.Flags().StringVarP(&cmd.Options.OutputFolderPath, "output-dir", "d", "", "save generated policies into target folder - one file per policy")
	c.Flags().StringVarP(&cmd.Options.OutputFilePath, "output-file", "f", "", "save and merge generated policies into a single yaml file")
	return c
}

// ShortText provides short command description
func (cmd *NetpolGenerateCmd) ShortText() string {
	return "(Technology Preview) Recommend Network Policies based on deployment information."
}

// LongText provides long command description
func (cmd *NetpolGenerateCmd) LongText() string {
	return `Based on a given folder containing deployment YAMLs, will generate a list of recommended Network Policies. Will write to stdout if no output flags are provided.

** This is a Technology Preview feature **
Technology Preview features are not supported with Red Hat production service level agreements (SLAs) and might not be functionally complete.
Red Hat does not recommend using them in production.
These features provide early access to upcoming product features, enabling customers to test functionality and provide feedback during the development process.
For more information about the support scope of Red Hat Technology Preview features, see https://access.redhat.com/support/offerings/techpreview/`
}

// RunE runs the command
func (cmd *NetpolGenerateCmd) RunE(c *cobra.Command, args []string) error {
	cmd.env.Logger().WarnfLn("This is a Technology Preview feature. Red Hat does not recommend using Technology Preview features in production.")
	synth, err := cmd.construct(args, c)
	if err != nil {
		return err
	}
	if err := cmd.validate(); err != nil {
		return err
	}
	return cmd.generateNetpol(synth)
}

func (cmd *NetpolGenerateCmd) construct(args []string, c *cobra.Command) (netpolGenerator, error) {
	cmd.inputFolderPath = args[0]
	cmd.splitMode = c.Flags().Changed("output-dir")
	cmd.mergeMode = c.Flags().Changed("output-file")

	var opts []npguard.PoliciesSynthesizerOption
	if cmd.env != nil && cmd.env.Logger() != nil {
		opts = append(opts, npguard.WithLogger(npg.NewLogger(cmd.env.Logger())))
	}
	if cmd.Options.StopOnFirstError {
		opts = append(opts, npguard.WithStopOnError())
	}
	return npguard.NewPoliciesSynthesizer(opts...), nil
}

func (cmd *NetpolGenerateCmd) validate() error {
	if cmd.Options.OutputFolderPath != "" && cmd.Options.OutputFilePath != "" {
		return errors.New("Flags [-d|--output-dir, -f|--output-file] cannot be used together")
	}
	if cmd.splitMode {
		if err := cmd.setupPath(cmd.Options.OutputFolderPath); err != nil {
			return errors.Wrap(err, "failed to set up folder path")
		}
	} else if cmd.mergeMode {
		if err := cmd.setupPath(cmd.Options.OutputFilePath); err != nil {
			return errors.Wrap(err, "failed to set up file path")
		}
	}

	return nil
}

func (cmd *NetpolGenerateCmd) setupPath(path string) error {
	if _, err := os.Stat(path); err == nil && !cmd.Options.RemoveOutputPath {
		return errox.AlreadyExists.Newf("path %s already exists. Use --remove to overwrite or select a different path.", path)
	} else if !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to check if path %s exists", path)
	}
	return nil
}
