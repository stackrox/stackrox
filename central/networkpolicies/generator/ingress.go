package generator

import "github.com/stackrox/rox/generated/storage"

var allowAllIngress = &storage.NetworkPolicyIngressRule{
	From: []*storage.NetworkPolicyPeer{},
}

func generateIngressRule(node *node, namespacesByName map[string]*storage.NamespaceMetadata) *storage.NetworkPolicyIngressRule {
	if node.hasInternetIngress() {
		return allowAllIngress
	}

	var peers []*storage.NetworkPolicyPeer

	for srcNode := range node.incoming {
		if srcNode.deployment == nil || isSystemDeployment(srcNode.deployment) {
			continue
		}
		peer := &storage.NetworkPolicyPeer{
			PodSelector: labelSelectorForDeployment(srcNode.deployment),
		}
		if node.deployment.Namespace != srcNode.deployment.Namespace {
			peer.NamespaceSelector = labelSelectorForNamespace(namespacesByName[srcNode.deployment.Namespace])
		}

		peers = append(peers, peer)
	}

	if len(peers) == 0 {
		return nil
	}

	return &storage.NetworkPolicyIngressRule{
		From: peers,
	}
}
