package generate

import (
	"github.com/np-guard/cluster-topology-analyzer/pkg/controller"
	"github.com/stackrox/rox/pkg/protoconv/networkpolicy"
)

func (cmd *netpolGenerateCommand) generateNetpol() error {
	recommendedNetpols, err := controller.PoliciesFromFolderPath(cmd.folderPath)
	if err != nil {
		return err
	}

	for _, netpol := range recommendedNetpols {
		yamlPolicy, err := networkpolicy.KubernetesNetworkPolicyWrap{NetworkPolicy: netpol}.ToYaml()
		if err != nil {
			return err
		}
		cmd.env.Logger().PrintfLn("---\n\n%s", yamlPolicy)
	}
	return nil
}
