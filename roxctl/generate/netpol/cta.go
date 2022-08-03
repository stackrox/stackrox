package netpol

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/np-guard/cluster-topology-analyzer/pkg/controller"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/protoconv/networkpolicy"
)

func (cmd *generateNetpolCommand) generateNetpol() error {
	recommendedNetpols, err := controller.PoliciesFromFolderPath(cmd.folderPath)
	if err != nil {
		return errors.Wrap(err, "Error synthesizing policies from folder")
	}

	var yamlPolicies []string
	var mergedPolicy string
	for _, netpol := range recommendedNetpols {
		yamlPolicy, err := networkpolicy.KubernetesNetworkPolicyWrap{NetworkPolicy: netpol}.ToYaml()
		if err != nil {
			return errors.Wrap(err, "Error converting Network Policy object to YAML")
		}
		yamlPolicies = append(yamlPolicies, yamlPolicy)
		mergedPolicy = fmt.Sprintf("%s---\n%s", mergedPolicy, yamlPolicy)
	}

	if !cmd.mergePolicies && !cmd.splitPolicies {
		cmd.printNetpolsToStdout(mergedPolicy)
		return nil
	}

	if cmd.mergePolicies {
		if err := cmd.saveNetpolsToMergedFile(mergedPolicy); err != nil {
			return errors.Wrapf(err, "Error writing merged Network Policies to file")
		}
	}

	return nil
}

func (cmd *generateNetpolCommand) printNetpolsToStdout(combinedNetpols string) {
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
		return errors.Wrapf(err, "Error writing policy to file")
	}
	return nil
}

func writeFile(filename string, destDir string, content string) error {
	outputPath := filepath.Join(destDir, filename)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return errors.Wrapf(err, "Error creating directory for file %q", filename)
	}

	perms := os.FileMode(0644)
	return errors.Wrap(os.WriteFile(outputPath, []byte(content), perms), "Error writing file")
}
