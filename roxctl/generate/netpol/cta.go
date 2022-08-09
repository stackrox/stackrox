package netpol

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/np-guard/cluster-topology-analyzer/pkg/controller"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	v1 "k8s.io/api/networking/v1"
)

func (cmd *generateNetpolCommand) generateNetpol() error {
	recommendedNetpols, err := controller.PoliciesFromFolderPath(cmd.folderPath)
	if err != nil {
		return errors.Wrap(err, "error synthesizing policies from folder")
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
			return errors.Wrapf(err, "error saving merged Network Policies")
		}
		return nil
	}

	if cmd.splitMode {
		if err := cmd.saveNetpolsToFolder(recommendedNetpols); err != nil {
			return errors.Wrapf(err, "error saving split Network Policies")
		}
		return nil
	}

	if cmd.stdoutMode {
		cmd.printNetpols(mergedPolicy)
		return nil
	}

	cmd.env.Logger().WarnfLn("No mode selected, not providing any output.")
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
		return errors.Wrapf(err, "error writing merged Network Policies")
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
			return errors.Wrapf(err, "error writing policy to file")
		}
	}
	return nil
}

func writeFile(filename string, destDir string, content string) error {
	outputPath := filepath.Join(destDir, filename)
	if err := os.MkdirAll(filepath.Dir(destDir), 0755); err != nil {
		return errors.Wrapf(err, "error creating directory for file %q", filename)
	}

	return errors.Wrap(os.WriteFile(outputPath, []byte(content), os.FileMode(0644)), "error writing file")
}
