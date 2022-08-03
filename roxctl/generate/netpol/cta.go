package netpol

import (
	"github.com/np-guard/cluster-topology-analyzer/pkg/controller"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/protoconv/networkpolicy"
)

func (cmd *generateNetpolCommand) generateNetpol() error {
	recommendedNetpols, err := controller.PoliciesFromFolderPath(cmd.folderPath)
	if err != nil {
		return errors.Wrap(err, "Error synthesizing policies from folder")
	}

	for _, netpol := range recommendedNetpols {
		yamlPolicy, err := networkpolicy.KubernetesNetworkPolicyWrap{NetworkPolicy: netpol}.ToYaml()
		if err != nil {
			return errors.Wrap(err, "Error converting YAML into Network Policy")
		}
		cmd.env.Logger().PrintfLn("---\n\n%s", yamlPolicy)
	}
	return nil
}
