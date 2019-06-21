package generator

import (
	"github.com/stackrox/rox/generated/storage"
)

var allowAllIngress = &storage.NetworkPolicyIngressRule{
	From: []*storage.NetworkPolicyPeer{},
}

var allowAllPodsAllNS = &storage.NetworkPolicyIngressRule{
	From: []*storage.NetworkPolicyPeer{
		{
			NamespaceSelector: &storage.LabelSelector{},
			PodSelector:       &storage.LabelSelector{},
		},
	},
}

func generateIngressRule(node *node, namespacesByName map[string]*storage.NamespaceMetadata) *storage.NetworkPolicyIngressRule {
	if node.hasInternetIngress() {
		return allowAllIngress
	}

	// If any peer deployment is not visible, generate 'allow all pods in all namespaces' selector
	if node.hasMaskedPeer() {
		log.Infof("insufficient permissions to peer deployment(s) of node %s; generating allow all pods and allow all namespaces selector", node.entity.ID)
		return allowAllPodsAllNS
	}

	var peers []*storage.NetworkPolicyPeer

	for srcNode := range node.incoming {
		if srcNode.deployment == nil || isSystemDeployment(srcNode.deployment) {
			continue
		}
		peer := &storage.NetworkPolicyPeer{
			PodSelector: labelSelectorForDeployment(srcNode.deployment),
		}
		// If peer namespace is not visible, this will generate 'allow all namespaces' selector
		if node.deployment.Namespace != srcNode.deployment.Namespace {
			if _, visible := namespacesByName[srcNode.deployment.Namespace]; !visible {
				log.Infof("insufficient permissions to peer namespace(s) of node %s; generating allow all namespaces selector", node.entity.ID)
			}
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
