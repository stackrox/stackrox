package generator

import (
	"github.com/stackrox/rox/generated/storage"
)

var allowAllIngress = &storage.NetworkPolicyIngressRule{
	From: []*storage.NetworkPolicyPeer{},
}

var allowAllPodsAllNS = storage.NetworkPolicyIngressRule_builder{
	From: []*storage.NetworkPolicyPeer{
		storage.NetworkPolicyPeer_builder{
			NamespaceSelector: &storage.LabelSelector{},
			PodSelector:       &storage.LabelSelector{},
		}.Build(),
	},
}.Build()

func generateIngressRules(node *node, namespacesByName map[string]*storage.NamespaceMetadata) []*storage.NetworkPolicyIngressRule {
	var rules []*storage.NetworkPolicyIngressRule

	for port := range node.incoming {
		rules = append(rules, generateIngressRule(node, port, namespacesByName))
	}

	return rules
}

func generateIngressRule(node *node, port portDesc, namespacesByName map[string]*storage.NamespaceMetadata) *storage.NetworkPolicyIngressRule {
	rule := &storage.NetworkPolicyIngressRule{}
	rule.SetPorts(port.toNetPolPorts())

	srcs := node.incoming[port]

	if srcs.exposed || srcs.hasInternetPeer() {
		// Generate an "allow all" rule if the node either exposes a port externally, or has incoming
		// traffic from the internet.
		rule.SetFrom(allowAllIngress.GetFrom())
		return rule
	}

	// If any peer deployment is not visible, generate 'allow all pods in all namespaces' selector
	if srcs.hasMaskedPeer() {
		log.Debugf("insufficient permissions to peer deployment(s) of node %s; generating allow all pods and allow all namespaces selector", node.entity.ID)
		rule.SetFrom(allowAllPodsAllNS.GetFrom())
		return rule
	}

	var netPolPeers []*storage.NetworkPolicyPeer

	for srcNode := range srcs.peers {
		if srcNode.deployment == nil {
			continue
		}
		netPolPeer := &storage.NetworkPolicyPeer{}
		netPolPeer.SetPodSelector(labelSelectorForDeployment(srcNode.deployment))
		// If netPolPeer namespace is not visible, this will generate 'allow all namespaces' selector
		if node.deployment.GetNamespace() != srcNode.deployment.GetNamespace() {
			nsInfo, nsVisible := namespacesByName[srcNode.deployment.GetNamespace()]
			if !nsVisible {
				log.Infof("insufficient permissions to netPolPeer namespace(s) of node %s; generating allow all namespaces selector", node.entity.ID)
				// Note that we intentionally continue - nsInfo is nil in this case, which in alignment with the
				// emitted log message results in an "all namespaces" selector being generated.
			}
			netPolPeer.SetNamespaceSelector(labelSelectorForNamespace(nsInfo))
		}

		netPolPeers = append(netPolPeers, netPolPeer)
	}

	rule.SetFrom(netPolPeers)
	return rule
}
