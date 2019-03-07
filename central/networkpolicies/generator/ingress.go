package generator

import "github.com/stackrox/rox/generated/storage"

var allowAllIngress = &storage.NetworkPolicyIngressRule{
	From: []*storage.NetworkPolicyPeer{
		{
			IpBlock: &storage.IPBlock{
				Cidr: "0.0.0.0/0",
			},
		},
	},
}

func generateIngressRule(node *node) *storage.NetworkPolicyIngressRule {
	if node.hasInternetIngress() {
		return allowAllIngress
	}

	var peers []*storage.NetworkPolicyPeer

	for srcNode := range node.incoming {
		if srcNode.deployment == nil || isSystemDeployment(srcNode.deployment) {
			continue
		}
		peer := &storage.NetworkPolicyPeer{
			PodSelector: srcNode.deployment.GetLabelSelector(),
		}
		if node.deployment.Namespace != srcNode.deployment.Namespace {
			peer.NamespaceSelector = labelSelectorForNamespace(srcNode.deployment.Namespace)
		}

		peers = append(peers, peer)
	}

	return &storage.NetworkPolicyIngressRule{
		From: peers,
	}
}
