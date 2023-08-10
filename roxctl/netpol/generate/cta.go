// Package generate provides primitives for command 'roxctl generate netpol'
package generate

import (
	"os"
	"path/filepath"
	"strings"

	npguard "github.com/np-guard/cluster-topology-analyzer/pkg/controller"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	"github.com/stackrox/rox/roxctl/common/npg"
	v1 "k8s.io/api/networking/v1"
)

const (
	generatedNetworkPolicyLabel = `network-policy-buildtime-generator.stackrox.io/generated`
)

type netpolGenerator interface {
	PoliciesFromFolderPath(string) ([]*v1.NetworkPolicy, error)
	Errors() []npguard.FileProcessingError
}

func (cmd *NetpolGenerateCmd) generateNetpol(synth netpolGenerator) error {
	recommendedNetpols, err := synth.PoliciesFromFolderPath(cmd.inputFolderPath)
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
