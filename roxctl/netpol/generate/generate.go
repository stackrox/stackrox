package generate

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	npguard "github.com/np-guard/cluster-topology-analyzer/v2/pkg/analyzer"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/npg"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stackrox/rox/roxctl/netpol/resources"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/cli-runtime/pkg/resource"
)

const (
	generatedNetworkPolicyLabel = `network-policy-buildtime-generator.stackrox.io/generated`
)

// NetpolGenerateOptions represents input parameters to netpolGenerateCmd
type NetpolGenerateOptions struct {
	StopOnFirstError      bool
	TreatWarningsAsErrors bool
	DNSPort               string
	OutputFolderPath      string
	OutputFilePath        string
	RemoveOutputPath      bool
}

// netpolGenerateCmd represents NP-Guard functionality for generating network policies
type netpolGenerateCmd struct {
	Options NetpolGenerateOptions
	// Properties that are bound to cobra flags.
	offline         bool
	inputFolderPath string
	mergeMode       bool
	splitMode       bool
	dnsPortNum      *int
	dnsPortName     *string

	// injected or constructed values
	env     environment.Environment
	printer printer.ObjectPrinter
}

// AddFlags binds command flags to parameters
func (cmd *netpolGenerateCmd) AddFlags(c *cobra.Command) *cobra.Command {
	c.Flags().BoolVar(&cmd.Options.TreatWarningsAsErrors, "strict", false, "Treat warnings as errors.")
	c.Flags().BoolVar(&cmd.Options.StopOnFirstError, "fail", false, "Fail on the first encountered error.")
	c.Flags().BoolVar(&cmd.Options.RemoveOutputPath, "remove", false, "Remove the output path if it already exists.")
	c.Flags().StringVarP(&cmd.Options.DNSPort, "dnsport", "", "", "Set the DNS port (port number or port name) to be used in egress rules of synthesized NetworkPolicies.")
	c.Flags().StringVarP(&cmd.Options.OutputFolderPath, "output-dir", "d", "", "Save generated policies into target folder - one file per policy.")
	c.Flags().StringVarP(&cmd.Options.OutputFilePath, "output-file", "f", "", "Save and merge generated policies into a single yaml file.")
	return c
}

// RunE runs the command
func (cmd *netpolGenerateCmd) RunE(c *cobra.Command, args []string) error {
	synth, err := cmd.construct(args, c)
	if err != nil {
		return err
	}
	if err := cmd.validate(); err != nil {
		return err
	}
	warns, errs := cmd.generateNetpol(synth)
	err = npg.SummarizeErrors(warns, errs, cmd.Options.TreatWarningsAsErrors, cmd.env.Logger())
	if err != nil {
		return errors.Wrap(err, "generating netpols")
	}
	return nil
}

func (cmd *netpolGenerateCmd) construct(args []string, c *cobra.Command) (*npguard.PoliciesSynthesizer, error) {
	cmd.inputFolderPath = args[0]
	cmd.splitMode = c.Flags().Changed("output-dir")
	cmd.mergeMode = c.Flags().Changed("output-file")
	if c.Flags().Changed("dnsport") {
		dnsPortNum, err := strconv.Atoi(cmd.Options.DNSPort)
		if err == nil {
			cmd.dnsPortNum = &dnsPortNum
		} else {
			cmd.dnsPortName = &cmd.Options.DNSPort
		}
	}

	opts := []npguard.PoliciesSynthesizerOption{}
	if cmd.dnsPortNum != nil {
		opts = append(opts, npguard.WithDNSPort(*cmd.dnsPortNum))
	} else if cmd.dnsPortName != nil {
		opts = append(opts, npguard.WithDNSNamedPort(*cmd.dnsPortName))
	}

	if cmd.env != nil && cmd.env.Logger() != nil {
		opts = append(opts, npguard.WithLogger(npg.NewLogger(cmd.env.Logger())))
	}
	if cmd.Options.StopOnFirstError {
		opts = append(opts, npguard.WithStopOnError())
	}
	return npguard.NewPoliciesSynthesizer(opts...), nil
}

func (cmd *netpolGenerateCmd) validate() error {
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

	if cmd.dnsPortName != nil {
		portErrs := validation.IsValidPortName(*cmd.dnsPortName)
		if len(portErrs) > 0 {
			return errox.InvalidArgs.Newf("illegal port name: %s", portErrs[0])
		}
	}
	if cmd.dnsPortNum != nil {
		portErrs := validation.IsValidPortNum(*cmd.dnsPortNum)
		if len(portErrs) > 0 {
			return errox.InvalidArgs.Newf("illegal port number: %s", portErrs[0])
		}
	}

	return nil
}

func (cmd *netpolGenerateCmd) setupPath(path string) error {
	if _, err := os.Stat(path); err == nil && !cmd.Options.RemoveOutputPath {
		return errox.AlreadyExists.Newf("path %s already exists. Use --remove to overwrite or select a different path.", path)
	} else if !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to check if path %s exists", path)
	}
	return nil
}

type netpolGenerator interface {
	PoliciesFromInfos(infos []*resource.Info) ([]*v1.NetworkPolicy, error)
	ErrorPtrs() []*npguard.FileProcessingError
}

func (cmd *netpolGenerateCmd) generateNetpol(synth netpolGenerator) (w []error, e []error) {
	infos, warns, errs := resources.GetK8sInfos(cmd.inputFolderPath, cmd.Options.StopOnFirstError, cmd.Options.TreatWarningsAsErrors)
	if cmd.Options.StopOnFirstError && (len(errs) > 0 || (len(warns) > 0 && cmd.Options.TreatWarningsAsErrors)) {
		return warns, errs
	}

	recommendedNetpols, err := synth.PoliciesFromInfos(infos)
	if err != nil {
		return warns, append(errs, errors.Wrap(err, "error generating network policies"))
	}
	if err := cmd.ouputNetpols(recommendedNetpols); err != nil {
		return warns, append(errs, err)
	}
	w, e = npg.HandleNPGuardErrors(synth.ErrorPtrs())
	return append(warns, w...), append(errs, e...)
}

func (cmd *netpolGenerateCmd) ouputNetpols(recommendedNetpols []*v1.NetworkPolicy) error {
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

func (cmd *netpolGenerateCmd) printNetpols(combinedNetpols string) {
	cmd.env.Logger().PrintfLn(combinedNetpols)
}

func (cmd *netpolGenerateCmd) saveNetpolsToMergedFile(combinedNetpols string) error {
	dirpath, filename := filepath.Split(cmd.Options.OutputFilePath)
	if filename == "" {
		filename = "policies.yaml"
	}

	if err := writeFile(filename, dirpath, combinedNetpols); err != nil {
		return errors.Wrap(err, "error writing merged Network Policies")
	}
	return nil
}

func (cmd *netpolGenerateCmd) saveNetpolsToFolder(recommendedNetpols []*v1.NetworkPolicy) error {
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
