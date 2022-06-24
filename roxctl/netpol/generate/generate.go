package generate

import (
	"fmt"

	"github.com/np-guard/cluster-topology-analyzer/pkg/controller"
	"github.com/stackrox/rox/pkg/protoconv/networkpolicy"
)

func (cmd *netpolGenerateCommand) generateNetpol() error {
	recommendedNetpols, err := controller.PoliciesFromFolderPath(cmd.folderPath)
	if err != nil {
		return err
	}

	for i, netpol := range recommendedNetpols {
		yamlPolicy, err := networkpolicy.KubernetesNetworkPolicyWrap{NetworkPolicy: netpol}.ToYaml()
		if err != nil {
			return err
		}
		fmt.Printf("Network Policy %d:\n%s\n\n", i, yamlPolicy)
	}
	return nil
}
