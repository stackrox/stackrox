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

func generateIngressRules(node *node, namespacesByName map[string]*storage.NamespaceMetadata) []*storage.NetworkPolicyIngressRule {
	var rules []*storage.NetworkPolicyIngressRule

	for port := range node.incoming {
		rules = append(rules, generateIngressRule(node, port, namespacesByName))
	}

	return rules
}

func generateIngressRule(node *node, port portDesc, namespacesByName map[string]*storage.NamespaceMetadata) *storage.NetworkPolicyIngressRule {
	rule := &storage.NetworkPolicyIngressRule{
		Ports: port.toNetPolPorts(),
	}

	srcs := node.incoming[port]

	if srcs.exposed || srcs.hasInternetPeer() {
		// Generate an "allow all" rule if the node either exposes a port externally, or has incoming
		// traffic from the internet.
		rule.From = allowAllIngress.GetFrom()
		return rule
	}

	// If any peer deployment is not visible, generate 'allow all pods in all namespaces' selector
	if srcs.hasMaskedPeer() {
		log.Debugf("insufficient permissions to peer deployment(s) of node %s; generating allow all pods and allow all namespaces selector", node.entity.ID)
		rule.From = allowAllPodsAllNS.GetFrom()
		return rule
	}

	var netPolPeers []*storage.NetworkPolicyPeer

	for srcNode := range srcs.peers {
		if srcNode.deployment == nil {
			continue
		}
		netPolPeer := &storage.NetworkPolicyPeer{
			PodSelector: labelSelectorForDeployment(srcNode.deployment),
		}
		// If netPolPeer namespace is not visible, this will generate 'allow all namespaces' selector
		if node.deployment.Namespace != srcNode.deployment.Namespace {
			nsInfo, nsVisible := namespacesByName[srcNode.deployment.Namespace]
			if !nsVisible {
				log.Infof("insufficient permissions to netPolPeer namespace(s) of node %s; generating allow all namespaces selector", node.entity.ID)
				// Note that we intentionally continue - nsInfo is nil in this case, which in alignment with the
				// emitted log message results in an "all namespaces" selector being generated.
			}
			netPolPeer.NamespaceSelector = labelSelectorForNamespace(nsInfo)
		}

		netPolPeers = append(netPolPeers, netPolPeer)
	}

	rule.From = netPolPeers
	return rule
}
