package graph

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/labels"
	"github.com/stackrox/rox/pkg/set"
)

// DeploymentPolicyData groups the network policy relationships and internet access flag together for output.
type DeploymentPolicyData struct {
	appliedIngress set.StringSet
	appliedEgress  set.StringSet
	matchedIngress set.StringSet
	matchedEgress  set.StringSet
	internetAccess bool
}

// MatchDeploymentToPolicies takes in a deployment and a set of policies, and returns a struct that describes which
// of the input policies affect the deployment, and whether or not the deployment is able to access the internet as
// a result.
func MatchDeploymentToPolicies(namespace *storage.NamespaceMetadata, deployments *storage.Deployment, networkPolicies []*storage.NetworkPolicy) *DeploymentPolicyData {
	dpd := &DeploymentPolicyData{
		appliedIngress: set.NewStringSet(),
		appliedEgress:  set.NewStringSet(),
		matchedIngress: set.NewStringSet(),
		matchedEgress:  set.NewStringSet(),
	}
	for _, n := range networkPolicies {
		if n.GetSpec() == nil {
			continue
		}
		if ingressNetworkPolicySelectorAppliesToDeployment(deployments, n) {
			dpd.appliedIngress.Add(n.GetId())
		}
		if doesIngressNetworkPolicyRuleMatchDeployment(namespace, deployments, n) {
			dpd.matchedIngress.Add(n.GetId())
		}
		if applies, internetConnection := egressNetworkPolicySelectorAppliesToDeployment(deployments, n); applies {
			dpd.appliedEgress.Add(n.GetId())
			if internetConnection {
				dpd.internetAccess = true
			}
		}
		if doesEgressNetworkPolicyRuleMatchDeployment(namespace, deployments, n) {
			dpd.matchedEgress.Add(n.GetId())
		}
	}
	return dpd
}

func egressNetworkPolicySelectorAppliesToDeployment(d *storage.Deployment, np *storage.NetworkPolicy) (bool, bool) {
	spec := np.GetSpec()
	// If no egress rules are defined, then it doesn't apply
	if applies := hasEgress(spec.GetPolicyTypes()); !applies {
		return false, false
	}
	// Check if the src matches the pod selector and deployment then the egress rules actually apply to that deployment
	if !doesPodLabelsMatchLabel(d, spec.GetPodSelector()) || d.GetNamespace() != np.GetNamespace() {
		return false, false
	}

	// If there is a rule with an IPBlock that is not nil, then we can assume that they have some sort of internet access
	// This isn't exactly full proof, but probably a pretty decent indicator
	for _, rule := range spec.GetEgress() {
		// An empty list of peers means `allow all`.
		if len(rule.GetTo()) == 0 {
			return true, true
		}
		for _, to := range rule.GetTo() {
			if to.IpBlock != nil {
				return true, true
			}
		}
	}
	return true, false
}

func ingressNetworkPolicySelectorAppliesToDeployment(d *storage.Deployment, np *storage.NetworkPolicy) bool {
	spec := np.GetSpec()
	if !hasIngress(spec.GetPolicyTypes()) {
		return false
	}
	// Check if the src matches the pod selector and deployment then the egress rules actually apply to that deployment
	if !doesPodLabelsMatchLabel(d, spec.GetPodSelector()) || d.GetNamespace() != np.GetNamespace() {
		return false
	}
	return true
}

func doesEgressNetworkPolicyRuleMatchDeployment(namespace *storage.NamespaceMetadata, src *storage.Deployment, np *storage.NetworkPolicy) bool {
	for _, egressRule := range np.GetSpec().GetEgress() {
		if matchPolicyPeers(namespace, src, np.GetNamespace(), egressRule.GetTo()) {
			return true
		}
	}
	return false
}

func doesIngressNetworkPolicyRuleMatchDeployment(namespace *storage.NamespaceMetadata, src *storage.Deployment, np *storage.NetworkPolicy) bool {
	for _, ingressRule := range np.GetSpec().GetIngress() {
		if matchPolicyPeers(namespace, src, np.GetNamespace(), ingressRule.GetFrom()) {
			return true
		}
	}
	return false
}

func matchPolicyPeers(namespace *storage.NamespaceMetadata, d *storage.Deployment, policyNamespace string, peers []*storage.NetworkPolicyPeer) bool {
	if len(peers) == 0 {
		return true
	}
	for _, p := range peers {
		if matchPolicyPeer(namespace, d, policyNamespace, p) {
			return true
		}
	}
	return false
}

func matchPolicyPeer(namespace *storage.NamespaceMetadata, deployment *storage.Deployment, policyNamespace string, peer *storage.NetworkPolicyPeer) bool {
	if peer.IpBlock != nil {
		log.Debug("IP Block network policy is currently not handled")
		return false
	}

	// If namespace selector is specified, then make sure the namespace matches
	// Other you fall back to the fact that the deployment must be in the policy's namespace
	if peer.GetNamespaceSelector() != nil {
		if !doesNamespaceMatchLabel(namespace, peer.GetNamespaceSelector()) {
			return false
		}
	} else if deployment.GetNamespace() != policyNamespace {
		return false
	}

	if peer.GetPodSelector() != nil {
		return doesPodLabelsMatchLabel(deployment, peer.GetPodSelector())
	}
	return true
}

func doesNamespaceMatchLabel(namespace *storage.NamespaceMetadata, selector *storage.LabelSelector) bool {
	return labels.MatchLabels(selector, namespace.GetLabels())
}

func doesPodLabelsMatchLabel(deployment *storage.Deployment, podSelector *storage.LabelSelector) bool {
	return labels.MatchLabels(podSelector, deployment.GetPodLabels())
}

func hasEgress(types []storage.NetworkPolicyType) bool {
	return hasPolicyType(types, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE)
}

func hasIngress(types []storage.NetworkPolicyType) bool {
	if len(types) == 0 {
		return true
	}
	return hasPolicyType(types, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE)
}

func hasPolicyType(types []storage.NetworkPolicyType, t storage.NetworkPolicyType) bool {
	for _, pType := range types {
		if pType == t {
			return true
		}
	}
	return false
}
