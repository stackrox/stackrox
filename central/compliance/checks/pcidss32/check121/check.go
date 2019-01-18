package check121

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

const checkID = "PCI_DSS_3_2:1_2_1"

func init() {
	framework.MustRegisterNewCheck(
		checkID,
		framework.DeploymentKind,
		[]string{"NetworkGraph", "NetworkPolicies"},
		clusterIsCompliant)
}

func clusterIsCompliant(ctx framework.ComplianceContext) {
	// Map deployments to nodes.
	networkGraph := ctx.Data().NetworkGraph()
	deploymentIDToNodes := make(map[string]*v1.NetworkNode, len(networkGraph.GetNodes()))
	for _, node := range networkGraph.GetNodes() {
		if node.GetEntity().GetType() != storage.NetworkEntityInfo_DEPLOYMENT {
			continue
		}
		deploymentIDToNodes[node.GetEntity().GetId()] = node
	}

	// Use the deployment node map to validate each deployment.
	framework.ForEachDeployment(ctx, func(ctx framework.ComplianceContext, deployment *storage.Deployment) {
		deploymentIsCompliant(ctx, deployment, deploymentIDToNodes)
	})
}

func deploymentIsCompliant(ctx framework.ComplianceContext, deployment *storage.Deployment, deploymentIDToNodes map[string]*v1.NetworkNode) {
	if isKubeSystem(deployment) {
		return
	}

	hasIngress := deploymentHasPolicy(ctx, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE, deploymentIDToNodes, deployment)
	hasEgress := deploymentHasPolicy(ctx, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE, deploymentIDToNodes, deployment)
	usesHostNamespace := deployment.GetHostNetwork()

	if hasIngress && hasEgress && !usesHostNamespace {
		framework.Passf(ctx, "Deployment has both ingress and egress network policies applied to it, and does not use host network namespace")
		return
	}
	if !hasIngress {
		framework.Failf(ctx, "No ingress network policies apply to Deployment %s (%s), hence all ingress connections are allowed", deployment.GetName(), deployment.GetId())
	}
	if !hasEgress {
		framework.Failf(ctx, "No egress network policies apply to Deployment %s (%s), hence all egress connections are allowed", deployment.GetName(), deployment.GetId())
	}
	if usesHostNamespace {
		framework.Failf(ctx, "Deployment %s (%s) uses host network, which allows it to subvert network policies", deployment.GetName(), deployment.GetId())
	}
}

func deploymentHasPolicy(ctx framework.ComplianceContext, policyType storage.NetworkPolicyType, deploymentIDToNodes map[string]*v1.NetworkNode, deployment *storage.Deployment) bool {
	for _, policyID := range deploymentIDToNodes[deployment.GetId()].GetPolicyIds() {
		policy := ctx.Data().NetworkPolicies()[policyID]
		if policy == nil || !policyIsOfType(policy.GetSpec(), policyType) {
			continue
		}
		return true
	}
	return false
}

func policyIsOfType(spec *storage.NetworkPolicySpec, policyType storage.NetworkPolicyType) bool {
	for _, ty := range spec.GetPolicyTypes() {
		if ty == policyType {
			return true
		}
	}
	return false
}

func isKubeSystem(deployment *storage.Deployment) bool {
	return deployment.GetNamespace() == "kube-system"
}
