// Package netpol provides primitives for command 'roxctl generate netpol'
package netpol

import (
	"os"
	"path/filepath"
	"strings"

	npguard "github.com/np-guard/cluster-topology-analyzer/pkg/controller"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	v1 "k8s.io/api/networking/v1"
)

var (
	errNPGErrorsIndicator   = errors.New("there were errors during execution")
	errNPGWarningsIndicator = errors.New("there were warnings during execution")
)

type netpolGenerator interface {
	PoliciesFromFolderPath(string) ([]*v1.NetworkPolicy, error)
	Errors() []npguard.FileProcessingError
}

func (cmd *generateNetpolCommand) generateNetpol(synth netpolGenerator) error {
	recommendedNetpols, err := synth.PoliciesFromFolderPath(cmd.folderPath)
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
			roxerr = errNPGErrorsIndicator
		} else {
			cmd.env.Logger().WarnfLn("%s %s", e.Error(), e.Location())
			if cmd.treatWarningsAsErrors && roxerr == nil {
				roxerr = errNPGWarningsIndicator
			}
		}
	}
	return roxerr
}

func (cmd *generateNetpolCommand) ouputNetpols(recommendedNetpols []*v1.NetworkPolicy) error {
	if _, err := os.Stat(cmd.outputFolderPath); err == nil {
		if err := os.RemoveAll(cmd.outputFolderPath); err != nil {
			return errors.Wrapf(err, "failed to remove output path %s", cmd.outputFolderPath)
		}
		cmd.env.Logger().WarnfLn("Removed output path %s", cmd.outputFolderPath)
	}
	if cmd.outputFolderPath != "" {
		cmd.env.Logger().InfofLn("Writing generated Network Policies to %q", cmd.outputFolderPath)
	}

	var mergedPolicy string
	yamlPolicies := make([]string, 0, len(recommendedNetpols))
	for _, netpol := range recommendedNetpols {
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

func (cmd *generateNetpolCommand) printNetpols(combinedNetpols string) {
	cmd.env.Logger().PrintfLn(combinedNetpols)
}

func (cmd *generateNetpolCommand) saveNetpolsToMergedFile(combinedNetpols string) error {
	dirpath, filename := filepath.Split(cmd.outputFilePath)
	if filename == "" {
		filename = "policies.yaml"
	}

	if err := writeFile(filename, dirpath, combinedNetpols); err != nil {
		return errors.Wrap(err, "error writing merged Network Policies")
	}
	return nil
}

func (cmd *generateNetpolCommand) saveNetpolsToFolder(recommendedNetpols []*v1.NetworkPolicy) error {
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

		if err := writeFile(filename, cmd.outputFolderPath, yamlPolicy); err != nil {
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
