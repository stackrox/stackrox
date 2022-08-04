package netpol

import (
	"fmt"
	"os"
	"path/filepath"

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
	for _, netpol := range recommendedNetpols {
		yamlPolicy, err := networkpolicy.KubernetesNetworkPolicyWrap{NetworkPolicy: netpol}.ToYaml()
		if err != nil {
			return errors.Wrap(err, "error converting Network Policy object to YAML")
		}
	}
mergedPolicy = strings.Join(yamlPolicies, "\n---\n")
	if !cmd.mergePolicies && !cmd.splitPolicies {
		cmd.printNetpols(mergedPolicy)
		return nil
	}

	if cmd.mergePolicies {
		if err := cmd.saveNetpolsToMergedFile(mergedPolicy); err != nil {
			return errors.Wrapf(err, "error saving merged Network Policies")
		}
	}

	if cmd.splitPolicies {
		if err := cmd.saveNetpolsToFolder(recommendedNetpols); err != nil {
			return errors.Wrapf(err, "error saving split Network Policies")
		}
	}

	return nil
}

func (cmd *generateNetpolCommand) printNetpols(combinedNetpols string) {
	cmd.env.Logger().PrintfLn(combinedNetpols)
}

func (cmd *generateNetpolCommand) saveNetpolsToMergedFile(combinedNetpols string) error {
	dirpath, filename := filepath.Split(cmd.outputFilePath)
	if dirpath == "" {
		dirpath = "./"
	}
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
			return errors.Wrap(err, "error converting Network Policy object to YAML")
		}

		if err := writeFile(filename, cmd.outputFolderPath, yamlPolicy); err != nil {
			return errors.Wrapf(err, "error writing policy to file")
		}
	}
	return nil
}

func writeFile(filename string, destDir string, content string) error {
	outputPath := filepath.Join(destDir, filename)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return errors.Wrapf(err, "error creating directory for file %q", filename)
	}

	perms := os.FileMode(0644)
	return errors.Wrap(os.WriteFile(outputPath, []byte(content), perms), "error writing file")
}
