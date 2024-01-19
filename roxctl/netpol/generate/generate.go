package generate

import (
	"os"
	"path/filepath"
	"strings"

	npguard "github.com/np-guard/cluster-topology-analyzer/v2/pkg/analyzer"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/npg"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stackrox/rox/roxctl/netpol/connectivity/netpolerrors"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/cli-runtime/pkg/resource"
)

const (
	generatedNetworkPolicyLabel = `network-policy-buildtime-generator.stackrox.io/generated`
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
	return `Based on a given folder containing deployment YAMLs, will generate a list of recommended Network Policies. Will write to stdout if no output flags are provided.` + common.TechPreviewLongText
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

type netpolGenerator interface {
	PoliciesFromInfos(infos []*resource.Info) ([]*v1.NetworkPolicy, error)
	Errors() []npguard.FileProcessingError
}

func getInfoObj(path string, failFast, treatWarningsAsErrors bool) ([]*resource.Info, error) {
	b := resource.NewLocalBuilder().
		Unstructured().
		FilenameParam(false,
			&resource.FilenameOptions{Filenames: []string{path}, Recursive: true}).
		Flatten()
	// only for the combination of --fail & --strict, should not run with ContinueOnError, and stop on first warning.
	// the only error which is not warning returned from this call is errox.NotFound, for which it already fails fast.
	if !(failFast && treatWarningsAsErrors) {
		b.ContinueOnError()
	}
	//nolint:wrapcheck // we do wrap the errors later in `errHandler.HandleErrors`
	return b.Do().Infos()
}

func (cmd *NetpolGenerateCmd) generateNetpol(synth netpolGenerator) error {
	errHandler := netpolerrors.NewErrHandler(cmd.Options.TreatWarningsAsErrors)
	infos, err := getInfoObj(cmd.inputFolderPath, cmd.Options.StopOnFirstError, cmd.Options.TreatWarningsAsErrors)
	if err := errHandler.HandleError(err); err != nil {
		//nolint:wrapcheck // The package claimed to be external is local and shared by all related netpol-commands
		return err
	}

	recommendedNetpols, err := synth.PoliciesFromInfos(infos)
	if err != nil {
		return errors.Wrap(err, "error generating network policies")
	}
	if err := cmd.ouputNetpols(recommendedNetpols); err != nil {
		return err
	}
	var roxerr error
	for _, e := range synth.Errors() {
		if e.IsSevere() {
			cmd.env.Logger().ErrfLn("%s %s", e.Error(), e.Location())
			roxerr = npg.ErrErrors
		} else {
			cmd.env.Logger().WarnfLn("%s %s", e.Error(), e.Location())
			if cmd.Options.TreatWarningsAsErrors && roxerr == nil {
				roxerr = npg.ErrWarnings
			}
		}
	}
	return roxerr
}

func (cmd *NetpolGenerateCmd) ouputNetpols(recommendedNetpols []*v1.NetworkPolicy) error {
	if _, err := os.Stat(cmd.Options.OutputFolderPath); err == nil {
		if err := os.RemoveAll(cmd.Options.OutputFolderPath); err != nil {
			return errors.Wrapf(err, "failed to remove output path %s", cmd.Options.OutputFolderPath)
		}
		cmd.env.Logger().WarnfLn("Removed output path %s", cmd.Options.OutputFolderPath)
	}
	if cmd.Options.OutputFolderPath != "" {
		cmd.env.Logger().InfofLn("Writing generated Network Policies to %q", cmd.Options.OutputFolderPath)
	}

	var mergedPolicy string
	yamlPolicies := make([]string, 0, len(recommendedNetpols))
	for _, netpol := range recommendedNetpols {
		if netpol.Labels == nil {
			netpol.Labels = make(map[string]string)
		}
		netpol.Labels[generatedNetworkPolicyLabel] = "true"
		yamlPolicy, err := networkpolicy.KubernetesNetworkPolicyWrap{NetworkPolicy: netpol}.ToYaml()
		if err != nil {
			return errors.Wrap(err, "error converting Network Policy object to YAML")
		}
		yamlPolicies = append(yamlPolicies, yamlPolicy)
	}
	mergedPolicy = strings.Join(yamlPolicies, "\n---\n")

	if cmd.mergeMode {
		if err := cmd.saveNetpolsToMergedFile(mergedPolicy); err != nil {
			return errors.Wrap(err, "error saving merged Network Policies")
		}
		return nil
	}

	if cmd.splitMode {
		if err := cmd.saveNetpolsToFolder(recommendedNetpols); err != nil {
			return errors.Wrap(err, "error saving split Network Policies")
		}
		return nil
	}
	cmd.printNetpols(mergedPolicy)
	return nil
}

func (cmd *NetpolGenerateCmd) printNetpols(combinedNetpols string) {
	cmd.env.Logger().PrintfLn(combinedNetpols)
}

func (cmd *NetpolGenerateCmd) saveNetpolsToMergedFile(combinedNetpols string) error {
	dirpath, filename := filepath.Split(cmd.Options.OutputFilePath)
	if filename == "" {
		filename = "policies.yaml"
	}

	if err := writeFile(filename, dirpath, combinedNetpols); err != nil {
		return errors.Wrap(err, "error writing merged Network Policies")
	}
	return nil
}

func (cmd *NetpolGenerateCmd) saveNetpolsToFolder(recommendedNetpols []*v1.NetworkPolicy) error {
	for _, netpol := range recommendedNetpols {
		policyName := netpol.GetName()
		if policyName == "" {
			policyName = string(netpol.GetUID())
		}
		filename := policyName + ".yaml"

		yamlPolicy, err := networkpolicy.KubernetesNetworkPolicyWrap{NetworkPolicy: netpol}.ToYaml()
		if err != nil {
			return errors.Wrap(err, "error converting Network Policy object to yaml")
		}

		if err := writeFile(filename, cmd.Options.OutputFolderPath, yamlPolicy); err != nil {
			return errors.Wrap(err, "error writing policy to file")
		}
	}
	return nil
}

func writeFile(filename string, destDir string, content string) error {
	outputPath := filepath.Join(destDir, filename)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return errors.Wrapf(err, "error creating directory for file %q", filename)
	}

	return errors.Wrap(os.WriteFile(outputPath, []byte(content), os.FileMode(0644)), "error writing file")
}
