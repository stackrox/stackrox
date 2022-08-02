package generate

import (
	"github.com/np-guard/cluster-topology-analyzer/pkg/controller"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/protoconv/networkpolicy"
)

func (cmd *netpolGenerateCommand) generateNetpol() error {
	recommendedNetpols, err := controller.PoliciesFromFolderPath(cmd.folderPath)
	if err != nil {
		return errors.Errorf("Error synthesizing policies from folder: %v", err)
	}

	for _, netpol := range recommendedNetpols {
		yamlPolicy, err := networkpolicy.KubernetesNetworkPolicyWrap{NetworkPolicy: netpol}.ToYaml()
		if err != nil {
			return errors.Errorf("Error converting YAML into Network Policies: %v", err)
		}
		cmd.env.Logger().PrintfLn("---\n\n%s", yamlPolicy)
	}
	return nil
}
